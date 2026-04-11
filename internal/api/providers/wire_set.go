package providers

import (
	"neko-tool/internal/api"
	pkgApi "neko-tool/pkg/api"
)

// ProvideApis 聚合所有业务 API 为 IApi 切片，供 AppWebManager 使用
func ProvideApis(authApi *api.AuthApi, projectApi *api.ProjectApi, nodeApi *api.NodeApi, compareApi *api.CompareApi, siteApi *api.SiteApi) []pkgApi.IApi {
	return []pkgApi.IApi{authApi, projectApi, nodeApi, compareApi, siteApi}
}
