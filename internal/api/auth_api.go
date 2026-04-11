package api

import (
	"net/http"

	"neko-tool/internal/service"
	pkgApi "neko-tool/pkg/api"
	"neko-tool/pkg/common"

	"github.com/gin-gonic/gin"
)

var _ pkgApi.IApi = (*AuthApi)(nil)

type AuthApi struct {
	pkgApi.BaseApi
	authService *service.AccessAuthService
}

type AuthLoginRequest struct {
	AccessToken string `json:"accessToken" binding:"required"`
}

func NewAuthApi(authService *service.AccessAuthService) *AuthApi {
	return &AuthApi{authService: authService}
}

func (a *AuthApi) Register(router *gin.RouterGroup) {
	router.POST("/auth/login", a.Login)
}

func (a *AuthApi) Login(c *gin.Context) {
	defer a.DeferPanicHandler(c)
	var req AuthLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, common.F[any](400, "参数错误: "+err.Error()))
		return
	}
	result, err := a.authService.IssueAuthKey(req.AccessToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, common.F[any](401, err.Error()))
		return
	}
	var data any = result
	c.JSON(http.StatusOK, common.S(&data))
}
