package api

import (
	"net/http"
	"strconv"

	"neko-tool/internal/models"
	"neko-tool/internal/service"
	pkgApi "neko-tool/pkg/api"
	"neko-tool/pkg/common"
	"neko-tool/pkg/until"

	"github.com/gin-gonic/gin"
	"github.com/ncruces/zenity"
)

// 编译期接口检查
var _ pkgApi.IApi = (*ProjectApi)(nil)

var projectLog = until.Log

type ProjectApi struct {
	pkgApi.BaseApi
	workSpaceService *service.WorkSpaceService
	projectAccess    *service.ProjectAccessService
	browserCache     *service.ProjectBrowserCacheService
}

// NewProjectApi 构造器，由 Wire 注入依赖。
func NewProjectApi(svc *service.WorkSpaceService, projectAccess *service.ProjectAccessService, browserCache *service.ProjectBrowserCacheService) *ProjectApi {
	return &ProjectApi{workSpaceService: svc, projectAccess: projectAccess, browserCache: browserCache}

}

type CreateProjectResponse struct {
	Project  models.WorkSpace `json:"project"`
	Restored bool             `json:"restored"`
}

// Register 注册路由
func (p *ProjectApi) Register(router *gin.RouterGroup) {
	router.GET("/projects", p.List)
	router.POST("/projects", p.Create)
	router.PUT("/projects/:id", p.Rename)
	router.DELETE("/projects/:id", p.Delete)
	router.GET("/select-directory", p.SelectDirectory)
	router.GET("/project-access/capabilities", p.Capabilities)
	router.GET("/project-access/directories", p.ListDirectories)
}

type CreateProjectRequest struct {
	Name string `json:"name" binding:"required"`
	Code string `json:"code"`
	Path string `json:"path" binding:"required"`
}

type RenameProjectRequest struct {
	Name string `json:"name" binding:"required"`
}

// List 分页查询项目列表。
func (p *ProjectApi) List(c *gin.Context) {
	defer p.DeferPanicHandler(c)

	pageNo, _ := strconv.Atoi(c.DefaultQuery("pageNo", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	if pageNo < 1 {
		pageNo = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	page := common.Page[models.WorkSpace]{
		PageNo:   pageNo,
		PageSize: pageSize,
	}

	result, err := p.workSpaceService.Page(c.Request.Context(), nil, page)
	if err != nil {
		projectLog.Error("查询项目列表失败: ", err)
		c.JSON(http.StatusOK, common.F[any](500, "查询项目列表失败"))
		return
	}
	var data any = result
	c.JSON(http.StatusOK, common.S(&data))
}

// Create 创建新的本地项目记录。
func (p *ProjectApi) Create(c *gin.Context) {
	defer p.DeferPanicHandler(c)

	var req CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, common.F[any](400, "参数错误: "+err.Error()))
		return
	}
	validatedPath, err := p.projectAccess.ValidateProjectPathForCreate(req.Path, c.ClientIP())
	if err != nil {
		c.JSON(http.StatusOK, common.F[any](400, err.Error()))
		return
	}
	req.Path = validatedPath

	result, err := p.workSpaceService.CreateProject(c.Request.Context(), req.Name, req.Code, req.Path)
	if err != nil {
		c.JSON(http.StatusOK, common.F[any](500, err.Error()))
		return
	}
	if result.Restored {
		projectId := uint64(0)
		if result.Project.Id != nil {
			projectId = *result.Project.Id
		}
		if _, refreshErr := p.browserCache.RefreshProject(c.Request.Context(), service.ProjectBrowseRequest{
			NodeId:    0,
			ProjectId: projectId,
			BasePath:  "",
			Depth:     3,
		}); refreshErr != nil {
			c.JSON(http.StatusOK, common.F[any](500, "恢复项目后重新扫描失败: "+refreshErr.Error()))
			return
		}
		refreshedProject, getErr := p.workSpaceService.GetProject(c.Request.Context(), projectId)
		if getErr == nil {
			result.Project = refreshedProject
		}
	}
	var data any = CreateProjectResponse{Project: result.Project, Restored: result.Restored}
	c.JSON(http.StatusOK, common.S(&data))
}

// Rename 修改指定项目的展示名称。
func (p *ProjectApi) Rename(c *gin.Context) {
	defer p.DeferPanicHandler(c)

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, common.F[any](400, "无效的ID"))
		return
	}

	var req RenameProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, common.F[any](400, "参数错误: "+err.Error()))
		return
	}

	if err := p.workSpaceService.RenameProject(c.Request.Context(), id, req.Name); err != nil {
		c.JSON(http.StatusOK, common.F[any](500, "修改项目名称失败"))
		return
	}
	c.JSON(http.StatusOK, common.S[any](nil))
}

// Delete 软删除指定项目。
func (p *ProjectApi) Delete(c *gin.Context) {
	defer p.DeferPanicHandler(c)

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, common.F[any](400, "无效的ID"))
		return
	}

	if err := p.workSpaceService.DeleteProject(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusOK, common.F[any](500, "删除项目失败"))
		return
	}
	c.JSON(http.StatusOK, common.S[any](nil))
}

// SelectDirectory 调用系统目录选择器选择项目根目录。
func (p *ProjectApi) SelectDirectory(c *gin.Context) {
	defer p.DeferPanicHandler(c)
	if !p.projectAccess.CanUseNativePicker(c.ClientIP()) {
		c.JSON(http.StatusOK, common.F[string](403, "当前访问不支持系统目录选择器"))
		return
	}

	path, err := zenity.SelectFile(
		zenity.Directory(),
		zenity.Title("选择项目目录"),
	)
	if err != nil {
		if err == zenity.ErrCanceled {
			c.JSON(http.StatusOK, common.F[string](400, "用户取消选择"))
			return
		}
		projectLog.Error("打开目录选择器失败: ", err)
		c.JSON(http.StatusOK, common.F[string](500, "打开目录选择器失败"))
		return
	}
	c.JSON(http.StatusOK, common.S(&path))
}

// Capabilities 返回当前访问来源可用的目录访问能力。
func (p *ProjectApi) Capabilities(c *gin.Context) {
	defer p.DeferPanicHandler(c)
	data := p.projectAccess.GetCapabilities(c.ClientIP())
	c.JSON(http.StatusOK, common.S(&data))
}

// ListDirectories 返回目录选择器使用的下级目录列表。
func (p *ProjectApi) ListDirectories(c *gin.Context) {
	defer p.DeferPanicHandler(c)
	entries, err := p.projectAccess.ListDirectories(c.Query("path"), c.ClientIP())
	if err != nil {
		c.JSON(http.StatusOK, common.F[any](400, err.Error()))
		return
	}
	var data any = entries
	c.JSON(http.StatusOK, common.S(&data))
}
