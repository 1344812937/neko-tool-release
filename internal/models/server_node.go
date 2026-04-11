package models

import pkgModels "neko-tool/pkg/models"

type ServerNode struct {
	*pkgModels.BaseModel
	Name        string `gorm:"column:name;comment:节点名称" json:"name"`
	BaseURL     string `gorm:"column:base_url;uniqueIndex;comment:节点基础地址" json:"baseUrl"`
	ApiToken    string `gorm:"column:api_token;comment:节点访问令牌" json:"apiToken"`
	Description string `gorm:"column:description;comment:节点描述" json:"description"`
	Enabled     int    `gorm:"column:enabled;default:1;comment:启用状态" json:"enabled"`
}
