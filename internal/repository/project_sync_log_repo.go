package repository

import (
	"errors"
	"strings"

	"neko-tool/internal/models"
	"neko-tool/pkg/core/tx"
	"neko-tool/pkg/repository"

	"gorm.io/gorm"
)

const latestProjectSyncLogPathBatchSize = 200

// ProjectSyncLogRepository 文件同步详细日志数据访问层。
type ProjectSyncLogRepository struct {
	*repository.BaseRepository[models.ProjectSyncLog]
}

// NewProjectSyncLogRepository 构造器，完成表结构自动迁移。
func NewProjectSyncLogRepository(ds *tx.DataSource) *ProjectSyncLogRepository {
	repo := &ProjectSyncLogRepository{
		BaseRepository: repository.NewBaseRepository[models.ProjectSyncLog](ds),
	}
	repo.InitializeRepository()
	return repo
}

func (r *ProjectSyncLogRepository) WithScope(scope tx.ITransactionScope) repository.IExecutor[models.ProjectSyncLog] {
	return r.BaseRepository.WithScope(scope)
}

func (r *ProjectSyncLogRepository) WithDb(db *gorm.DB) repository.IExecutor[models.ProjectSyncLog] {
	return r.BaseRepository.WithDb(db)
}

func (r *ProjectSyncLogRepository) Scoped(scope tx.ITransactionScope) *ProjectSyncLogRepository {
	return &ProjectSyncLogRepository{BaseRepository: r.BaseRepository.WithScope(scope).(*repository.BaseRepository[models.ProjectSyncLog])}
}

func (r *ProjectSyncLogRepository) ListByTargetProjectAndPath(projectId uint64, relativePath string, limit int) ([]models.ProjectSyncLog, error) {
	query := r.Db().Model(&models.ProjectSyncLog{}).
		Where("`target_project_id` = ? AND `relative_path` = ? AND `Valid` = ?", projectId, strings.TrimSpace(relativePath), 1).
		Order("operated_at desc, modify_time desc, id desc")
	if limit > 0 {
		query = query.Limit(limit)
	}
	var rows []models.ProjectSyncLog
	err := query.Find(&rows).Error
	return rows, err
}

func (r *ProjectSyncLogRepository) GetLatestByTargetProjectAndPath(projectId uint64, relativePath string) (models.ProjectSyncLog, error) {
	var row models.ProjectSyncLog
	err := r.Db().Model(&models.ProjectSyncLog{}).
		Where("`target_project_id` = ? AND `relative_path` = ? AND `Valid` = ?", projectId, strings.TrimSpace(relativePath), 1).
		Order("operated_at desc, modify_time desc, id desc").
		First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.ProjectSyncLog{}, nil
		}
		return models.ProjectSyncLog{}, err
	}
	return row, nil
}

func (r *ProjectSyncLogRepository) ListLatestByTargetProject(projectId uint64) ([]models.ProjectSyncLog, error) {
	var rows []models.ProjectSyncLog
	err := r.latestByTargetPathQuery().
		Where("current_log.`target_project_id` = ?", projectId).
		Find(&rows).Error
	return rows, err
}

func (r *ProjectSyncLogRepository) ListLatestByTargetProjectAndPaths(projectId uint64, relativePaths []string) ([]models.ProjectSyncLog, error) {
	paths := sanitizeProjectSyncLogPaths(relativePaths)
	if len(paths) == 0 {
		return []models.ProjectSyncLog{}, nil
	}
	rows := make([]models.ProjectSyncLog, 0, len(paths))
	for start := 0; start < len(paths); start += latestProjectSyncLogPathBatchSize {
		end := start + latestProjectSyncLogPathBatchSize
		if end > len(paths) {
			end = len(paths)
		}
		batch := make([]models.ProjectSyncLog, 0, end-start)
		if err := r.latestByTargetPathQuery().
			Where("current_log.`target_project_id` = ?", projectId).
			Where("current_log.`relative_path` IN ?", paths[start:end]).
			Find(&batch).Error; err != nil {
			return nil, err
		}
		rows = append(rows, batch...)
	}
	return rows, nil
}

func (r *ProjectSyncLogRepository) CountAllValid() (int64, error) {
	var total int64
	err := r.Db().Model(&models.ProjectSyncLog{}).
		Where("`Valid` = ?", 1).
		Count(&total).Error
	return total, err
}

func (r *ProjectSyncLogRepository) ListAllValidForCleanup() ([]models.ProjectSyncLog, error) {
	var rows []models.ProjectSyncLog
	err := r.Db().Model(&models.ProjectSyncLog{}).
		Where("`Valid` = ?", 1).
		Order("target_project_id asc, relative_path asc, operated_at desc, modify_time desc, id desc").
		Find(&rows).Error
	return rows, err
}

func (r *ProjectSyncLogRepository) HardDeleteLogsForCleanup(keepIDs []uint64) (int64, error) {
	deletedRows := int64(0)
	query := r.Db().Unscoped().Where("`Valid` = ?", 1)
	if len(keepIDs) > 0 {
		query = query.Where("`id` NOT IN ?", keepIDs)
	}
	result := query.Delete(&models.ProjectSyncLog{})
	if result.Error != nil {
		return 0, result.Error
	}
	deletedRows += result.RowsAffected
	invalidResult := r.Db().Unscoped().Where("`Valid` = ?", 0).Delete(&models.ProjectSyncLog{})
	if invalidResult.Error != nil {
		return deletedRows, invalidResult.Error
	}
	deletedRows += invalidResult.RowsAffected
	return deletedRows, nil
}

func (r *ProjectSyncLogRepository) VacuumDatabase() error {
	return r.Db().Exec("VACUUM").Error
}

func (r *ProjectSyncLogRepository) CountAllValidWithFilter(validProjectIds []uint64, keyword, changeType, projectName string) (int64, error) {
	var total int64
	err := r.buildAllValidFilterQuery(validProjectIds, keyword, changeType, projectName).
		Count(&total).Error
	return total, err
}

func (r *ProjectSyncLogRepository) ListAllValid(limit, offset int, validProjectIds []uint64, keyword, changeType, projectName string) ([]models.ProjectSyncLog, error) {
	query := r.buildAllValidFilterQuery(validProjectIds, keyword, changeType, projectName).
		Order("operated_at desc, modify_time desc, id desc")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	var rows []models.ProjectSyncLog
	err := query.Find(&rows).Error
	return rows, err
}

func (r *ProjectSyncLogRepository) buildAllValidFilterQuery(validProjectIds []uint64, keyword, changeType, projectName string) *gorm.DB {
	query := r.Db().Model(&models.ProjectSyncLog{}).
		Where("`Valid` = ?", 1)
	if len(validProjectIds) == 0 {
		return query.Where("1 = 0")
	}
	query = query.Where("`target_project_id` IN ?", validProjectIds)
	trimmedKeyword := strings.TrimSpace(keyword)
	if trimmedKeyword != "" {
		likeKeyword := "%" + trimmedKeyword + "%"
		query = query.Where("`relative_path` LIKE ? OR `target_project_name` LIKE ? OR `executor_node_name` LIKE ? OR `operator_ip` LIKE ?", likeKeyword, likeKeyword, likeKeyword, likeKeyword)
	}
	trimmedChangeType := strings.TrimSpace(changeType)
	if trimmedChangeType != "" {
		query = query.Where("`change_type` = ?", trimmedChangeType)
	}
	trimmedProjectName := strings.TrimSpace(projectName)
	if trimmedProjectName != "" {
		query = query.Where("`target_project_name` LIKE ?", "%"+trimmedProjectName+"%")
	}
	return query
}

func (r *ProjectSyncLogRepository) latestByTargetPathQuery() *gorm.DB {
	newerExistsQuery := r.Db().Table("project_sync_logs AS newer_log").
		Select("1").
		Where("newer_log.`Valid` = ?", 1).
		Where("newer_log.`target_project_id` = current_log.`target_project_id`").
		Where("newer_log.`relative_path` = current_log.`relative_path`").
		Where("(newer_log.`operated_at` > current_log.`operated_at` OR (newer_log.`operated_at` = current_log.`operated_at` AND newer_log.`modify_time` > current_log.`modify_time`) OR (newer_log.`operated_at` = current_log.`operated_at` AND newer_log.`modify_time` = current_log.`modify_time` AND newer_log.`id` > current_log.`id`))")
	return r.Db().Table("project_sync_logs AS current_log").
		Select("current_log.*").
		Where("current_log.`Valid` = ?", 1).
		Where("NOT EXISTS (?)", newerExistsQuery)
}

func sanitizeProjectSyncLogPaths(relativePaths []string) []string {
	if len(relativePaths) == 0 {
		return nil
	}
	paths := make([]string, 0, len(relativePaths))
	seen := make(map[string]struct{}, len(relativePaths))
	for _, path := range relativePaths {
		trimmed := strings.TrimSpace(path)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		paths = append(paths, trimmed)
	}
	return paths
}
