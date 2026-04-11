package service

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"neko-tool/internal/models"
	internalRepo "neko-tool/internal/repository"
	pkgModels "neko-tool/pkg/models"
	"neko-tool/pkg/service"
	"neko-tool/pkg/until"
)

var serverNodeLog = until.Log

type ServerNodeService struct {
	*service.BaseService[models.ServerNode]
}

type NodeRefreshResult struct {
	Checked int      `json:"checked"`
	Updated int      `json:"updated"`
	Failed  []string `json:"failed"`
}

// NewServerNodeService 构造远程节点管理服务。
func NewServerNodeService(repo *internalRepo.ServerNodeRepository) *ServerNodeService {
	return &ServerNodeService{
		BaseService: service.NewBaseService[models.ServerNode](repo),
	}
}

func normalizeNodeBaseURL(baseURL string) (string, error) {
	trimmed := strings.TrimSpace(baseURL)
	if trimmed == "" {
		return "", nil
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("节点地址格式不正确，请填写例如 http://127.0.0.1:8888 的地址")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("节点地址仅支持 http 或 https 协议")
	}
	return parsed.Scheme + "://" + parsed.Host, nil
}

// CreateNode 创建新的远程节点配置。
func (s *ServerNodeService) CreateNode(ctx context.Context, name, baseURL, apiToken, description string) (models.ServerNode, error) {
	normalizedBaseURL, err := normalizeNodeBaseURL(baseURL)
	if err != nil {
		return models.ServerNode{}, err
	}
	node := models.ServerNode{
		BaseModel:   &pkgModels.BaseModel{},
		Name:        strings.TrimSpace(name),
		BaseURL:     normalizedBaseURL,
		ApiToken:    strings.TrimSpace(apiToken),
		Description: strings.TrimSpace(description),
		Enabled:     1,
	}
	if node.Name == "" || node.BaseURL == "" {
		return node, fmt.Errorf("节点名称和地址不能为空")
	}
	if err := s.Exec(ctx).Create(&node); err != nil {
		serverNodeLog.Error("创建节点失败: ", err)
		return node, err
	}
	return node, nil
}

// UpdateNode 更新指定节点的连接信息与启用状态。
func (s *ServerNodeService) UpdateNode(ctx context.Context, id uint64, name, baseURL, apiToken, description string, enabled int) error {
	entity, err := s.GetNode(ctx, id)
	if err != nil {
		return err
	}
	normalizedBaseURL, err := normalizeNodeBaseURL(baseURL)
	if err != nil {
		return err
	}
	entity.Name = strings.TrimSpace(name)
	entity.BaseURL = normalizedBaseURL
	entity.ApiToken = strings.TrimSpace(apiToken)
	entity.Description = strings.TrimSpace(description)
	entity.Enabled = enabled
	if entity.Name == "" || entity.BaseURL == "" {
		return fmt.Errorf("节点名称和地址不能为空")
	}
	if err := s.Exec(ctx).Update(&entity); err != nil {
		serverNodeLog.Error("更新节点失败: ", err)
		return err
	}
	return nil
}

// GetNode 按 ID 查询单个节点配置。
func (s *ServerNodeService) GetNode(ctx context.Context, id uint64) (models.ServerNode, error) {
	entity, err := s.Exec(ctx).One(nil, "`id` = ? AND `Valid` = ?", id, 1)
	if err != nil {
		serverNodeLog.Error("查询节点失败: ", err)
		return entity, err
	}
	return entity, nil
}

// DeleteNode 软删除指定节点配置。
func (s *ServerNodeService) DeleteNode(ctx context.Context, id uint64) error {
	if err := s.Exec(ctx).Delete("`id` = ?", id); err != nil {
		serverNodeLog.Error("删除节点失败: ", err)
		return err
	}
	return nil
}

// ApplyRemoteNodeInfo 用远端返回的节点名称更新当前配置节点。
func (s *ServerNodeService) ApplyRemoteNodeInfo(ctx context.Context, id uint64, baseURL, apiToken, remoteName, description string, enabled int) error {
	entity, err := s.GetNode(ctx, id)
	if err != nil {
		return err
	}
	return s.updateNodeEntity(ctx, &entity, strings.TrimSpace(remoteName), baseURL, apiToken, description, enabled)
}

// CreateNodeWithRemoteName 创建节点，并强制使用远端返回的名称。
func (s *ServerNodeService) CreateNodeWithRemoteName(ctx context.Context, remoteName, baseURL, apiToken, description string) (models.ServerNode, error) {
	node := models.ServerNode{BaseModel: &pkgModels.BaseModel{}, Enabled: 1}
	if err := s.updateNodeEntity(ctx, &node, strings.TrimSpace(remoteName), baseURL, apiToken, description, 1); err != nil {
		return node, err
	}
	if err := s.Exec(ctx).Create(&node); err != nil {
		serverNodeLog.Error("创建节点失败: ", err)
		return node, err
	}
	return node, nil
}

func (s *ServerNodeService) updateNodeEntity(ctx context.Context, entity *models.ServerNode, name, baseURL, apiToken, description string, enabled int) error {
	normalizedBaseURL, err := normalizeNodeBaseURL(baseURL)
	if err != nil {
		return err
	}
	entity.Name = strings.TrimSpace(name)
	entity.BaseURL = normalizedBaseURL
	entity.ApiToken = strings.TrimSpace(apiToken)
	entity.Description = strings.TrimSpace(description)
	entity.Enabled = enabled
	if entity.Name == "" || entity.BaseURL == "" {
		return fmt.Errorf("节点名称和地址不能为空")
	}
	if entity.BaseModel == nil || entity.Id == nil || *entity.Id == 0 {
		return nil
	}
	if err := s.Exec(ctx).Update(entity); err != nil {
		serverNodeLog.Error("更新节点失败: ", err)
		return err
	}
	return nil
}
