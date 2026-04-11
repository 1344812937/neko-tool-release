package service

import (
	"context"
	"fmt"
	"neko-tool/internal/models"
	internalRepo "neko-tool/internal/repository"
	pkgModels "neko-tool/pkg/models"
	"neko-tool/pkg/service"
	"neko-tool/pkg/until"
	"path/filepath"
	"strings"
)

var workSpaceLog = until.Log

// WorkSpaceService WorkSpace 业务逻辑层。
type WorkSpaceService struct {
	*service.BaseService[models.WorkSpace]
}

type CreateProjectResult struct {
	Project  models.WorkSpace
	Restored bool
}

// NewWorkSpaceService 构造器。
func NewWorkSpaceService(repo *internalRepo.WorkSpaceRepository) *WorkSpaceService {
	return &WorkSpaceService{
		BaseService: service.NewBaseService[models.WorkSpace](repo),
	}
}

// CreateProject 创建项目；当命中相同编码和路径的逻辑删除项目时自动恢复。
func (s *WorkSpaceService) CreateProject(ctx context.Context, name, code, path string) (CreateProjectResult, error) {
	trimmedName := strings.TrimSpace(name)
	trimmedCode := defaultProjectCode(code, trimmedName, path)
	existing, err := s.Exec(ctx).SelectList(nil, "`code` = ? AND `Valid` = ?", trimmedCode, 1)
	if err != nil {
		workSpaceLog.Error("查询项目编码失败: ", err)
		return CreateProjectResult{}, err
	}
	if len(existing) > 0 {
		return CreateProjectResult{}, fmt.Errorf("项目编码已存在: %s", trimmedCode)
	}
	sortParams := "modify_time desc"
	deletedProjects, err := s.Exec(ctx).SelectList(&sortParams, "`code` = ? AND `path` = ? AND `Valid` = ?", trimmedCode, path, 0)
	if err != nil {
		workSpaceLog.Error("查询已删除项目失败: ", err)
		return CreateProjectResult{}, err
	}
	if len(deletedProjects) > 0 {
		deletedProject := deletedProjects[0]
		updates := map[string]any{
			"name":              trimmedName,
			"code":              trimmedCode,
			"path":              path,
			"valid":             1,
			"disk_deleted":      false,
			"cache_initialized": false,
		}
		result := s.Exec(ctx).Where("`id` = ?", derefUint64(deletedProject.Id)).Updates(updates)
		if result.Error != nil {
			workSpaceLog.Error("恢复已删除项目失败: ", result.Error)
			return CreateProjectResult{}, result.Error
		}
		restoredProject, getErr := s.GetProject(ctx, derefUint64(deletedProject.Id))
		if getErr != nil {
			return CreateProjectResult{}, getErr
		}
		return CreateProjectResult{Project: restoredProject, Restored: true}, nil
	}
	project := models.WorkSpace{
		BaseModel: &pkgModels.BaseModel{},
		Name:      trimmedName,
		Code:      trimmedCode,
		Path:      path,
	}
	if err := s.Exec(ctx).Create(&project); err != nil {
		workSpaceLog.Error("创建项目失败: ", err)
		return CreateProjectResult{}, err
	}
	return CreateProjectResult{Project: project, Restored: false}, nil
}

// GetProject 按 ID 查询单个项目记录。
func (s *WorkSpaceService) GetProject(ctx context.Context, id uint64) (models.WorkSpace, error) {
	project, err := s.Exec(ctx).One(nil, "`id` = ? AND `Valid` = ?", id, 1)
	if err != nil {
		workSpaceLog.Error("查询项目失败: ", err)
		return project, err
	}
	return project, nil
}

// RenameProject 修改项目名称。
func (s *WorkSpaceService) RenameProject(ctx context.Context, id uint64, name string) error {
	result := s.Exec(ctx).Where("`id` = ? AND `Valid` = ?", id, 1).Update("name", name)
	if result.Error != nil {
		workSpaceLog.Error("修改项目名称失败: ", result.Error)
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("项目不存在")
	}
	return nil
}

// DeleteProject 按 ID 软删除项目。
func (s *WorkSpaceService) DeleteProject(ctx context.Context, id uint64) error {
	if err := s.Exec(ctx).Delete("`id` = ?", id); err != nil {
		workSpaceLog.Error("删除项目失败: ", err)
		return err
	}
	return nil
}

func (s *WorkSpaceService) ListValidProjectIDs(ctx context.Context) ([]uint64, error) {
	projects, err := s.SelectList(ctx, nil, "`Valid` = ?", 1)
	if err != nil {
		return nil, err
	}
	ids := make([]uint64, 0, len(projects))
	for _, project := range projects {
		projectId := derefUint64(project.Id)
		if projectId == 0 {
			continue
		}
		ids = append(ids, projectId)
	}
	return ids, nil
}

func defaultProjectCode(code, name, path string) string {
	trimmedCode := strings.TrimSpace(code)
	if trimmedCode != "" {
		return trimmedCode
	}
	trimmedName := strings.TrimSpace(name)
	if trimmedName != "" {
		return trimmedName
	}
	trimmedPath := strings.TrimRight(strings.TrimSpace(path), "/\\")
	if trimmedPath == "" {
		return ""
	}
	return filepath.Base(trimmedPath)
}
