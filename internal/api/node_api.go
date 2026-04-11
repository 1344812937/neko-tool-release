package api

import (
	"net/http"
	"strconv"
	"strings"

	"neko-tool/internal/service"
	pkgApi "neko-tool/pkg/api"
	"neko-tool/pkg/common"
	"neko-tool/pkg/until"

	"github.com/gin-gonic/gin"
)

var _ pkgApi.IApi = (*NodeApi)(nil)

var nodeLog = until.Log

type NodeApi struct {
	pkgApi.BaseApi
	nodeService       *service.ServerNodeService
	nodeClientService *service.NodeClientService
}

type CreateNodeRequest struct {
	BaseURL     string `json:"baseUrl" binding:"required"`
	ApiToken    string `json:"apiToken"`
	Description string `json:"description"`
}

type UpdateNodeRequest struct {
	BaseURL     string `json:"baseUrl" binding:"required"`
	ApiToken    string `json:"apiToken"`
	Description string `json:"description"`
	Enabled     *int   `json:"enabled"`
}

type ResolveNodeInfoRequest struct {
	BaseURL  string `json:"baseUrl" binding:"required"`
	ApiToken string `json:"apiToken"`
}

// NewNodeApi 构造远程节点管理 API 控制器。
func NewNodeApi(nodeService *service.ServerNodeService, nodeClientService *service.NodeClientService) *NodeApi {
	return &NodeApi{nodeService: nodeService, nodeClientService: nodeClientService}
}

// Register 注册节点管理与节点探测相关路由。
func (a *NodeApi) Register(router *gin.RouterGroup) {
	router.GET("/nodes", a.List)
	router.POST("/nodes/resolve-info", a.ResolveInfo)
	router.POST("/nodes/refresh", a.Refresh)
	router.POST("/nodes", a.Create)
	router.PUT("/nodes/:id", a.Update)
	router.DELETE("/nodes/:id", a.Delete)
	router.GET("/nodes/:id/ping", a.Ping)
	router.GET("/nodes/:id/projects", a.Projects)
}

// List 查询已配置的节点列表。
func (a *NodeApi) List(c *gin.Context) {
	defer a.DeferPanicHandler(c)
	list, err := a.nodeService.List(c.Request.Context())
	if err != nil {
		nodeLog.Error("查询节点列表失败: ", err)
		c.JSON(http.StatusOK, common.F[any](500, "查询节点列表失败"))
		return
	}
	var data any = list
	c.JSON(http.StatusOK, common.S(&data))
}

// Create 创建新的远程节点配置。
func (a *NodeApi) Create(c *gin.Context) {
	defer a.DeferPanicHandler(c)
	var req CreateNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, common.F[any](400, "参数错误: "+err.Error()))
		return
	}
	nodeInfo, err := a.nodeClientService.FetchNodeInfo(c.Request.Context(), req.BaseURL, req.ApiToken)
	if err != nil {
		c.JSON(http.StatusOK, common.F[any](500, err.Error()))
		return
	}
	entity, err := a.nodeService.CreateNodeWithRemoteName(c.Request.Context(), nodeInfo.Name, req.BaseURL, req.ApiToken, req.Description)
	if err != nil {
		c.JSON(http.StatusOK, common.F[any](500, err.Error()))
		return
	}
	var data any = entity
	c.JSON(http.StatusOK, common.S(&data))
}

// Update 更新指定节点的连接信息与启用状态。
func (a *NodeApi) Update(c *gin.Context) {
	defer a.DeferPanicHandler(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, common.F[any](400, "无效的节点ID"))
		return
	}
	var req UpdateNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, common.F[any](400, "参数错误: "+err.Error()))
		return
	}
	enabled := 1
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	nodeInfo, err := a.nodeClientService.FetchNodeInfo(c.Request.Context(), req.BaseURL, req.ApiToken)
	if err != nil {
		c.JSON(http.StatusOK, common.F[any](500, err.Error()))
		return
	}
	if err := a.nodeService.ApplyRemoteNodeInfo(c.Request.Context(), id, req.BaseURL, req.ApiToken, nodeInfo.Name, req.Description, enabled); err != nil {
		c.JSON(http.StatusOK, common.F[any](500, err.Error()))
		return
	}
	c.JSON(http.StatusOK, common.S[any](nil))
}

// ResolveInfo 根据输入的节点地址和共享令牌读取远端节点信息。
func (a *NodeApi) ResolveInfo(c *gin.Context) {
	defer a.DeferPanicHandler(c)
	var req ResolveNodeInfoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, common.F[any](400, "参数错误: "+err.Error()))
		return
	}
	data, err := a.nodeClientService.FetchNodeInfo(c.Request.Context(), req.BaseURL, req.ApiToken)
	if err != nil {
		c.JSON(http.StatusOK, common.F[any](500, err.Error()))
		return
	}
	var result any = data
	c.JSON(http.StatusOK, common.S(&result))
}

// Refresh 重新探测全部已配置节点，并在远端名称变化时回写数据库。
func (a *NodeApi) Refresh(c *gin.Context) {
	defer a.DeferPanicHandler(c)
	list, err := a.nodeService.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusOK, common.F[any](500, "查询节点列表失败"))
		return
	}
	result := service.NodeRefreshResult{Failed: make([]string, 0)}
	for _, row := range list {
		result.Checked++
		info, fetchErr := a.nodeClientService.FetchNodeInfo(c.Request.Context(), row.BaseURL, row.ApiToken)
		if fetchErr != nil {
			result.Failed = append(result.Failed, row.Name+": "+fetchErr.Error())
			continue
		}
		if strings.TrimSpace(info.Name) == "" || strings.TrimSpace(info.Name) == strings.TrimSpace(row.Name) {
			continue
		}
		if row.Id == nil || *row.Id == 0 {
			result.Failed = append(result.Failed, row.Name+": 节点ID无效")
			continue
		}
		if applyErr := a.nodeService.ApplyRemoteNodeInfo(c.Request.Context(), *row.Id, row.BaseURL, row.ApiToken, info.Name, row.Description, row.Enabled); applyErr != nil {
			result.Failed = append(result.Failed, row.Name+": "+applyErr.Error())
			continue
		}
		result.Updated++
	}
	var data any = result
	c.JSON(http.StatusOK, common.S(&data))
}

// Delete 删除指定的远程节点配置。
func (a *NodeApi) Delete(c *gin.Context) {
	defer a.DeferPanicHandler(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, common.F[any](400, "无效的节点ID"))
		return
	}
	if err := a.nodeService.DeleteNode(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusOK, common.F[any](500, "删除节点失败"))
		return
	}
	c.JSON(http.StatusOK, common.S[any](nil))
}

// Ping 探测远程节点是否可达并返回节点信息。
func (a *NodeApi) Ping(c *gin.Context) {
	defer a.DeferPanicHandler(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, common.F[any](400, "无效的节点ID"))
		return
	}
	data, err := a.nodeClientService.Ping(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusOK, common.F[any](500, err.Error()))
		return
	}
	var result any = data
	c.JSON(http.StatusOK, common.S(&result))
}

// Projects 查询远程节点暴露的项目列表。
func (a *NodeApi) Projects(c *gin.Context) {
	defer a.DeferPanicHandler(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, common.F[any](400, "无效的节点ID"))
		return
	}
	data, err := a.nodeClientService.ListProjects(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusOK, common.F[any](500, err.Error()))
		return
	}
	var result any = data
	c.JSON(http.StatusOK, common.S(&result))
}
