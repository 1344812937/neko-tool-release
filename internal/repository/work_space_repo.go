package repository

import (
	"neko-tool/internal/models"
	"neko-tool/pkg/core/tx"
	"neko-tool/pkg/repository"
)

// WorkSpaceRepository WorkSpace 数据访问层。
type WorkSpaceRepository struct {
	*repository.BaseRepository[models.WorkSpace]
}

// NewWorkSpaceRepository 构造器，完成表结构自动迁移。
func NewWorkSpaceRepository(ds *tx.DataSource) *WorkSpaceRepository {
	repo := &WorkSpaceRepository{
		BaseRepository: repository.NewBaseRepository[models.WorkSpace](ds),
	}
	repo.InitializeRepository()
	return repo
}
