package models

import (
	"time"

	pkgModels "neko-tool/pkg/models"
)

type ProjectNodeCache struct {
	*pkgModels.BaseModel
	ProjectId    uint64     `gorm:"column:project_id;uniqueIndex:uk_project_relative_path,priority:1;index:idx_project_parent_path,priority:1;comment:所属项目ID" json:"projectId"`
	RelativePath string     `gorm:"column:relative_path;size:2048;uniqueIndex:uk_project_relative_path,priority:2;comment:相对路径" json:"relativePath"`
	ParentPath   string     `gorm:"column:parent_path;size:2048;index:idx_project_parent_path,priority:2;comment:父级相对路径" json:"parentPath"`
	Name         string     `gorm:"column:name;size:512;comment:节点名称" json:"name"`
	EntryType    string     `gorm:"column:entry_type;size:32;index;comment:节点类型" json:"entryType"`
	Hash         string     `gorm:"column:hash;size:128;comment:节点摘要哈希" json:"hash"`
	Size         int64      `gorm:"column:size;comment:文件大小" json:"size"`
	EntryModTime int64      `gorm:"column:entry_mod_time;comment:节点修改时间" json:"modifyTime"`
	Depth        int        `gorm:"column:depth;index;comment:路径层级深度" json:"depth"`
	HasChildren  bool       `gorm:"column:has_children;comment:是否存在子节点" json:"hasChildren"`
	DiskDeleted  bool       `gorm:"column:disk_deleted;comment:磁盘中是否已删除" json:"diskDeleted"`
	LastScanAt   *time.Time `gorm:"column:last_scan_at;comment:最近扫描时间" json:"lastScanAt"`
}
