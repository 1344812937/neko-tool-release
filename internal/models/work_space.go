package models

import (
	"time"

	"neko-tool/pkg/models"
)

type WorkSpace struct {
	*models.BaseModel
	Hash             string     `gorm:"column:hash;comment:项目摘要哈希" json:"hash"`
	Name             string     `gorm:"column:name;comment:项目名称" json:"name"`
	Code             string     `gorm:"column:code;size:255;uniqueIndex:uk_workspace_code_active,where:Valid = 1;comment:项目编码" json:"code"`
	Path             string     `gorm:"column:path;index;comment:项目根路径" json:"path"`
	PathNodes        string     `gorm:"column:path_nodes;comment:路径节点缓存摘要" json:"pathNodes"`
	CacheInitialized bool       `gorm:"column:cache_initialized;comment:缓存是否已初始化" json:"cacheInitialized"`
	DiskDeleted      bool       `gorm:"column:disk_deleted;comment:磁盘目录是否已删除" json:"diskDeleted"`
	LastScanAt       *time.Time `gorm:"column:last_scan_at;comment:最近扫描时间" json:"lastScanAt"`
}
