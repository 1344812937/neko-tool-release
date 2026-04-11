package service

import (
	"context"
	"strings"
	"time"

	"neko-tool/internal/models"
	internalRepo "neko-tool/internal/repository"
	"neko-tool/pkg/core/tx"
	pkgModels "neko-tool/pkg/models"
	"neko-tool/pkg/service"
	"neko-tool/pkg/until"
)

var projectSyncLog = until.Log

// ProjectSyncLogService 文件同步详细日志服务。
type ProjectSyncLogService struct {
	*service.BaseService[models.ProjectSyncLog]
	repo *internalRepo.ProjectSyncLogRepository
}

type DatabaseVacuumResult struct {
	Before      DatabaseFileInfo
	After       DatabaseFileInfo
	VacuumError string
}

// NewProjectSyncLogService 构造器。
func NewProjectSyncLogService(repo *internalRepo.ProjectSyncLogRepository) *ProjectSyncLogService {
	return &ProjectSyncLogService{
		BaseService: service.NewBaseService[models.ProjectSyncLog](repo),
		repo:        repo,
	}
}

// RecordLog 持久化一次文件同步详细日志。
func (s *ProjectSyncLogService) RecordLog(ctx context.Context, entry models.ProjectSyncLog) error {
	if entry.BaseModel == nil {
		entry.BaseModel = &pkgModels.BaseModel{}
	}
	if entry.OperatedAt == nil {
		now := time.Now()
		entry.OperatedAt = &now
	}
	if err := s.Exec(ctx).Create(&entry); err != nil {
		projectSyncLog.Error("记录文件同步日志失败: ", err)
		return err
	}
	return nil
}

func (s *ProjectSyncLogService) ListByTargetProjectAndPath(ctx context.Context, projectId uint64, relativePath string) ([]models.ProjectSyncLog, error) {
	rows, err := s.repoFor(ctx).ListByTargetProjectAndPath(projectId, relativePath, 0)
	if err != nil {
		projectSyncLog.Error("查询文件修改日志失败: ", err)
		return nil, err
	}
	return rows, nil
}

func (s *ProjectSyncLogService) GetLatestByTargetProjectAndPath(ctx context.Context, projectId uint64, relativePath string) (models.ProjectSyncLog, bool, error) {
	row, err := s.repoFor(ctx).GetLatestByTargetProjectAndPath(projectId, relativePath)
	if err != nil {
		projectSyncLog.Error("查询最新文件修改日志失败: ", err)
		return models.ProjectSyncLog{}, false, err
	}
	if row.BaseModel == nil || row.Id == nil || *row.Id == 0 {
		return models.ProjectSyncLog{}, false, nil
	}
	return row, true, nil
}

func (s *ProjectSyncLogService) ListLatestByTargetProject(ctx context.Context, projectId uint64) (map[string]models.ProjectSyncLog, error) {
	rows, err := s.repoFor(ctx).ListLatestByTargetProject(projectId)
	if err != nil {
		projectSyncLog.Error("批量查询项目最新文件修改日志失败: ", err)
		return nil, err
	}
	return buildLatestProjectSyncLogIndex(rows), nil
}

func (s *ProjectSyncLogService) ListLatestByTargetProjectAndPaths(ctx context.Context, projectId uint64, relativePaths []string) (map[string]models.ProjectSyncLog, error) {
	rows, err := s.repoFor(ctx).ListLatestByTargetProjectAndPaths(projectId, relativePaths)
	if err != nil {
		projectSyncLog.Error("按路径批量查询最新文件修改日志失败: ", err)
		return nil, err
	}
	return buildLatestProjectSyncLogIndex(rows), nil
}

func (s *ProjectSyncLogService) CountAllValid(ctx context.Context, validProjectIds []uint64, keyword, changeType, projectName string) (int64, error) {
	total, err := s.repoFor(ctx).CountAllValidWithFilter(validProjectIds, keyword, changeType, projectName)
	if err != nil {
		projectSyncLog.Error("统计全站操作日志失败: ", err)
		return 0, err
	}
	return total, nil
}

func (s *ProjectSyncLogService) ListAllValid(ctx context.Context, pageNo, pageSize int, validProjectIds []uint64, keyword, changeType, projectName string) ([]models.ProjectSyncLog, error) {
	if pageNo < 1 {
		pageNo = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (pageNo - 1) * pageSize
	rows, err := s.repoFor(ctx).ListAllValid(pageSize, offset, validProjectIds, keyword, changeType, projectName)
	if err != nil {
		projectSyncLog.Error("查询全站操作日志失败: ", err)
		return nil, err
	}
	return rows, nil
}

func (s *ProjectSyncLogService) ListAllValidForCleanup(ctx context.Context) ([]models.ProjectSyncLog, error) {
	rows, err := s.repoFor(ctx).ListAllValidForCleanup()
	if err != nil {
		projectSyncLog.Error("查询待清理操作日志失败: ", err)
		return nil, err
	}
	return rows, nil
}

func (s *ProjectSyncLogService) HardDeleteLogsForCleanup(ctx context.Context, keepIDs []uint64) (int64, error) {
	rowsAffected, err := s.repoFor(ctx).HardDeleteLogsForCleanup(keepIDs)
	if err != nil {
		projectSyncLog.Error("清理操作日志失败: ", err)
		return 0, err
	}
	return rowsAffected, nil
}

func (s *ProjectSyncLogService) HardDeleteLogsForCleanupAndVacuum(ctx context.Context, keepIDs []uint64) (int64, DatabaseVacuumResult, error) {
	before := readPrimaryDatabaseFileInfo()
	rowsAffected, err := s.HardDeleteLogsForCleanup(ctx, keepIDs)
	if err != nil {
		return 0, DatabaseVacuumResult{Before: before, After: readPrimaryDatabaseFileInfo()}, err
	}
	vacuumErr := s.repo.VacuumDatabase()
	if vacuumErr != nil {
		projectSyncLog.Error("整理 SQLite 数据库失败: ", vacuumErr)
	}
	after := readPrimaryDatabaseFileInfo()
	result := DatabaseVacuumResult{Before: before, After: after}
	if vacuumErr != nil {
		result.VacuumError = vacuumErr.Error()
	}
	return rowsAffected, result, nil
}

// EnsureLocalFileLog 当磁盘文件内容与最近一次日志记录不一致时，补充一条本地修改日志。
func (s *ProjectSyncLogService) EnsureLocalFileLog(ctx context.Context, project models.WorkSpace, relativePath string, current FileSide, localNodeName, localNodeAddress string) (bool, error) {
	projectId := derefUint64(project.Id)
	if projectId == 0 {
		return false, nil
	}
	relativePath = normalizeRelativePath(relativePath)
	latest, found, err := s.GetLatestByTargetProjectAndPath(ctx, projectId, relativePath)
	if err != nil {
		return false, err
	}
	return s.ensureLocalFileLogWithLatest(ctx, project, relativePath, current, latest, found, localNodeName, localNodeAddress)
}

func (s *ProjectSyncLogService) ensureLocalFileLogWithLatest(ctx context.Context, project models.WorkSpace, relativePath string, current FileSide, latest models.ProjectSyncLog, found bool, localNodeName, localNodeAddress string) (bool, error) {
	projectId := derefUint64(project.Id)
	if projectId == 0 {
		return false, nil
	}
	relativePath = normalizeRelativePath(relativePath)
	if found && sameAsLatestLog(latest, current) {
		return false, nil
	}
	if !found && !current.Exists {
		return false, nil
	}
	beforeExists, beforeEncoding, beforeHash, beforeContent, beforeSize, beforeOmittedReason := previousSnapshot(latest, found)
	beforeSnapshot, afterSnapshot, diffAlgorithm, err := buildProjectSyncLogSnapshots(FileSide{
		Exists:  beforeExists,
		Hash:    beforeHash,
		Text:    beforeEncoding == logContentEncodingText,
		Content: beforeContent,
		Size:    beforeSize,
	}, current)
	if err != nil {
		return false, err
	}
	now := time.Now()
	entry := models.ProjectSyncLog{
		ChangeType:          buildLocalChangeType(found, current.Exists),
		ScopeType:           "file",
		RelativePath:        relativePath,
		SourceNodeId:        0,
		SourceNodeName:      normalizedLocalNodeName(localNodeName),
		SourceProjectId:     projectId,
		SourceProjectName:   project.Name,
		TargetNodeId:        0,
		TargetNodeName:      normalizedLocalNodeName(localNodeName),
		TargetProjectId:     projectId,
		TargetProjectName:   project.Name,
		ExecutorNodeName:    "本地",
		ExecutorNodeAddress: strings.TrimSpace(localNodeAddress),
		OperatorIP:          "本地",
		BeforeExists:        beforeExists,
		BeforeHash:          beforeHash,
		BeforeEncoding:      beforeSnapshot.Encoding,
		BeforeStorageKind:   beforeSnapshot.StorageKind,
		BeforeContentSize:   beforeSnapshot.ContentSize,
		BeforeOmittedReason: coalesceLogOmittedReason(beforeSnapshot.OmittedReason, beforeOmittedReason),
		BeforeContent:       beforeSnapshot.Content,
		AfterHash:           current.Hash,
		AfterEncoding:       afterSnapshot.Encoding,
		AfterStorageKind:    afterSnapshot.StorageKind,
		AfterContentSize:    afterSnapshot.ContentSize,
		AfterOmittedReason:  afterSnapshot.OmittedReason,
		AfterContent:        afterSnapshot.Content,
		DiffAlgorithm:       diffAlgorithm,
		OperatedAt:          &now,
	}
	if err := s.RecordLog(ctx, entry); err != nil {
		return false, err
	}
	return true, nil
}

func (s *ProjectSyncLogService) repoFor(ctx context.Context) *internalRepo.ProjectSyncLogRepository {
	if scope, ok := tx.FromCtx(ctx); ok {
		return s.repo.Scoped(scope)
	}
	return s.repo
}

func normalizedLocalNodeName(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "本地"
	}
	return trimmed
}

func previousSnapshot(entry models.ProjectSyncLog, found bool) (bool, string, string, string, int64, string) {
	if !found {
		return false, logContentEncodingNone, "", "", 0, ""
	}
	afterMeta := normalizeLogContentMeta(entry.AfterEncoding, entry.AfterStorageKind, entry.AfterContent, entry.AfterContentSize, entry.AfterOmittedReason)
	content := visibleProjectSyncLogContent(afterMeta, entry.AfterContent)
	return logEntryContentExists(entry.AfterEncoding, entry.AfterStorageKind), afterMeta.Encoding, entry.AfterHash, content, afterMeta.ContentSize, afterMeta.OmittedReason
}

func sameAsLatestLog(entry models.ProjectSyncLog, current FileSide) bool {
	if !current.Exists {
		return !logEntryContentExists(entry.AfterEncoding, entry.AfterStorageKind) && strings.TrimSpace(entry.AfterHash) == ""
	}
	if (entry.AfterEncoding == logContentEncodingText) != current.Text {
		return false
	}
	return latestLogCompareHash(entry) == fileSideCompareHash(current)
}

func latestLogCompareHash(entry models.ProjectSyncLog) string {
	return logEntryCompareHash(entry.AfterEncoding, entry.AfterStorageKind, entry.AfterContent, entry.AfterHash)
}

func buildLocalChangeType(found bool, exists bool) string {
	if !found {
		return "local_snapshot"
	}
	if !exists {
		return "local_deleted"
	}
	return "local_modified"
}

func coalesceLogOmittedReason(primary, fallback string) string {
	if strings.TrimSpace(primary) != "" {
		return primary
	}
	return strings.TrimSpace(fallback)
}

func buildLatestProjectSyncLogIndex(rows []models.ProjectSyncLog) map[string]models.ProjectSyncLog {
	index := make(map[string]models.ProjectSyncLog, len(rows))
	for _, row := range rows {
		relativePath := normalizeRelativePath(row.RelativePath)
		if relativePath == "" {
			continue
		}
		index[relativePath] = row
	}
	return index
}
