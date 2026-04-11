package app

import (
	"net/http"
	"strings"

	"neko-tool/internal/service"
	"neko-tool/pkg/common"

	"github.com/gin-gonic/gin"
)

func (awm *AppWebManager) userAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestPath := c.Request.URL.Path
		if strings.HasPrefix(requestPath, "/api/internal/") || strings.HasPrefix(requestPath, "/api/auth/") {
			c.Next()
			return
		}
		authKey := strings.TrimSpace(c.GetHeader(service.UserAuthHeader()))
		if authKey == "" {
			authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
			if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
				authKey = strings.TrimSpace(authHeader[7:])
			}
		}
		if awm.authService != nil && awm.authService.VerifyAuthKey(authKey) {
			c.Next()
			return
		}
		c.AbortWithStatusJSON(http.StatusUnauthorized, common.F[any](401, "未认证或认证已过期，请重新登录"))
	}
}
