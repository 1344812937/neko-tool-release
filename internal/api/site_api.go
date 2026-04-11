package api

import (
	"net/http"

	"neko-tool/internal/service"
	pkgApi "neko-tool/pkg/api"
	"neko-tool/pkg/common"

	"github.com/gin-gonic/gin"
)

var _ pkgApi.IApi = (*SiteApi)(nil)

type SiteApi struct {
	pkgApi.BaseApi
	siteInfoService *service.SiteInfoService
	compareService  *service.CompareService
}

func NewSiteApi(siteInfoService *service.SiteInfoService, compareService *service.CompareService) *SiteApi {
	return &SiteApi{siteInfoService: siteInfoService, compareService: compareService}
}

func (a *SiteApi) Register(router *gin.RouterGroup) {
	router.GET("/site/info", a.Info)
	router.POST("/site/logs", a.ListLogs)
	router.POST("/site/log-detail", a.LogDetail)
	router.POST("/site/logs/cleanup", a.CleanupLogs)
}

func (a *SiteApi) Info(c *gin.Context) {
	defer a.DeferPanicHandler(c)
	data := a.siteInfoService.GetSiteInfo()
	var result any = data
	c.JSON(http.StatusOK, common.S(&result))
}

func (a *SiteApi) ListLogs(c *gin.Context) {
	defer a.DeferPanicHandler(c)
	var req service.SiteProjectSyncLogListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, common.F[any](400, "参数错误: "+err.Error()))
		return
	}
	result, err := a.compareService.ListSiteLogs(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusOK, common.F[any](500, err.Error()))
		return
	}
	var data any = result
	c.JSON(http.StatusOK, common.S(&data))
}

func (a *SiteApi) LogDetail(c *gin.Context) {
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

func (a *SiteApi) CleanupLogs(c *gin.Context) {
	defer a.DeferPanicHandler(c)
	result, err := a.compareService.CleanupSiteLogs(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusOK, common.F[any](500, err.Error()))
		return
	}
	var data any = result
	c.JSON(http.StatusOK, common.S(&data))
}
