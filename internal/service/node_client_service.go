package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"neko-tool/internal/config"
	"neko-tool/pkg/common"
	"neko-tool/pkg/until"
)

const nodeTokenHeader = "X-Neko-Node-Token"
const nodeTimestampHeader = "X-Neko-Node-Timestamp"
const nodeAuthTTL = 5 * time.Minute

var nodeClientLog = until.Log

type NodeClientService struct {
	nodeService   *ServerNodeService
	configManager *config.ApplicationConfigManager
	httpClient    *http.Client
}

// NewNodeClientService 构造远程节点 HTTP 客户端服务。
func NewNodeClientService(nodeService *ServerNodeService, configManager *config.ApplicationConfigManager) *NodeClientService {
	return &NodeClientService{
		nodeService:   nodeService,
		configManager: configManager,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// NodeTokenHeader 返回内部节点协同请求使用的签名请求头名称。
func NodeTokenHeader() string {
	return nodeTokenHeader
}

// NodeTimestampHeader 返回内部节点协同请求使用的时间戳请求头名称。
func NodeTimestampHeader() string {
	return nodeTimestampHeader
}

// NodeAuthTTL 返回内部节点签名允许的最大时间漂移。
func NodeAuthTTL() time.Duration {
	return nodeAuthTTL
}

// LocalNodeInfo 返回当前节点自身的基本信息。
func (s *NodeClientService) LocalNodeInfo() NodeInfo {
	cfg := s.configManager.GetConfig()
	return NodeInfo{
		Name:              cfg.NodeConfig.Name,
		TokenConfigured:   strings.TrimSpace(cfg.NodeConfig.SharedToken) != "",
		System:            runtime.GOOS,
		PathSeparator:     string(os.PathSeparator),
		PathCaseSensitive: filesystemCaseSensitive("."),
	}
}

func (s *NodeClientService) getNode(ctx context.Context, nodeID uint64) (string, string, error) {
	node, err := s.nodeService.GetNode(ctx, nodeID)
	if err != nil {
		return "", "", err
	}
	normalizedBaseURL, err := normalizeNodeBaseURL(node.BaseURL)
	if err != nil {
		return "", "", fmt.Errorf("节点地址配置无效: %w", err)
	}
	return normalizedBaseURL, node.ApiToken, nil
}

func resolveNodeEndpoint(baseURL, apiToken string) (string, string, error) {
	normalizedBaseURL, err := normalizeNodeBaseURL(baseURL)
	if err != nil {
		return "", "", fmt.Errorf("节点地址配置无效: %w", err)
	}
	return normalizedBaseURL, strings.TrimSpace(apiToken), nil
}

func summarizeNodeResponseBody(body []byte) string {
	summary := strings.TrimSpace(string(body))
	if summary == "" {
		return ""
	}
	summary = strings.Join(strings.Fields(summary), " ")
	const limit = 120
	if len(summary) > limit {
		return summary[:limit] + "..."
	}
	return summary
}

func signNodeTimestamp(token string, timestamp string) string {
	mac := hmac.New(sha256.New, []byte(token))
	mac.Write([]byte(timestamp))
	return hex.EncodeToString(mac.Sum(nil))
}

// BuildNodeAuthHeaders 使用共享 token 对时间戳做签名，返回请求头需要的签名和值。
func BuildNodeAuthHeaders(token string, now time.Time) (string, string) {
	timestamp := strconv.FormatInt(now.Unix(), 10)
	return signNodeTimestamp(token, timestamp), timestamp
}

// VerifyNodeAuth 校验时间戳是否在允许范围内，并验证签名是否匹配。
func VerifyNodeAuth(token, signature, timestamp string, now time.Time) bool {
	timestampValue, err := strconv.ParseInt(strings.TrimSpace(timestamp), 10, 64)
	if err != nil {
		return false
	}
	requestTime := time.Unix(timestampValue, 0)
	delta := now.Sub(requestTime)
	if delta < 0 {
		delta = -delta
	}
	if delta > nodeAuthTTL {
		return false
	}
	expectedSignature := signNodeTimestamp(token, strings.TrimSpace(timestamp))
	return subtleConstantTimeCompare(signature, expectedSignature)
}

func subtleConstantTimeCompare(left string, right string) bool {
	if len(left) != len(right) {
		return false
	}
	return hmac.Equal([]byte(left), []byte(right))
}

func requestNodeJSON[T any](ctx context.Context, client *http.Client, baseURL, apiToken, method, path string, payload any) (T, error) {
	var zero T
	var body *bytes.Reader
	if payload != nil {
		raw, marshalErr := json.Marshal(payload)
		if marshalErr != nil {
			return zero, marshalErr
		}
		body = bytes.NewReader(raw)
	} else {
		body = bytes.NewReader(nil)
	}
	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, body)
	if err != nil {
		return zero, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if strings.TrimSpace(apiToken) != "" {
		signature, timestamp := BuildNodeAuthHeaders(apiToken, time.Now())
		req.Header.Set(nodeTokenHeader, signature)
		req.Header.Set(nodeTimestampHeader, timestamp)
	}
	resp, err := client.Do(req)
	if err != nil {
		nodeClientLog.Error("请求远程节点失败: ", err)
		return zero, err
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return zero, fmt.Errorf("读取远程节点响应失败: %w", err)
	}
	bodySummary := summarizeNodeResponseBody(bodyBytes)
	if resp.StatusCode != http.StatusOK {
		if bodySummary == "" {
			return zero, fmt.Errorf("远程节点返回 HTTP %d", resp.StatusCode)
		}
		return zero, fmt.Errorf("远程节点返回 HTTP %d: %s", resp.StatusCode, bodySummary)
	}
	trimmedBody := bytes.TrimSpace(bodyBytes)
	if len(trimmedBody) == 0 {
		return zero, fmt.Errorf("远程节点返回空响应")
	}
	var result common.R[T]
	if err := json.Unmarshal(trimmedBody, &result); err != nil {
		if strings.HasPrefix(bodySummary, "<") {
			return zero, fmt.Errorf("远程节点返回了 HTML 页面，请检查节点地址是否填写为服务根地址，例如 http://127.0.0.1:8888，不要填写 /static 或页面路径")
		}
		if bodySummary != "" {
			return zero, fmt.Errorf("解析远程节点响应失败: %v，响应片段: %s", err, bodySummary)
		}
		return zero, fmt.Errorf("解析远程节点响应失败: %w", err)
	}
	if !result.Success {
		return zero, errors.New(result.Message)
	}
	if result.Data == nil {
		return zero, nil
	}
	return *result.Data, nil
}

// Ping 调用远程节点的内部探活接口。
func (s *NodeClientService) Ping(ctx context.Context, nodeID uint64) (NodeInfo, error) {
	baseURL, apiToken, err := s.getNode(ctx, nodeID)
	if err != nil {
		return NodeInfo{}, err
	}
	return requestNodeJSON[NodeInfo](ctx, s.httpClient, baseURL, apiToken, http.MethodGet, "/api/internal/node-info", nil)
}

// FetchNodeInfo 根据节点地址和共享令牌直接读取远端节点信息。
func (s *NodeClientService) FetchNodeInfo(ctx context.Context, baseURL, apiToken string) (NodeInfo, error) {
	resolvedBaseURL, resolvedToken, err := resolveNodeEndpoint(baseURL, apiToken)
	if err != nil {
		return NodeInfo{}, err
	}
	return requestNodeJSON[NodeInfo](ctx, s.httpClient, resolvedBaseURL, resolvedToken, http.MethodGet, "/api/internal/node-info", nil)
}

// ListProjects 查询远程节点暴露的项目列表。
func (s *NodeClientService) ListProjects(ctx context.Context, nodeID uint64) ([]NodeProject, error) {
	baseURL, apiToken, err := s.getNode(ctx, nodeID)
	if err != nil {
		return nil, err
	}
	return requestNodeJSON[[]NodeProject](ctx, s.httpClient, baseURL, apiToken, http.MethodGet, "/api/internal/projects", nil)
}

// BuildManifest 请求远程节点构建指定项目的目录清单。
func (s *NodeClientService) BuildManifest(ctx context.Context, nodeID uint64, request InternalManifestRequest) (ManifestResult, error) {
	baseURL, apiToken, err := s.getNode(ctx, nodeID)
	if err != nil {
		return ManifestResult{}, err
	}
	return requestNodeJSON[ManifestResult](ctx, s.httpClient, baseURL, apiToken, http.MethodPost, "/api/internal/manifest", request)
}

// ReadFile 请求远程节点读取指定项目中的文件内容。
func (s *NodeClientService) ReadFile(ctx context.Context, nodeID uint64, request InternalFileRequest) (FileSide, error) {
	baseURL, apiToken, err := s.getNode(ctx, nodeID)
	if err != nil {
		return FileSide{}, err
	}
	return requestNodeJSON[FileSide](ctx, s.httpClient, baseURL, apiToken, http.MethodPost, "/api/internal/file", request)
}

// WriteFile 请求远程节点写入指定项目中的文件内容。
func (s *NodeClientService) WriteFile(ctx context.Context, nodeID uint64, request InternalWriteFileRequest) error {
	baseURL, apiToken, err := s.getNode(ctx, nodeID)
	if err != nil {
		return err
	}
	_, err = requestNodeJSON[map[string]string](ctx, s.httpClient, baseURL, apiToken, http.MethodPost, "/api/internal/write-file", request)
	return err
}

func derefUint64(value *uint64) uint64 {
	if value == nil {
		return 0
	}
	return *value
}
