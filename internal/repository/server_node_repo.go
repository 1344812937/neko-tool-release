package repository

import (
	"neko-tool/internal/models"
	"neko-tool/pkg/core/tx"
	"neko-tool/pkg/repository"
)

type ServerNodeRepository struct {
	*repository.BaseRepository[models.ServerNode]
}

func NewServerNodeRepository(ds *tx.DataSource) *ServerNodeRepository {
	repo := &ServerNodeRepository{
		BaseRepository: repository.NewBaseRepository[models.ServerNode](ds),
	}
	repo.InitializeRepository()
	return repo
}
