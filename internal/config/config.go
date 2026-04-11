package config

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"neko-tool/pkg/until"
	"os"
	"path/filepath"
	"strings"

	"github.com/creasty/defaults"
	"github.com/pelletier/go-toml/v2"
)

var log = until.Log
var configPath = filepath.Join("./", "config", "config.toml")

type ApplicationConfigManager struct {
	config *ApplicationConfig
}

func (acm *ApplicationConfigManager) GetConfig() *ApplicationConfig {
	if acm.config == nil {
		acm.Load()
	}
	return acm.config
}

func (acm *ApplicationConfigManager) Load() {
	firstRun := false
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		createDefaultConfig(configPath)
		firstRun = true
	}
	content, err := os.ReadFile(configPath)
	if err != nil {
		panic("读取配置文件失败")
	}
	presence, err := detectConfigFieldPresence(content)
	if err != nil {
		panic(fmt.Sprintf("解析配置项存在性失败: %v", err))
	}

	// 解析配置文件
	var cfg ApplicationConfig
	if err := defaults.Set(&cfg); err != nil {
		panic(err)
	}
	if err := toml.Unmarshal(content, &cfg); err != nil {
		panic("解析配置文件失败")
	}
	normalizeConfig(&cfg)

	guidePlan, err := buildStartupGuidePlan(firstRun, presence, &cfg)
	if err != nil {
		panic(fmt.Sprintf("构建启动配置引导失败: %v", err))
	}
	if guidePlan.HasQuestions() {
		if canPromptForConfig() {
			if guideErr := runStartupConfigGuide(configPath, &cfg, guidePlan); guideErr != nil {
				panic(fmt.Sprintf("启动配置引导失败: %v", guideErr))
			}
		} else {
			log.Warn("检测到缺失启动配置，但当前环境不是交互式终端，已按默认值或自动生成值继续启动")
		}
	}
	needRewrite, err := applyRuntimeFallbacks(&cfg, guidePlan)
	if err != nil {
		panic(fmt.Sprintf("补全启动配置失败: %v", err))
	}
	normalizeConfig(&cfg)
	if firstRun || guidePlan.HasQuestions() || needRewrite {
		if writeErr := writeConfigFile(configPath, &cfg); writeErr != nil {
			panic(fmt.Sprintf("写入配置文件失败: %v", writeErr))
		}
	}

	acm.config = &cfg
}

func NewApplicationConfigManager() *ApplicationConfigManager {
	configManager := &ApplicationConfigManager{}
	configManager.Load()
	return configManager
}

type ApplicationConfig struct {
	WebConfig     WebConfig     `toml:"web_config"`
	ProjectAccess ProjectAccess `toml:"project_access"`
	NodeConfig    NodeConfig    `toml:"node_config"`
	AuthConfig    AuthConfig    `toml:"auth_config"`
}

type WebConfig struct {
	Host string `toml:"host" default:"localhost"`
	Port string `toml:"port" default:"8888"`
}

type ProjectAccess struct {
	EnableNativeLocalPicker bool     `toml:"enable_native_local_picker" default:"true"`
	AllowedRoots            []string `toml:"allowed_roots"`
	FollowSymlink           bool     `toml:"follow_symlink" default:"false"`
}

type NodeConfig struct {
	Name               string `toml:"name"`
	SharedToken        string `toml:"shared_token"`
	WorkstationAddress string `toml:"workstation_address"`
}

type AuthConfig struct {
	AccessToken string `toml:"access_token"`
}

func generateSharedToken() (string, error) {
	return generatePrefixedToken("neko")
}

func generateAccessToken() (string, error) {
	return generatePrefixedToken("access")
}

func generateWorkstationName() (string, error) {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	raw := make([]byte, 8)
	if _, err := io.ReadFull(rand.Reader, raw); err != nil {
		return "", err
	}
	var builder strings.Builder
	builder.Grow(len(raw))
	for _, item := range raw {
		builder.WriteByte(alphabet[int(item)%len(alphabet)])
	}
	return builder.String(), nil
}

func generatePrefixedToken(prefix string) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return prefix + "_" + base64.RawURLEncoding.EncodeToString(raw), nil
}

func encodeConfig(cfg *ApplicationConfig) ([]byte, error) {
	var buf bytes.Buffer
	encoder := toml.NewEncoder(&buf)
	encoder.SetIndentTables(true)
	encoder.SetTablesInline(false)
	if err := encoder.Encode(cfg); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeConfigFile(filePath string, cfg *ApplicationConfig) error {
	content, err := encodeConfig(cfg)
	if err != nil {
		return err
	}
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filePath, content, 0644)
}

func createDefaultConfig(filePath string) *ApplicationConfig {
	defaultConfig := &ApplicationConfig{}
	err := defaults.Set(defaultConfig)
	if err != nil {
		panic(err)
	}
	defaultConfig.ProjectAccess.AllowedRoots = []string{}
	if err := writeConfigFile(filePath, defaultConfig); err != nil {
		panic(err)
	}
	return defaultConfig
}

func normalizeConfig(cfg *ApplicationConfig) {
	if cfg == nil {
		return
	}
	for i, root := range cfg.ProjectAccess.AllowedRoots {
		trimmed := strings.TrimSpace(root)
		if trimmed == "" {
			continue
		}
		cfg.ProjectAccess.AllowedRoots[i] = filepath.Clean(trimmed)
	}
}
