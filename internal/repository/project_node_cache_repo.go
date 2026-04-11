package repository

import (
	"strings"
	"time"

	"neko-tool/internal/models"
	"neko-tool/pkg/core/tx"
	pkgModels "neko-tool/pkg/models"
	"neko-tool/pkg/repository"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ProjectNodeCacheRepository struct {
	*repository.BaseRepository[models.ProjectNodeCache]
}

func NewProjectNodeCacheRepository(ds *tx.DataSource) *ProjectNodeCacheRepository {
	repo := &ProjectNodeCacheRepository{
		BaseRepository: repository.NewBaseRepository[models.ProjectNodeCache](ds),
	}
	repo.InitializeRepository()
	return repo
}

func (r *ProjectNodeCacheRepository) WithScope(scope tx.ITransactionScope) *ProjectNodeCacheRepository {
	return &ProjectNodeCacheRepository{BaseRepository: r.BaseRepository.WithScope(scope).(*repository.BaseRepository[models.ProjectNodeCache])}
}

func (r *ProjectNodeCacheRepository) WithDb(db *gorm.DB) *ProjectNodeCacheRepository {
	return &ProjectNodeCacheRepository{BaseRepository: r.BaseRepository.WithDb(db).(*repository.BaseRepository[models.ProjectNodeCache])}
}

func (r *ProjectNodeCacheRepository) GetByProjectAndPath(projectId uint64, relativePath string) (models.ProjectNodeCache, error) {
	var node models.ProjectNodeCache
	err := r.Db().Model(&models.ProjectNodeCache{}).
		Where("`project_id` = ? AND `relative_path` = ? AND `Valid` = ?", projectId, relativePath, 1).
		First(&node).Error
	return node, err
}

func (r *ProjectNodeCacheRepository) ListSubTree(projectId uint64, basePath string, maxDepth int) ([]models.ProjectNodeCache, error) {
	query := r.Db().Model(&models.ProjectNodeCache{}).
		Where("`project_id` = ? AND `Valid` = ?", projectId, 1)
	if strings.TrimSpace(basePath) == "" {
		query = query.Where("`depth` <= ?", maxDepth)
	} else {
		baseDepth := pathDepth(basePath)
		prefix := basePath + "/%"
		query = query.Where("(`relative_path` = ? OR `relative_path` LIKE ?) AND `depth` <= ?", basePath, prefix, baseDepth+maxDepth)
	}
	query = query.Order("depth asc, entry_type asc, relative_path asc")
	var nodes []models.ProjectNodeCache
	err := query.Find(&nodes).Error
	return nodes, err
}

func (r *ProjectNodeCacheRepository) ListProjectFiles(projectId uint64) ([]models.ProjectNodeCache, error) {
	var nodes []models.ProjectNodeCache
	err := r.Db().Model(&models.ProjectNodeCache{}).
		Where("`project_id` = ? AND `entry_type` = ? AND `Valid` = ?", projectId, "file", 1).
		Order("relative_path asc").
		Find(&nodes).Error
	return nodes, err
}

func (r *ProjectNodeCacheRepository) ReplaceProjectNodes(projectId uint64, nodes []models.ProjectNodeCache, scannedAt time.Time) error {
	if err := r.Db().Model(&models.ProjectNodeCache{}).
		Where("`project_id` = ? AND `Valid` = ?", projectId, 1).
		Updates(map[string]any{"disk_deleted": true, "last_scan_at": scannedAt}).Error; err != nil {
		return err
	}
	if len(nodes) == 0 {
		return nil
	}
	for index := range nodes {
		if nodes[index].BaseModel == nil {
			nodes[index].BaseModel = &pkgModels.BaseModel{}
		}
		nodes[index].ProjectId = projectId
		nodes[index].Valid = 1
	}
	return r.UpsertBatch(nodes, clause.OnConflict{
		Columns: []clause.Column{{Name: "project_id"}, {Name: "relative_path"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"parent_path",
			"name",
			"entry_type",
			"hash",
			"size",
			"entry_mod_time",
			"depth",
			"has_children",
			"disk_deleted",
			"last_scan_at",
			"valid",
		}),
	})
}

func (r *ProjectNodeCacheRepository) MarkProjectDeleted(projectId uint64, scannedAt time.Time) error {
	return r.Db().Model(&models.ProjectNodeCache{}).
		Where("`project_id` = ? AND `Valid` = ?", projectId, 1).
		Updates(map[string]any{"disk_deleted": true, "last_scan_at": scannedAt}).Error
}

func (r *ProjectNodeCacheRepository) MarkPathDeleted(projectId uint64, relativePath string, scannedAt time.Time) error {
	query := r.Db().Model(&models.ProjectNodeCache{}).
		Where("`project_id` = ? AND `Valid` = ?", projectId, 1)
	if strings.TrimSpace(relativePath) == "" {
		return query.Updates(map[string]any{"disk_deleted": true, "last_scan_at": scannedAt}).Error
	}
	prefix := relativePath + "/%"
	return query.Where("`relative_path` = ? OR `relative_path` LIKE ?", relativePath, prefix).
		Updates(map[string]any{"disk_deleted": true, "last_scan_at": scannedAt}).Error
}

func pathDepth(relativePath string) int {
	trimmed := strings.Trim(relativePath, "/")
	if trimmed == "" {
		return 0
	}
	return len(strings.Split(trimmed, "/"))
}
