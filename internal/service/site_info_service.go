package service

import (
	"strings"
	"time"

	"neko-tool/internal/config"
)

const AppVersion = "1.1.1"

type SiteInfoService struct {
	configManager *config.ApplicationConfigManager
	startedAt     time.Time
	version       string
}

type SiteInfo struct {
	StartedAt         string `json:"startedAt"`
	UptimeSeconds     int64  `json:"uptimeSeconds"`
	NodeName          string `json:"nodeName"`
	WebAddress        string `json:"webAddress"`
	AuthEnabled       bool   `json:"authEnabled"`
	SharedToken       string `json:"sharedToken"`
	Version           string `json:"version"`
	DatabaseSizeBytes int64  `json:"databaseSizeBytes"`
	DatabaseSizeLabel string `json:"databaseSizeLabel"`
}

func NewSiteInfoService(configManager *config.ApplicationConfigManager) *SiteInfoService {
	return &SiteInfoService{
		configManager: configManager,
		startedAt:     time.Now(),
		version:       AppVersion,
	}
}

func (s *SiteInfoService) GetSiteInfo() SiteInfo {
	cfg := s.configManager.GetConfig()
	host := strings.TrimSpace(cfg.WebConfig.Host)
	if host == "" {
		host = "0.0.0.0"
	}
	databaseFileInfo := readPrimaryDatabaseFileInfo()
	return SiteInfo{
		StartedAt:         s.startedAt.Format(time.DateTime),
		UptimeSeconds:     int64(time.Since(s.startedAt).Seconds()),
		NodeName:          strings.TrimSpace(cfg.NodeConfig.Name),
		WebAddress:        host + ":" + strings.TrimSpace(cfg.WebConfig.Port),
		AuthEnabled:       strings.TrimSpace(cfg.AuthConfig.AccessToken) != "",
		SharedToken:       strings.TrimSpace(cfg.NodeConfig.SharedToken),
		Version:           s.version,
		DatabaseSizeBytes: databaseFileInfo.SizeBytes,
		DatabaseSizeLabel: databaseFileInfo.SizeLabel,
	}
}
