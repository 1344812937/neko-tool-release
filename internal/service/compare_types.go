package service

type NodeProject struct {
	Id   uint64 `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
	Path string `json:"path"`
}

type NodeInfo struct {
	Name              string `json:"name"`
	TokenConfigured   bool   `json:"tokenConfigured"`
	System            string `json:"system"`
	PathSeparator     string `json:"pathSeparator"`
	PathCaseSensitive bool   `json:"pathCaseSensitive"`
}

type ManifestEntry struct {
	RelativePath   string `json:"relativePath"`
	Name           string `json:"name"`
	EntryType      string `json:"entryType"`
	Size           int64  `json:"size"`
	ModifyTime     int64  `json:"modifyTime"`
	Hash           string `json:"hash"`
	Text           bool   `json:"text"`
	NormalizedHash string `json:"normalizedHash"`
	HasChildren    bool   `json:"hasChildren"`
	Deleted        bool   `json:"deleted"`
}

type ManifestResult struct {
	Project           NodeProject     `json:"project"`
	BasePath          string          `json:"basePath"`
	Depth             int             `json:"depth"`
	RootHash          string          `json:"rootHash"`
	PathCaseSensitive bool            `json:"pathCaseSensitive"`
	Entries           []ManifestEntry `json:"entries"`
	ProjectDeleted    bool            `json:"projectDeleted"`
	CacheInitialized  bool            `json:"cacheInitialized"`
	FromCache         bool            `json:"fromCache"`
	Message           string          `json:"message"`
}

type CompareRequest struct {
	LeftNodeId     uint64 `json:"leftNodeId"`
	LeftProjectId  uint64 `json:"leftProjectId"`
	RightNodeId    uint64 `json:"rightNodeId"`
	RightProjectId uint64 `json:"rightProjectId"`
	BasePath       string `json:"basePath"`
}

type ProjectBrowseRequest struct {
	NodeId    uint64 `json:"nodeId"`
	ProjectId uint64 `json:"projectId"`
	BasePath  string `json:"basePath"`
	Depth     int    `json:"depth"`
}

type ProjectBrowseFileRequest struct {
	NodeId    uint64 `json:"nodeId"`
	ProjectId uint64 `json:"projectId"`
	Path      string `json:"path"`
}

type ProjectBrowseDeleteRequest struct {
	ProjectId uint64 `json:"projectId"`
	Path      string `json:"path"`
}

type CompareSummary struct {
	RootHashLeft         string `json:"rootHashLeft"`
	RootHashRight        string `json:"rootHashRight"`
	DifferentFiles       int    `json:"differentFiles"`
	DifferentDirectories int    `json:"differentDirectories"`
	LeftOnly             int    `json:"leftOnly"`
	RightOnly            int    `json:"rightOnly"`
	SameCount            int    `json:"sameCount"`
	Total                int    `json:"total"`
}

type CompareItem struct {
	Path      string `json:"path"`
	Name      string `json:"name"`
	EntryType string `json:"entryType"`
	Status    string `json:"status"`
	LeftHash  string `json:"leftHash"`
	RightHash string `json:"rightHash"`
	LeftSize  int64  `json:"leftSize"`
	RightSize int64  `json:"rightSize"`
}

type CompareResult struct {
	Summary                CompareSummary `json:"summary"`
	Items                  []CompareItem  `json:"items"`
	LeftPathCaseSensitive  bool           `json:"leftPathCaseSensitive"`
	RightPathCaseSensitive bool           `json:"rightPathCaseSensitive"`
}

type FileDiffRequest struct {
	LeftNodeId     uint64 `json:"leftNodeId"`
	LeftProjectId  uint64 `json:"leftProjectId"`
	RightNodeId    uint64 `json:"rightNodeId"`
	RightProjectId uint64 `json:"rightProjectId"`
	Path           string `json:"path"`
}

type FileSide struct {
	Exists         bool   `json:"exists"`
	Deleted        bool   `json:"deleted"`
	Hash           string `json:"hash"`
	NormalizedHash string `json:"normalizedHash"`
	Text           bool   `json:"text"`
	Content        string `json:"content"`
	ContentBase64  string `json:"contentBase64"`
	Size           int64  `json:"size"`
}

type DiffLine struct {
	Type        string `json:"type"`
	LeftNumber  int    `json:"leftNumber"`
	RightNumber int    `json:"rightNumber"`
	LeftText    string `json:"leftText"`
	RightText   string `json:"rightText"`
}

type FileDiffResult struct {
	Path  string     `json:"path"`
	Left  FileSide   `json:"left"`
	Right FileSide   `json:"right"`
	Lines []DiffLine `json:"lines"`
}

type SyncRequest struct {
	SourceNodeId    uint64   `json:"sourceNodeId"`
	SourceProjectId uint64   `json:"sourceProjectId"`
	TargetNodeId    uint64   `json:"targetNodeId"`
	TargetProjectId uint64   `json:"targetProjectId"`
	ScopeType       string   `json:"scopeType"`
	Path            string   `json:"path"`
	SelectedPaths   []string `json:"selectedPaths"`
}

type SyncFailure struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

type SyncResult struct {
	Copied  int           `json:"copied"`
	Skipped int           `json:"skipped"`
	Failed  []SyncFailure `json:"failed"`
}

type ProjectSyncLogListRequest struct {
	ProjectId uint64 `json:"projectId"`
	Path      string `json:"path"`
}

type ProjectSyncLogListItem struct {
	Id                  uint64 `json:"id"`
	ChangeType          string `json:"changeType"`
	ScopeType           string `json:"scopeType"`
	RelativePath        string `json:"relativePath"`
	TargetProjectId     uint64 `json:"targetProjectId"`
	SourceNodeName      string `json:"sourceNodeName"`
	SourceProjectName   string `json:"sourceProjectName"`
	TargetNodeName      string `json:"targetNodeName"`
	TargetProjectName   string `json:"targetProjectName"`
	ExecutorNodeName    string `json:"executorNodeName"`
	ExecutorNodeAddress string `json:"executorNodeAddress"`
	OperatorIP          string `json:"operatorIP"`
	BeforeExists        bool   `json:"beforeExists"`
	BeforeHash          string `json:"beforeHash"`
	BeforeEncoding      string `json:"beforeEncoding"`
	AfterHash           string `json:"afterHash"`
	AfterEncoding       string `json:"afterEncoding"`
	OperatedAt          string `json:"operatedAt"`
}

type ProjectSyncLogListResult struct {
	Count int                      `json:"count"`
	Items []ProjectSyncLogListItem `json:"items"`
}

type ProjectSyncLogDetailRequest struct {
	LogId uint64 `json:"logId"`
}

type SiteProjectSyncLogListRequest struct {
	PageNo      int    `json:"pageNo"`
	PageSize    int    `json:"pageSize"`
	Keyword     string `json:"keyword"`
	ChangeType  string `json:"changeType"`
	ProjectName string `json:"projectName"`
}

type SiteProjectSyncLogListResult struct {
	PageNo   int                      `json:"pageNo"`
	PageSize int                      `json:"pageSize"`
	Total    int64                    `json:"total"`
	Items    []ProjectSyncLogListItem `json:"items"`
}

type SiteProjectSyncLogCleanupResult struct {
	FilesScanned            int    `json:"filesScanned"`
	KeptLogs                int    `json:"keptLogs"`
	ClearedLogs             int64  `json:"clearedLogs"`
	DatabaseSizeBeforeBytes int64  `json:"databaseSizeBeforeBytes"`
	DatabaseSizeBeforeLabel string `json:"databaseSizeBeforeLabel"`
	DatabaseSizeAfterBytes  int64  `json:"databaseSizeAfterBytes"`
	DatabaseSizeAfterLabel  string `json:"databaseSizeAfterLabel"`
	VacuumError             string `json:"vacuumError"`
}

type InternalManifestRequest struct {
	ProjectId uint64 `json:"projectId"`
	BasePath  string `json:"basePath"`
}

type InternalFileRequest struct {
	ProjectId    uint64 `json:"projectId"`
	RelativePath string `json:"relativePath"`
}

type SyncLogContext struct {
	ChangeType          string `json:"changeType"`
	ScopeType           string `json:"scopeType"`
	RelativePath        string `json:"relativePath"`
	SourceNodeId        uint64 `json:"sourceNodeId"`
	SourceNodeName      string `json:"sourceNodeName"`
	SourceProjectId     uint64 `json:"sourceProjectId"`
	SourceProjectName   string `json:"sourceProjectName"`
	TargetNodeId        uint64 `json:"targetNodeId"`
	TargetNodeName      string `json:"targetNodeName"`
	TargetProjectId     uint64 `json:"targetProjectId"`
	TargetProjectName   string `json:"targetProjectName"`
	ExecutorNodeName    string `json:"executorNodeName"`
	ExecutorNodeAddress string `json:"executorNodeAddress"`
	OperatorIP          string `json:"operatorIP"`
}

type InternalWriteFileRequest struct {
	ProjectId     uint64          `json:"projectId"`
	RelativePath  string          `json:"relativePath"`
	ContentBase64 string          `json:"contentBase64"`
	ExpectedHash  string          `json:"expectedHash"`
	SyncLog       *SyncLogContext `json:"syncLog,omitempty"`
}
