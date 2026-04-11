package models

import (
	"time"

	pkgModels "neko-tool/pkg/models"
)

// ProjectSyncLog 记录一次文件同步覆盖的详细内容与上下文。
type ProjectSyncLog struct {
	*pkgModels.BaseModel
	ChangeType          string     `gorm:"column:change_type;index;comment:变更类型" json:"changeType"`
	ScopeType           string     `gorm:"column:scope_type;comment:同步范围类型" json:"scopeType"`
	RelativePath        string     `gorm:"column:relative_path;size:2048;index:idx_sync_log_target_path,priority:2;comment:文件相对路径" json:"relativePath"`
	SourceNodeId        uint64     `gorm:"column:source_node_id;comment:来源节点ID" json:"sourceNodeId"`
	SourceNodeName      string     `gorm:"column:source_node_name;comment:来源节点名称" json:"sourceNodeName"`
	SourceProjectId     uint64     `gorm:"column:source_project_id;comment:来源项目ID" json:"sourceProjectId"`
	SourceProjectName   string     `gorm:"column:source_project_name;comment:来源项目名称" json:"sourceProjectName"`
	TargetNodeId        uint64     `gorm:"column:target_node_id;comment:目标节点ID" json:"targetNodeId"`
	TargetNodeName      string     `gorm:"column:target_node_name;comment:目标节点名称" json:"targetNodeName"`
	TargetProjectId     uint64     `gorm:"column:target_project_id;index:idx_sync_log_target_path,priority:1;comment:目标项目ID" json:"targetProjectId"`
	TargetProjectName   string     `gorm:"column:target_project_name;comment:目标项目名称" json:"targetProjectName"`
	ExecutorNodeName    string     `gorm:"column:executor_node_name;comment:执行操作站名称" json:"executorNodeName"`
	ExecutorNodeAddress string     `gorm:"column:executor_node_address;comment:执行操作站地址" json:"executorNodeAddress"`
	OperatorIP          string     `gorm:"column:operator_ip;comment:操作人IP地址" json:"operatorIP"`
	BeforeExists        bool       `gorm:"column:before_exists;comment:覆盖前文件是否存在" json:"beforeExists"`
	BeforeHash          string     `gorm:"column:before_hash;comment:覆盖前文件哈希" json:"beforeHash"`
	BeforeEncoding      string     `gorm:"column:before_encoding;comment:覆盖前内容编码" json:"beforeEncoding"`
	BeforeStorageKind   string     `gorm:"column:before_storage_kind;comment:覆盖前内容存储方式" json:"beforeStorageKind"`
	BeforeContentSize   int64      `gorm:"column:before_content_size;comment:覆盖前文件大小" json:"beforeContentSize"`
	BeforeOmittedReason string     `gorm:"column:before_omitted_reason;comment:覆盖前内容省略原因" json:"beforeOmittedReason"`
	BeforeContent       string     `gorm:"column:before_content;type:text;comment:覆盖前文件内容" json:"beforeContent"`
	AfterHash           string     `gorm:"column:after_hash;comment:覆盖后文件哈希" json:"afterHash"`
	AfterEncoding       string     `gorm:"column:after_encoding;comment:覆盖后内容编码" json:"afterEncoding"`
	AfterStorageKind    string     `gorm:"column:after_storage_kind;comment:覆盖后内容存储方式" json:"afterStorageKind"`
	AfterContentSize    int64      `gorm:"column:after_content_size;comment:覆盖后文件大小" json:"afterContentSize"`
	AfterOmittedReason  string     `gorm:"column:after_omitted_reason;comment:覆盖后内容省略原因" json:"afterOmittedReason"`
	AfterContent        string     `gorm:"column:after_content;type:text;comment:覆盖后文件内容" json:"afterContent"`
	DiffAlgorithm       string     `gorm:"column:diff_algorithm;comment:差异算法" json:"diffAlgorithm"`
	OperatedAt          *time.Time `gorm:"column:operated_at;index;comment:详细操作时间" json:"operatedAt"`
}
