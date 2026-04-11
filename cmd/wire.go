//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.
package cmd

import (
	"neko-tool/internal/api"
	apiProviders "neko-tool/internal/api/providers"
	"neko-tool/internal/app"
	"neko-tool/internal/config"
	dsProviders "neko-tool/internal/core/ds/providers"
	internalRepo "neko-tool/internal/repository"
	internalSvc "neko-tool/internal/service"

	"github.com/google/wire"
)

func InitializeApp() *app.ApplicationHolder {
	wire.Build(
		// 配置
		config.NewApplicationConfigManager,

		// 数据源
		dsProviders.NewMultiDataSource,
		dsProviders.GetPrimaryDataSource,

		// Repository → Service → API
		internalRepo.NewServerNodeRepository,
		internalRepo.NewWorkSpaceRepository,
		internalRepo.NewProjectNodeCacheRepository,
		internalRepo.NewProjectSyncLogRepository,
		internalSvc.NewAccessAuthService,
		internalSvc.NewProjectAccessService,
		internalSvc.NewNodeClientService,
		internalSvc.NewProjectSyncLogService,
		internalSvc.NewSiteInfoService,
		internalSvc.NewCompareService,
		internalSvc.NewProjectBrowserCacheService,
		internalSvc.NewServerNodeService,
		internalSvc.NewWorkSpaceService,
		api.NewAuthApi,
		api.NewCompareApi,
		api.NewNodeApi,
		api.NewProjectApi,
		api.NewSiteApi,
		apiProviders.ProvideApis,

		// 应用层
		app.NewAppWebManager,
		app.NewApplicationHolder,
	)
	return nil
}
