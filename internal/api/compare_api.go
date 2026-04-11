package api

import (
	"net"
	"net/http"
	"strings"
	"time"

	"neko-tool/internal/config"
	"neko-tool/internal/service"
	pkgApi "neko-tool/pkg/api"
	"neko-tool/pkg/common"
	"neko-tool/pkg/until"

	"github.com/gin-gonic/gin"
)

var _ pkgApi.IApi = (*CompareApi)(nil)

var compareApiLog = until.Log

type CompareApi struct {
	pkgApi.BaseApi
	compareService      *service.CompareService
	browserCacheService *service.ProjectBrowserCacheService
	configManager       *config.ApplicationConfigManager
}

// NewCompareApi 构造对比相关 API 控制器。
func NewCompareApi(compareService *service.CompareService, browserCacheService *service.ProjectBrowserCacheService, configManager *config.ApplicationConfigManager) *CompareApi {
	return &CompareApi{compareService: compareService, browserCacheService: browserCacheService, configManager: configManager}
}

// Register 注册项目对比、浏览与节点内部协同相关路由。
func (a *CompareApi) Register(router *gin.RouterGroup) {
	router.POST("/compare/projects", a.Compare)
	router.POST("/compare/browser", a.BrowseProject)
	router.POST("/compare/browser/refresh", a.RefreshProjectBrowser)
	router.POST("/compare/browser/file", a.ReadProjectFile)
	router.POST("/compare/browser/delete", a.DeleteProjectPath)
	router.POST("/compare/browser/file/logs", a.ListProjectFileLogs)
	router.POST("/compare/browser/file/log-detail", a.ProjectFileLogDetail)
	router.POST("/compare/file-diff", a.FileDiff)
	router.POST("/compare/sync", a.Sync)
	router.GET("/internal/node-info", a.NodeInfo)
	router.GET("/internal/projects", a.InternalProjects)
	router.POST("/internal/manifest", a.InternalManifest)
	router.POST("/internal/file", a.InternalFile)
	router.POST("/internal/write-file", a.InternalWriteFile)
}

// Compare 执行两个项目目录的差异对比。
func (a *CompareApi) Compare(c *gin.Context) {
	defer a.DeferPanicHandler(c)
	var req service.CompareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, common.F[any](400, "参数错误: "+err.Error()))
		return
	}
	result, err := a.compareService.Compare(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusOK, common.F[any](500, err.Error()))
		return
	}
	var data any = result
	c.JSON(http.StatusOK, common.S(&data))
}

// BrowseProject 获取项目浏览清单，本机项目优先走缓存浏览服务。
func (a *CompareApi) BrowseProject(c *gin.Context) {
	defer a.DeferPanicHandler(c)
	var req service.ProjectBrowseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, common.F[any](400, "参数错误: "+err.Error()))
		return
	}
	var (
		result service.ManifestResult
		err    error
	)
	if req.NodeId == 0 {
		result, err = a.browserCacheService.BrowseProject(c.Request.Context(), req)
	} else {
		result, err = a.compareService.BrowseProject(c.Request.Context(), req)
	}
	if err != nil {
		c.JSON(http.StatusOK, common.F[any](500, err.Error()))
		return
	}
	var data any = result
	c.JSON(http.StatusOK, common.S(&data))
}

// RefreshProjectBrowser 重新扫描本机项目目录并刷新缓存后返回浏览清单。
func (a *CompareApi) RefreshProjectBrowser(c *gin.Context) {
	defer a.DeferPanicHandler(c)
	var req service.ProjectBrowseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, common.F[any](400, "参数错误: "+err.Error()))
		return
	}
	if req.NodeId != 0 {
		c.JSON(http.StatusOK, common.F[any](400, "仅支持刷新本机项目缓存"))
		return
	}
	result, err := a.browserCacheService.RefreshProject(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusOK, common.F[any](500, err.Error()))
		return
	}
	var data any = result
	c.JSON(http.StatusOK, common.S(&data))
}

// ReadProjectFile 读取项目中的单个文件内容或文件状态。
func (a *CompareApi) ReadProjectFile(c *gin.Context) {
	defer a.DeferPanicHandler(c)
	var req service.ProjectBrowseFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, common.F[any](400, "参数错误: "+err.Error()))
		return
	}
	var (
		result service.FileSide
		err    error
	)
	if req.NodeId == 0 {
		result, err = a.browserCacheService.ReadProjectFile(c.Request.Context(), req)
	} else {
		result, err = a.compareService.ReadProjectFile(c.Request.Context(), req)
	}
	if err != nil {
		c.JSON(http.StatusOK, common.F[any](500, err.Error()))
		return
	}
	var data any = result
	c.JSON(http.StatusOK, common.S(&data))
}

// DeleteProjectPath 删除项目浏览页中选中的文件或目录。
func (a *CompareApi) DeleteProjectPath(c *gin.Context) {
	defer a.DeferPanicHandler(c)
	var req service.ProjectBrowseDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, common.F[any](400, "参数错误: "+err.Error()))
		return
	}
	if err := a.browserCacheService.DeleteProjectPath(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusOK, common.F[any](500, err.Error()))
		return
	}
	var data any = map[string]bool{"success": true}
	c.JSON(http.StatusOK, common.S(&data))
}

// ListProjectFileLogs 查询本机项目中指定文件的修改日志列表。
func (a *CompareApi) ListProjectFileLogs(c *gin.Context) {
	defer a.DeferPanicHandler(c)
	var req service.ProjectSyncLogListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, common.F[any](400, "参数错误: "+err.Error()))
		return
	}
	rows, err := a.compareService.ListProjectFileLogs(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusOK, common.F[any](500, err.Error()))
		return
	}
	var data any = rows
	c.JSON(http.StatusOK, common.S(&data))
}

// ProjectFileLogDetail 查询单条文件修改日志的详细内容。
func (a *CompareApi) ProjectFileLogDetail(c *gin.Context) {
	defer a.DeferPanicHandler(c)
	var req service.ProjectSyncLogDetailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, common.F[any](400, "参数错误: "+err.Error()))
		return
	}
	detail, err := a.compareService.GetProjectFileLogDetail(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusOK, common.F[any](500, err.Error()))
		return
	}
	var data any = detail
	c.JSON(http.StatusOK, common.S(&data))
}

// FileDiff 获取指定文件在两个项目之间的文本差异。
func (a *CompareApi) FileDiff(c *gin.Context) {
	defer a.DeferPanicHandler(c)
	var req service.FileDiffRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, common.F[any](400, "参数错误: "+err.Error()))
		return
	}
	result, err := a.compareService.FileDiff(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusOK, common.F[any](500, err.Error()))
		return
	}
	var data any = result
	c.JSON(http.StatusOK, common.S(&data))
}

// Sync 将来源项目中的文件内容同步到目标项目。
func (a *CompareApi) Sync(c *gin.Context) {
	defer a.DeferPanicHandler(c)
	var req service.SyncRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, common.F[any](400, "参数错误: "+err.Error()))
		return
	}
	result, err := a.compareService.Sync(c.Request.Context(), req, resolveOperatorIP(c), resolveExecutorNodeAddress(c, a.configManager))
	if err != nil {
		c.JSON(http.StatusOK, common.F[any](500, err.Error()))
		return
	}
	var data any = result
	c.JSON(http.StatusOK, common.S(&data))
}

// NodeInfo 返回当前节点的基础信息，供远程节点探活使用。
func (a *CompareApi) NodeInfo(c *gin.Context) {
	defer a.DeferPanicHandler(c)
	if !a.authorizeInternal(c) {
		return
	}
	data := a.compareService.NodeInfo()
	c.JSON(http.StatusOK, common.S(&data))
}

// InternalProjects 返回当前节点可访问的本地项目列表。
func (a *CompareApi) InternalProjects(c *gin.Context) {
	defer a.DeferPanicHandler(c)
	if !a.authorizeInternal(c) {
		return
	}
	projects, err := a.compareService.ListLocalProjects(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusOK, common.F[any](500, err.Error()))
		return
	}
	var data any = projects
	c.JSON(http.StatusOK, common.S(&data))
}

// InternalManifest 为远程协同节点构建本地目录清单。
func (a *CompareApi) InternalManifest(c *gin.Context) {
	defer a.DeferPanicHandler(c)
	if !a.authorizeInternal(c) {
		return
	}
	var req service.InternalManifestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, common.F[any](400, "参数错误: "+err.Error()))
		return
	}
	result, err := a.compareService.BuildLocalManifest(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusOK, common.F[any](500, err.Error()))
		return
	}
	var data any = result
	c.JSON(http.StatusOK, common.S(&data))
}

// InternalFile 为远程协同节点读取本地文件内容。
func (a *CompareApi) InternalFile(c *gin.Context) {
	defer a.DeferPanicHandler(c)
	if !a.authorizeInternal(c) {
		return
	}
	var req service.InternalFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, common.F[any](400, "参数错误: "+err.Error()))
		return
	}
	result, err := a.compareService.ReadLocalFile(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusOK, common.F[any](500, err.Error()))
		return
	}
	var data any = result
	c.JSON(http.StatusOK, common.S(&data))
}

// InternalWriteFile 为远程协同节点写入本地文件内容。
func (a *CompareApi) InternalWriteFile(c *gin.Context) {
	defer a.DeferPanicHandler(c)
	if !a.authorizeInternal(c) {
		return
	}
	var req service.InternalWriteFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, common.F[any](400, "参数错误: "+err.Error()))
		return
	}
	if err := a.compareService.WriteLocalFile(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusOK, common.F[any](500, err.Error()))
		return
	}
	c.JSON(http.StatusOK, common.S[any](nil))
}

func (a *CompareApi) authorizeInternal(c *gin.Context) bool {
	sharedToken := strings.TrimSpace(a.configManager.GetConfig().NodeConfig.SharedToken)
	requestSignature := strings.TrimSpace(c.GetHeader(service.NodeTokenHeader()))
	requestTimestamp := strings.TrimSpace(c.GetHeader(service.NodeTimestampHeader()))
	if sharedToken == "" && requestSignature == "" && requestTimestamp == "" {
		return true
	}
	if sharedToken != "" && requestSignature != "" && requestTimestamp != "" && service.VerifyNodeAuth(sharedToken, requestSignature, requestTimestamp, time.Now()) {
		return true
	}
	if sharedToken == "" {
		compareApiLog.Warn("收到携带节点签名的内部请求，但当前节点未配置 shared_token")
	} else if requestSignature == "" || requestTimestamp == "" {
		compareApiLog.Warn("收到缺少签名或时间戳的内部节点请求")
	} else {
		compareApiLog.Warnf("收到未授权的内部节点请求，要求时间误差不超过 %s", service.NodeAuthTTL())
	}
	c.JSON(http.StatusOK, common.F[any](401, "内部节点鉴权失败"))
	return false
}

func resolveOperatorIP(c *gin.Context) string {
	candidates := []string{
		firstHeaderIP(c.GetHeader("X-Forwarded-For")),
		normalizeIPToken(c.GetHeader("X-Real-IP")),
		forwardedHeaderIP(c.GetHeader("Forwarded")),
		normalizeIPToken(c.ClientIP()),
	}
	for _, candidate := range candidates {
		if candidate != "" {
			return candidate
		}
	}
	return ""
}

func resolveExecutorNodeAddress(c *gin.Context, configManager *config.ApplicationConfigManager) string {
	configuredAddress := ""
	if configManager != nil {
		configuredAddress = configManager.GetConfig().NodeConfig.WorkstationAddress
	}
	return service.ResolveWorkstationAddress(configuredAddress, c.Request.Host)
}

func firstHeaderIP(value string) string {
	for _, part := range strings.Split(value, ",") {
		if ip := normalizeIPToken(part); ip != "" {
			return ip
		}
	}
	return ""
}

func forwardedHeaderIP(value string) string {
	for _, segment := range strings.Split(value, ",") {
		for _, pair := range strings.Split(segment, ";") {
			trimmedPair := strings.TrimSpace(pair)
			if !strings.HasPrefix(strings.ToLower(trimmedPair), "for=") {
				continue
			}
			if ip := normalizeIPToken(strings.TrimSpace(trimmedPair[4:])); ip != "" {
				return ip
			}
		}
	}
	return ""
}

func normalizeIPToken(value string) string {
	trimmed := strings.Trim(strings.TrimSpace(value), "\"")
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, "[") {
		if endIndex := strings.Index(trimmed, "]"); endIndex > 0 {
			trimmed = trimmed[1:endIndex]
		}
	}
	if host, _, err := net.SplitHostPort(trimmed); err == nil {
		trimmed = host
	} else {
		colonIndex := strings.LastIndex(trimmed, ":")
		if colonIndex > 0 && strings.Count(trimmed, ":") == 1 && strings.Contains(trimmed, ".") {
			trimmed = trimmed[:colonIndex]
		}
	}
	parsedIP := net.ParseIP(trimmed)
	if parsedIP == nil {
		return ""
	}
	return parsedIP.String()
}
