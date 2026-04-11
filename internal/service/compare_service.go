package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"neko-tool/internal/models"
	"neko-tool/pkg/until"

	ignore "github.com/sabhiram/go-gitignore"
)

var compareLog = until.Log

type CompareService struct {
	workSpaceService  *WorkSpaceService
	projectAccess     *ProjectAccessService
	nodeClientService *NodeClientService
	serverNodeService *ServerNodeService
	projectSyncLogSvc *ProjectSyncLogService
}

// NewCompareService 构造项目对比服务。
func NewCompareService(workSpaceService *WorkSpaceService, projectAccess *ProjectAccessService, nodeClientService *NodeClientService, serverNodeService *ServerNodeService, projectSyncLogSvc *ProjectSyncLogService) *CompareService {
	return &CompareService{
		workSpaceService:  workSpaceService,
		projectAccess:     projectAccess,
		nodeClientService: nodeClientService,
		serverNodeService: serverNodeService,
		projectSyncLogSvc: projectSyncLogSvc,
	}
}

// NodeInfo 返回当前节点的基础描述信息。
func (s *CompareService) NodeInfo() NodeInfo {
	return s.nodeClientService.LocalNodeInfo()
}

// ListLocalProjects 返回当前节点本地可访问的项目列表。
func (s *CompareService) ListLocalProjects(ctx context.Context) ([]NodeProject, error) {
	projects, err := s.workSpaceService.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]NodeProject, 0)
	for _, project := range projects {
		validatedPath, validateErr := s.projectAccess.ValidateExistingProjectPath(project.Path)
		if validateErr != nil {
			continue
		}
		result = append(result, NodeProject{
			Id:   derefUint64(project.Id),
			Name: project.Name,
			Code: project.Code,
			Path: validatedPath,
		})
	}
	return result, nil
}

// BuildLocalManifest 为本地项目构建目录摘要清单。
func (s *CompareService) BuildLocalManifest(ctx context.Context, request InternalManifestRequest) (ManifestResult, error) {
	project, rootPath, err := s.getValidatedProject(ctx, request.ProjectId)
	if err != nil {
		return ManifestResult{}, err
	}
	absBasePath, relativeBasePath, err := s.resolveRelativePath(rootPath, request.BasePath, true)
	if err != nil {
		return ManifestResult{}, err
	}
	entries, rootHash, err := s.scanManifest(rootPath, absBasePath, relativeBasePath)
	if err != nil {
		return ManifestResult{}, err
	}
	return ManifestResult{
		Project: NodeProject{
			Id:   derefUint64(project.Id),
			Name: project.Name,
			Path: rootPath,
		},
		BasePath:          relativeBasePath,
		RootHash:          rootHash,
		PathCaseSensitive: filesystemCaseSensitive(rootPath),
		Entries:           entries,
	}, nil
}

// ReadLocalFile 读取本地项目中的单个文件内容。
func (s *CompareService) ReadLocalFile(ctx context.Context, request InternalFileRequest) (FileSide, error) {
	_, rootPath, err := s.getValidatedProject(ctx, request.ProjectId)
	if err != nil {
		return FileSide{}, err
	}
	absPath, _, err := s.resolveRelativePath(rootPath, request.RelativePath, false)
	if err != nil {
		return FileSide{}, err
	}
	return readFileSide(absPath)
}

// WriteLocalFile 按相对路径写入本地项目中的文件内容。
func (s *CompareService) WriteLocalFile(ctx context.Context, request InternalWriteFileRequest) error {
	project, rootPath, err := s.getValidatedProject(ctx, request.ProjectId)
	if err != nil {
		return err
	}
	requestedRelativePath := normalizeRelativePath(request.RelativePath)
	if requestedRelativePath != "" && !filesystemCaseSensitive(rootPath) {
		resolvedRelativePath, exists, exactMatch, resolveErr := resolveExistingRelativePathCaseInsensitive(rootPath, requestedRelativePath)
		if resolveErr != nil {
			return resolveErr
		}
		if exists && !exactMatch {
			return fmt.Errorf("目标工作站文件系统大小写不敏感，路径 %s 将与现有路径 %s 冲突，请先统一路径大小写后再同步", requestedRelativePath, resolvedRelativePath)
		}
	}
	absPath, relativePath, err := s.resolveRelativePath(rootPath, request.RelativePath, false)
	if err != nil {
		return err
	}
	var beforeFile FileSide
	if request.SyncLog != nil {
		beforeFile, err = readFileSide(absPath)
		if err != nil {
			return err
		}
	}
	content, err := base64.StdEncoding.DecodeString(request.ContentBase64)
	if err != nil {
		return fmt.Errorf("文件内容编码无效")
	}
	afterFile := buildFileSideFromContent(content)
	if request.ExpectedHash != "" {
		existingHash := ""
		fileExists := false
		if request.SyncLog != nil {
			existingHash = beforeFile.Hash
			fileExists = beforeFile.Exists
		} else {
			existingHash, err = s.currentFileHash(absPath)
			if err != nil {
				if !os.IsNotExist(err) {
					return err
				}
			} else {
				fileExists = true
			}
		}
		if !fileExists || existingHash != request.ExpectedHash {
			return fmt.Errorf("目标文件已变更，请重新比较后再执行同步")
		}
	}
	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return err
	}
	fileMode := os.FileMode(0644)
	if info, statErr := os.Stat(absPath); statErr == nil {
		if perm := info.Mode().Perm(); perm != 0 {
			fileMode = perm
		}
	}
	tempFile, err := os.CreateTemp(filepath.Dir(absPath), ".neko-*.tmp")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
	defer func() {
		_ = os.Remove(tempPath)
	}()
	if err := tempFile.Chmod(fileMode); err != nil {
		_ = tempFile.Close()
		return err
	}
	if _, err := tempFile.Write(content); err != nil {
		_ = tempFile.Close()
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempPath, absPath); err != nil {
		if runtime.GOOS == "windows" {
			if removeErr := os.Remove(absPath); removeErr != nil && !os.IsNotExist(removeErr) {
				return removeErr
			}
			if retryErr := os.Rename(tempPath, absPath); retryErr != nil {
				return retryErr
			}
		} else {
			return err
		}
	}
	if request.SyncLog != nil {
		logContext := s.normalizeSyncLogContext(project, relativePath, *request.SyncLog)
		if logErr := s.recordSyncLog(ctx, logContext, beforeFile, afterFile); logErr != nil {
			compareLog.Warn("记录目标项目同步日志失败: ", logErr)
		}
	}
	return nil
}

func readFileSide(absPath string) (FileSide, error) {
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return FileSide{Exists: false}, nil
		}
		return FileSide{}, err
	}
	if info.IsDir() {
		return FileSide{}, fmt.Errorf("目标路径是目录，无法读取文件内容")
	}
	content, err := os.ReadFile(absPath)
	if err != nil {
		return FileSide{}, err
	}
	return buildFileSideFromContent(content), nil
}

func buildFileSideFromContent(content []byte) FileSide {
	text := isTextContent(content)
	fileSide := FileSide{
		Exists:        true,
		Hash:          hashBytes(content),
		Text:          text,
		ContentBase64: base64.StdEncoding.EncodeToString(content),
		Size:          int64(len(content)),
	}
	if text {
		fileSide.NormalizedHash = normalizedTextHash(content)
		fileSide.Content = string(content)
	}
	return fileSide
}

// Compare 比较两个项目在指定目录范围内的差异。
func (s *CompareService) Compare(ctx context.Context, request CompareRequest) (CompareResult, error) {
	leftManifest, err := s.getManifest(ctx, request.LeftNodeId, request.LeftProjectId, request.BasePath)
	if err != nil {
		return CompareResult{}, err
	}
	rightManifest, err := s.getManifest(ctx, request.RightNodeId, request.RightProjectId, request.BasePath)
	if err != nil {
		return CompareResult{}, err
	}
	return s.compareManifest(leftManifest, rightManifest), nil
}

// BrowseProject 获取指定项目的浏览清单。
func (s *CompareService) BrowseProject(ctx context.Context, request ProjectBrowseRequest) (ManifestResult, error) {
	return s.getManifest(ctx, request.NodeId, request.ProjectId, request.BasePath)
}

// ReadProjectFile 读取指定项目中的单个文件。
func (s *CompareService) ReadProjectFile(ctx context.Context, request ProjectBrowseFileRequest) (FileSide, error) {
	return s.getFile(ctx, request.NodeId, request.ProjectId, request.Path)
}

// FileDiff 读取并计算指定文件在两个项目之间的文本差异。
func (s *CompareService) FileDiff(ctx context.Context, request FileDiffRequest) (FileDiffResult, error) {
	leftFile, err := s.getFile(ctx, request.LeftNodeId, request.LeftProjectId, request.Path)
	if err != nil {
		return FileDiffResult{}, err
	}
	rightFile, err := s.getFile(ctx, request.RightNodeId, request.RightProjectId, request.Path)
	if err != nil {
		return FileDiffResult{}, err
	}
	result := FileDiffResult{Path: request.Path, Left: leftFile, Right: rightFile}
	if leftFile.Text && rightFile.Text {
		result.Lines = buildDiffLines(leftFile.Content, rightFile.Content)
	}
	return result, nil
}

// ListProjectFileLogs 返回本机项目文件的修改日志列表，按修改时间倒序。
func (s *CompareService) ListProjectFileLogs(ctx context.Context, request ProjectSyncLogListRequest) (ProjectSyncLogListResult, error) {
	project, err := s.workSpaceService.GetProject(ctx, request.ProjectId)
	if err != nil {
		return ProjectSyncLogListResult{}, err
	}
	_ = project
	rows, err := s.projectSyncLogSvc.ListByTargetProjectAndPath(ctx, request.ProjectId, request.Path)
	if err != nil {
		return ProjectSyncLogListResult{}, err
	}
	items := make([]ProjectSyncLogListItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, ProjectSyncLogListItem{
			Id:                  derefUint64(row.Id),
			ChangeType:          row.ChangeType,
			ScopeType:           row.ScopeType,
			RelativePath:        row.RelativePath,
			TargetProjectId:     row.TargetProjectId,
			SourceNodeName:      row.SourceNodeName,
			SourceProjectName:   row.SourceProjectName,
			TargetNodeName:      row.TargetNodeName,
			TargetProjectName:   row.TargetProjectName,
			ExecutorNodeName:    row.ExecutorNodeName,
			ExecutorNodeAddress: row.ExecutorNodeAddress,
			OperatorIP:          row.OperatorIP,
			BeforeExists:        row.BeforeExists,
			BeforeHash:          row.BeforeHash,
			BeforeEncoding:      row.BeforeEncoding,
			AfterHash:           row.AfterHash,
			AfterEncoding:       row.AfterEncoding,
			OperatedAt:          formatLogTime(row.OperatedAt, row.ModifyTime),
		})
	}
	return ProjectSyncLogListResult{Count: len(items), Items: items}, nil
}

// ListSiteLogs 返回当前环境全部项目操作日志，按时间倒序。
func (s *CompareService) ListSiteLogs(ctx context.Context, request SiteProjectSyncLogListRequest) (SiteProjectSyncLogListResult, error) {
	pageNo := request.PageNo
	if pageNo < 1 {
		pageNo = 1
	}
	pageSize := request.PageSize
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}
	validProjectIds, err := s.workSpaceService.ListValidProjectIDs(ctx)
	if err != nil {
		return SiteProjectSyncLogListResult{}, err
	}
	total, err := s.projectSyncLogSvc.CountAllValid(ctx, validProjectIds, request.Keyword, request.ChangeType, request.ProjectName)
	if err != nil {
		return SiteProjectSyncLogListResult{}, err
	}
	rows, err := s.projectSyncLogSvc.ListAllValid(ctx, pageNo, pageSize, validProjectIds, request.Keyword, request.ChangeType, request.ProjectName)
	if err != nil {
		return SiteProjectSyncLogListResult{}, err
	}
	items := make([]ProjectSyncLogListItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, ProjectSyncLogListItem{
			Id:                  derefUint64(row.Id),
			ChangeType:          row.ChangeType,
			ScopeType:           row.ScopeType,
			RelativePath:        row.RelativePath,
			TargetProjectId:     row.TargetProjectId,
			SourceNodeName:      row.SourceNodeName,
			SourceProjectName:   row.SourceProjectName,
			TargetNodeName:      row.TargetNodeName,
			TargetProjectName:   row.TargetProjectName,
			ExecutorNodeName:    row.ExecutorNodeName,
			ExecutorNodeAddress: row.ExecutorNodeAddress,
			OperatorIP:          row.OperatorIP,
			BeforeExists:        row.BeforeExists,
			BeforeHash:          row.BeforeHash,
			BeforeEncoding:      row.BeforeEncoding,
			AfterHash:           row.AfterHash,
			AfterEncoding:       row.AfterEncoding,
			OperatedAt:          formatLogTime(row.OperatedAt, row.ModifyTime),
		})
	}
	return SiteProjectSyncLogListResult{PageNo: pageNo, PageSize: pageSize, Total: total, Items: items}, nil
}

func (s *CompareService) CleanupSiteLogs(ctx context.Context) (SiteProjectSyncLogCleanupResult, error) {
	rows, err := s.projectSyncLogSvc.ListAllValidForCleanup(ctx)
	if err != nil {
		return SiteProjectSyncLogCleanupResult{}, err
	}
	projects, err := s.workSpaceService.List(ctx)
	if err != nil {
		return SiteProjectSyncLogCleanupResult{}, err
	}
	projectMap := make(map[uint64]models.WorkSpace, len(projects))
	for _, project := range projects {
		projectID := derefUint64(project.Id)
		if projectID == 0 {
			continue
		}
		projectMap[projectID] = project
	}
	keepIDs := make([]uint64, 0)
	filesScanned := 0
	for index := 0; index < len(rows); {
		current := rows[index]
		filesScanned++
		project, _ := projectMap[current.TargetProjectId]
		keepLatest, keepID := s.shouldKeepLatestProjectSyncLog(project, current)
		if keepLatest && keepID != 0 {
			keepIDs = append(keepIDs, keepID)
		}
		keyProjectID := current.TargetProjectId
		keyPath := current.RelativePath
		index++
		for index < len(rows) && rows[index].TargetProjectId == keyProjectID && rows[index].RelativePath == keyPath {
			index++
		}
	}
	clearedLogs, vacuumResult, err := s.projectSyncLogSvc.HardDeleteLogsForCleanupAndVacuum(ctx, keepIDs)
	if err != nil {
		return SiteProjectSyncLogCleanupResult{}, err
	}
	return SiteProjectSyncLogCleanupResult{
		FilesScanned:            filesScanned,
		KeptLogs:                len(keepIDs),
		ClearedLogs:             clearedLogs,
		DatabaseSizeBeforeBytes: vacuumResult.Before.SizeBytes,
		DatabaseSizeBeforeLabel: vacuumResult.Before.SizeLabel,
		DatabaseSizeAfterBytes:  vacuumResult.After.SizeBytes,
		DatabaseSizeAfterLabel:  vacuumResult.After.SizeLabel,
		VacuumError:             vacuumResult.VacuumError,
	}, nil
}

// GetProjectFileLogDetail 返回单条文件修改日志详情。
func (s *CompareService) GetProjectFileLogDetail(ctx context.Context, request ProjectSyncLogDetailRequest) (ProjectSyncLogDetail, error) {
	row, err := s.projectSyncLogSvc.Exec(ctx).One(nil, "`id` = ? AND `Valid` = ?", request.LogId, 1)
	if err != nil {
		return ProjectSyncLogDetail{}, err
	}
	return newProjectSyncLogDetailFromRow(row)
}

type syncEndpointDescriptor struct {
	NodeID      uint64
	NodeName    string
	ProjectID   uint64
	ProjectName string
}

// Sync 将来源项目的文件同步到目标项目。
func (s *CompareService) Sync(ctx context.Context, request SyncRequest, operatorIP, executorNodeAddress string) (SyncResult, error) {
	sourceDescriptor, err := s.getSyncEndpointDescriptor(ctx, request.SourceNodeId, request.SourceProjectId)
	if err != nil {
		return SyncResult{}, err
	}
	targetDescriptor, err := s.getSyncEndpointDescriptor(ctx, request.TargetNodeId, request.TargetProjectId)
	if err != nil {
		return SyncResult{}, err
	}
	executorNodeName := s.nodeClientService.LocalNodeInfo().Name
	items, err := s.resolveSyncItems(ctx, request)
	if err != nil {
		return SyncResult{}, err
	}
	result := SyncResult{Failed: make([]SyncFailure, 0)}
	for _, item := range items {
		if item.EntryType != "file" {
			continue
		}
		sourceFile, readErr := s.getFile(ctx, request.SourceNodeId, request.SourceProjectId, item.Path)
		if readErr != nil {
			result.Failed = append(result.Failed, SyncFailure{Path: item.Path, Message: readErr.Error()})
			continue
		}
		targetFile, targetReadErr := s.getFile(ctx, request.TargetNodeId, request.TargetProjectId, item.Path)
		if targetReadErr != nil {
			result.Failed = append(result.Failed, SyncFailure{Path: item.Path, Message: targetReadErr.Error()})
			continue
		}
		if !sourceFile.Exists {
			result.Skipped++
			continue
		}
		logContext := s.buildSyncLogContext(request, item, sourceDescriptor, targetDescriptor, executorNodeName, executorNodeAddress, operatorIP)
		writeErr := s.writeFile(ctx, request.TargetNodeId, InternalWriteFileRequest{
			ProjectId:     request.TargetProjectId,
			RelativePath:  item.Path,
			ContentBase64: sourceFile.ContentBase64,
			ExpectedHash:  item.RightHash,
			SyncLog:       logContext,
		})
		if writeErr != nil {
			result.Failed = append(result.Failed, SyncFailure{Path: item.Path, Message: writeErr.Error()})
			continue
		}
		if request.TargetNodeId != 0 {
			if logErr := s.recordSyncLog(ctx, *logContext, targetFile, sourceFile); logErr != nil {
				compareLog.Warn("记录文件同步详细日志失败: ", logErr)
			}
		}
		result.Copied++
	}
	return result, nil
}

func (s *CompareService) resolveSyncItems(ctx context.Context, request SyncRequest) ([]CompareItem, error) {
	selectedPaths := normalizeSelectedSyncPaths(request.SelectedPaths)
	if len(selectedPaths) > 0 {
		return s.buildSyncItemsByPath(ctx, request, selectedPaths)
	}
	if request.ScopeType == "file" {
		normalizedPath := normalizeRelativePath(request.Path)
		if normalizedPath == "" {
			return []CompareItem{}, nil
		}
		return s.buildSyncItemsByPath(ctx, request, []string{normalizedPath})
	}
	compareResult, err := s.Compare(ctx, CompareRequest{
		LeftNodeId:     request.SourceNodeId,
		LeftProjectId:  request.SourceProjectId,
		RightNodeId:    request.TargetNodeId,
		RightProjectId: request.TargetProjectId,
	})
	if err != nil {
		return nil, err
	}
	items := s.filterSyncItems(compareResult.Items, request.ScopeType, request.Path)
	return filterSyncItemsBySelectedPaths(items, request.SelectedPaths), nil
}

func normalizeSelectedSyncPaths(paths []string) []string {
	seen := make(map[string]bool, len(paths))
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		normalizedPath := normalizeRelativePath(path)
		if normalizedPath == "" || seen[normalizedPath] {
			continue
		}
		seen[normalizedPath] = true
		result = append(result, normalizedPath)
	}
	return result
}

func (s *CompareService) buildSyncItemsByPath(ctx context.Context, request SyncRequest, paths []string) ([]CompareItem, error) {
	items := make([]CompareItem, 0, len(paths))
	for _, path := range paths {
		item, ok, err := s.buildSyncItemByPath(ctx, request, path)
		if err != nil {
			return nil, err
		}
		if ok {
			items = append(items, item)
		}
	}
	return items, nil
}

func (s *CompareService) buildSyncItemByPath(ctx context.Context, request SyncRequest, path string) (CompareItem, bool, error) {
	sourceFile, err := s.getFile(ctx, request.SourceNodeId, request.SourceProjectId, path)
	if err != nil {
		return CompareItem{}, false, err
	}
	targetFile, err := s.getFile(ctx, request.TargetNodeId, request.TargetProjectId, path)
	if err != nil {
		return CompareItem{}, false, err
	}
	item := CompareItem{
		Path:      path,
		Name:      filepath.Base(path),
		EntryType: "file",
		LeftHash:  sourceFile.Hash,
		RightHash: targetFile.Hash,
		LeftSize:  sourceFile.Size,
		RightSize: targetFile.Size,
	}
	status, ok := buildSyncItemStatus(sourceFile, targetFile)
	if !ok {
		return CompareItem{}, false, nil
	}
	item.Status = status
	return item, true, nil
}

func buildSyncItemStatus(sourceFile, targetFile FileSide) (string, bool) {
	switch {
	case sourceFile.Exists && !targetFile.Exists:
		return "left_only", true
	case !sourceFile.Exists && targetFile.Exists:
		return "right_only", true
	case !sourceFile.Exists && !targetFile.Exists:
		return "", false
	case sameFileSideContent(sourceFile, targetFile):
		return "", false
	default:
		return "file_changed", true
	}
}

func filterSyncItemsBySelectedPaths(items []CompareItem, selectedPaths []string) []CompareItem {
	if len(selectedPaths) == 0 {
		return items
	}
	selectedMap := make(map[string]bool, len(selectedPaths))
	for _, path := range selectedPaths {
		normalizedPath := normalizeRelativePath(path)
		if normalizedPath != "" {
			selectedMap[normalizedPath] = true
		}
	}
	if len(selectedMap) == 0 {
		return []CompareItem{}
	}
	filtered := make([]CompareItem, 0, len(selectedMap))
	for _, item := range items {
		if selectedMap[item.Path] {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func (s *CompareService) getSyncEndpointDescriptor(ctx context.Context, nodeID uint64, projectID uint64) (syncEndpointDescriptor, error) {
	if nodeID == 0 {
		project, err := s.workSpaceService.GetProject(ctx, projectID)
		if err != nil {
			return syncEndpointDescriptor{}, err
		}
		nodeName := strings.TrimSpace(s.nodeClientService.LocalNodeInfo().Name)
		if nodeName == "" {
			nodeName = "本机节点"
		}
		return syncEndpointDescriptor{
			NodeID:      nodeID,
			NodeName:    nodeName,
			ProjectID:   projectID,
			ProjectName: project.Name,
		}, nil
	}
	node, err := s.serverNodeService.GetNode(ctx, nodeID)
	if err != nil {
		return syncEndpointDescriptor{}, err
	}
	projects, err := s.nodeClientService.ListProjects(ctx, nodeID)
	if err != nil {
		return syncEndpointDescriptor{}, err
	}
	projectName := fmt.Sprintf("项目 #%d", projectID)
	for _, project := range projects {
		if project.Id == projectID {
			projectName = project.Name
			break
		}
	}
	return syncEndpointDescriptor{
		NodeID:      nodeID,
		NodeName:    node.Name,
		ProjectID:   projectID,
		ProjectName: projectName,
	}, nil
}

func (s *CompareService) buildSyncLogContext(request SyncRequest, item CompareItem, sourceDescriptor, targetDescriptor syncEndpointDescriptor, executorNodeName, executorNodeAddress, operatorIP string) *SyncLogContext {
	return &SyncLogContext{
		ChangeType:          item.Status,
		ScopeType:           request.ScopeType,
		RelativePath:        item.Path,
		SourceNodeId:        sourceDescriptor.NodeID,
		SourceNodeName:      sourceDescriptor.NodeName,
		SourceProjectId:     sourceDescriptor.ProjectID,
		SourceProjectName:   sourceDescriptor.ProjectName,
		TargetNodeId:        targetDescriptor.NodeID,
		TargetNodeName:      targetDescriptor.NodeName,
		TargetProjectId:     targetDescriptor.ProjectID,
		TargetProjectName:   targetDescriptor.ProjectName,
		ExecutorNodeName:    executorNodeName,
		ExecutorNodeAddress: strings.TrimSpace(executorNodeAddress),
		OperatorIP:          strings.TrimSpace(operatorIP),
	}
}

func (s *CompareService) normalizeSyncLogContext(project models.WorkSpace, relativePath string, logContext SyncLogContext) SyncLogContext {
	logContext.RelativePath = normalizeRelativePath(relativePath)
	logContext.TargetNodeId = 0
	logContext.TargetProjectId = derefUint64(project.Id)
	logContext.TargetProjectName = project.Name
	localNodeName := strings.TrimSpace(s.nodeClientService.LocalNodeInfo().Name)
	if localNodeName == "" {
		localNodeName = "本地"
	}
	logContext.TargetNodeName = localNodeName
	logContext.ExecutorNodeAddress = strings.TrimSpace(logContext.ExecutorNodeAddress)
	return logContext
}

func (s *CompareService) recordSyncLog(ctx context.Context, logContext SyncLogContext, beforeFile, afterFile FileSide) error {
	beforeSnapshot, afterSnapshot, diffAlgorithm, err := buildProjectSyncLogSnapshots(beforeFile, afterFile)
	if err != nil {
		return err
	}
	now := time.Now()
	return s.projectSyncLogSvc.RecordLog(ctx, models.ProjectSyncLog{
		ChangeType:          logContext.ChangeType,
		ScopeType:           logContext.ScopeType,
		RelativePath:        logContext.RelativePath,
		SourceNodeId:        logContext.SourceNodeId,
		SourceNodeName:      logContext.SourceNodeName,
		SourceProjectId:     logContext.SourceProjectId,
		SourceProjectName:   logContext.SourceProjectName,
		TargetNodeId:        logContext.TargetNodeId,
		TargetNodeName:      logContext.TargetNodeName,
		TargetProjectId:     logContext.TargetProjectId,
		TargetProjectName:   logContext.TargetProjectName,
		ExecutorNodeName:    logContext.ExecutorNodeName,
		ExecutorNodeAddress: strings.TrimSpace(logContext.ExecutorNodeAddress),
		OperatorIP:          strings.TrimSpace(logContext.OperatorIP),
		BeforeExists:        beforeFile.Exists,
		BeforeHash:          beforeFile.Hash,
		BeforeEncoding:      beforeSnapshot.Encoding,
		BeforeStorageKind:   beforeSnapshot.StorageKind,
		BeforeContentSize:   beforeSnapshot.ContentSize,
		BeforeOmittedReason: beforeSnapshot.OmittedReason,
		BeforeContent:       beforeSnapshot.Content,
		AfterHash:           afterFile.Hash,
		AfterEncoding:       afterSnapshot.Encoding,
		AfterStorageKind:    afterSnapshot.StorageKind,
		AfterContentSize:    afterSnapshot.ContentSize,
		AfterOmittedReason:  afterSnapshot.OmittedReason,
		AfterContent:        afterSnapshot.Content,
		DiffAlgorithm:       diffAlgorithm,
		OperatedAt:          &now,
	})
}

func formatLogTime(operatedAt, modifyTime *time.Time) string {
	if operatedAt != nil {
		return operatedAt.Format(time.DateTime)
	}
	if modifyTime != nil {
		return modifyTime.Format(time.DateTime)
	}
	return ""
}

func (s *CompareService) shouldKeepLatestProjectSyncLog(project models.WorkSpace, latest models.ProjectSyncLog) (bool, uint64) {
	if project.BaseModel == nil || project.Id == nil {
		return false, 0
	}
	if latest.BaseModel == nil || latest.Id == nil {
		return false, 0
	}
	projectID := derefUint64(project.Id)
	if projectID == 0 || project.DiskDeleted {
		return false, 0
	}
	validatedRootPath, err := s.projectAccess.ValidateExistingProjectPath(project.Path)
	if err != nil {
		return false, 0
	}
	absPath, _, err := s.resolveRelativePath(validatedRootPath, latest.RelativePath, false)
	if err != nil {
		return false, 0
	}
	info, err := os.Stat(absPath)
	if err != nil || info.IsDir() {
		return false, 0
	}
	return true, derefUint64(latest.Id)
}

// sameManifestEntryContent 只按类型和内容摘要判断两个清单项是否一致，不把保存时间当作差异。
func sameManifestEntryContent(left, right ManifestEntry) bool {
	return left.EntryType == right.EntryType && manifestEntryCompareHash(left) == manifestEntryCompareHash(right)
}

func (s *CompareService) compareManifest(left, right ManifestResult) CompareResult {
	leftMap := make(map[string]ManifestEntry, len(left.Entries))
	rightMap := make(map[string]ManifestEntry, len(right.Entries))
	for _, entry := range left.Entries {
		leftMap[entry.RelativePath] = entry
	}
	for _, entry := range right.Entries {
		rightMap[entry.RelativePath] = entry
	}
	result := CompareResult{
		Summary:                CompareSummary{RootHashLeft: left.RootHash, RootHashRight: right.RootHash},
		Items:                  make([]CompareItem, 0),
		LeftPathCaseSensitive:  left.PathCaseSensitive,
		RightPathCaseSensitive: right.PathCaseSensitive,
	}
	matchedLeftPaths := make(map[string]bool, len(leftMap))
	matchedRightPaths := make(map[string]bool, len(rightMap))
	pairs := make([]manifestComparePair, 0)

	for path, leftEntry := range leftMap {
		rightEntry, ok := rightMap[path]
		if !ok {
			continue
		}
		pairs = append(pairs, manifestComparePair{Path: path, Left: leftEntry, Right: rightEntry})
		matchedLeftPaths[path] = true
		matchedRightPaths[path] = true
	}

	if !left.PathCaseSensitive || !right.PathCaseSensitive {
		fuzzyPairs := matchManifestEntriesByFoldedPath(leftMap, rightMap, matchedLeftPaths, matchedRightPaths, left.PathCaseSensitive, right.PathCaseSensitive)
		for _, pair := range fuzzyPairs {
			pairs = append(pairs, pair)
			matchedLeftPaths[pair.Left.RelativePath] = true
			matchedRightPaths[pair.Right.RelativePath] = true
		}
	}

	for _, pair := range pairs {
		if sameManifestEntryContent(pair.Left, pair.Right) {
			result.Summary.SameCount++
			continue
		}
		item := buildCompareItem(pair.Path, pair.Left, true, pair.Right, true)
		result.Items = append(result.Items, item)
		applyCompareSummary(&result.Summary, item)
	}

	for _, entry := range left.Entries {
		if matchedLeftPaths[entry.RelativePath] {
			continue
		}
		item := buildCompareItem(entry.RelativePath, entry, true, ManifestEntry{}, false)
		result.Items = append(result.Items, item)
		applyCompareSummary(&result.Summary, item)
	}
	for _, entry := range right.Entries {
		if matchedRightPaths[entry.RelativePath] {
			continue
		}
		item := buildCompareItem(entry.RelativePath, ManifestEntry{}, false, entry, true)
		result.Items = append(result.Items, item)
		applyCompareSummary(&result.Summary, item)
	}
	sort.Slice(result.Items, func(i, j int) bool {
		return result.Items[i].Path < result.Items[j].Path
	})
	result.Summary.Total = result.Summary.SameCount + len(result.Items)
	return result
}

type manifestComparePair struct {
	Path  string
	Left  ManifestEntry
	Right ManifestEntry
}

func matchManifestEntriesByFoldedPath(leftMap, rightMap map[string]ManifestEntry, matchedLeftPaths, matchedRightPaths map[string]bool, leftCaseSensitive, rightCaseSensitive bool) []manifestComparePair {
	leftGroups := groupManifestEntriesByCompareKey(leftMap, matchedLeftPaths)
	rightGroups := groupManifestEntriesByCompareKey(rightMap, matchedRightPaths)
	keys := make([]string, 0)
	for key := range leftGroups {
		if _, ok := rightGroups[key]; ok {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	pairs := make([]manifestComparePair, 0, len(keys))
	for _, key := range keys {
		leftEntries := leftGroups[key]
		rightEntries := rightGroups[key]
		if len(leftEntries) != 1 || len(rightEntries) != 1 {
			continue
		}
		leftEntry := leftEntries[0]
		rightEntry := rightEntries[0]
		pairs = append(pairs, manifestComparePair{
			Path:  pickManifestRepresentativePath(leftEntry.RelativePath, rightEntry.RelativePath, leftCaseSensitive, rightCaseSensitive),
			Left:  leftEntry,
			Right: rightEntry,
		})
	}
	return pairs
}

func groupManifestEntriesByCompareKey(entries map[string]ManifestEntry, matchedPaths map[string]bool) map[string][]ManifestEntry {
	grouped := make(map[string][]ManifestEntry)
	for path, entry := range entries {
		if matchedPaths[path] {
			continue
		}
		key := relativePathCompareKey(path, false)
		grouped[key] = append(grouped[key], entry)
	}
	return grouped
}

func pickManifestRepresentativePath(leftPath, rightPath string, leftCaseSensitive, rightCaseSensitive bool) string {
	switch {
	case leftPath == rightPath:
		return leftPath
	case leftCaseSensitive && !rightCaseSensitive:
		return leftPath
	case !leftCaseSensitive && rightCaseSensitive:
		return rightPath
	default:
		return leftPath
	}
}

func buildCompareItem(path string, leftEntry ManifestEntry, leftOk bool, rightEntry ManifestEntry, rightOk bool) CompareItem {
	item := CompareItem{Path: path, Name: filepath.Base(path)}
	if leftOk {
		item.EntryType = leftEntry.EntryType
		item.LeftHash = leftEntry.Hash
		item.LeftSize = leftEntry.Size
	}
	if rightOk {
		if item.EntryType == "" {
			item.EntryType = rightEntry.EntryType
		}
		item.RightHash = rightEntry.Hash
		item.RightSize = rightEntry.Size
	}
	switch {
	case leftOk && !rightOk:
		item.Status = "left_only"
	case !leftOk && rightOk:
		item.Status = "right_only"
	case leftEntry.EntryType != rightEntry.EntryType:
		item.Status = "type_changed"
	case item.EntryType == "directory":
		item.Status = "directory_changed"
	default:
		item.Status = "file_changed"
	}
	return item
}

func applyCompareSummary(summary *CompareSummary, item CompareItem) {
	switch item.Status {
	case "left_only":
		summary.LeftOnly++
	case "right_only":
		summary.RightOnly++
	case "type_changed":
		if item.EntryType == "directory" {
			summary.DifferentDirectories++
		} else {
			summary.DifferentFiles++
		}
	case "directory_changed":
		summary.DifferentDirectories++
	default:
		summary.DifferentFiles++
	}
}

// sameFileSideContent 仅按存在性、文本/二进制内容摘要判断文件是否相同，忽略保存日期差异。
func sameFileSideContent(left, right FileSide) bool {
	if left.Exists != right.Exists {
		return false
	}
	if !left.Exists {
		return true
	}
	return fileSideCompareHash(left) == fileSideCompareHash(right)
}

func (s *CompareService) getManifest(ctx context.Context, nodeID, projectID uint64, basePath string) (ManifestResult, error) {
	request := InternalManifestRequest{ProjectId: projectID, BasePath: basePath}
	if nodeID == 0 {
		return s.BuildLocalManifest(ctx, request)
	}
	return s.nodeClientService.BuildManifest(ctx, nodeID, request)
}

func (s *CompareService) getFile(ctx context.Context, nodeID, projectID uint64, path string) (FileSide, error) {
	request := InternalFileRequest{ProjectId: projectID, RelativePath: path}
	if nodeID == 0 {
		return s.ReadLocalFile(ctx, request)
	}
	return s.nodeClientService.ReadFile(ctx, nodeID, request)
}

func (s *CompareService) writeFile(ctx context.Context, nodeID uint64, request InternalWriteFileRequest) error {
	if nodeID == 0 {
		return s.WriteLocalFile(ctx, request)
	}
	return s.nodeClientService.WriteFile(ctx, nodeID, request)
}

func (s *CompareService) filterSyncItems(items []CompareItem, scopeType, path string) []CompareItem {
	normalizedPath := normalizeRelativePath(path)
	filtered := make([]CompareItem, 0)
	for _, item := range items {
		switch scopeType {
		case "file":
			if item.Path == normalizedPath {
				filtered = append(filtered, item)
			}
		case "directory":
			if normalizedPath == "" || item.Path == normalizedPath || strings.HasPrefix(item.Path, normalizedPath+"/") {
				filtered = append(filtered, item)
			}
		default:
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func (s *CompareService) getValidatedProject(ctx context.Context, projectID uint64) (models.WorkSpace, string, error) {
	project, err := s.workSpaceService.GetProject(ctx, projectID)
	if err != nil {
		return project, "", err
	}
	validatedPath, err := s.projectAccess.ValidateExistingProjectPath(project.Path)
	if err != nil {
		return project, "", err
	}
	return project, validatedPath, nil
}

func (s *CompareService) resolveRelativePath(rootPath, input string, requireExists bool) (string, string, error) {
	relativePath := normalizeRelativePath(input)
	absPath := rootPath
	if relativePath != "" {
		absPath = filepath.Join(rootPath, filepath.FromSlash(relativePath))
	}
	absPath = filepath.Clean(absPath)
	if !isSameOrWithinPath(rootPath, absPath) {
		return "", "", fmt.Errorf("路径超出项目根目录范围")
	}
	if !filesystemCaseSensitive(rootPath) && relativePath != "" {
		resolvedRelativePath, exists, _, err := resolveExistingRelativePathCaseInsensitive(rootPath, relativePath)
		if err != nil {
			return "", "", err
		}
		if exists {
			relativePath = resolvedRelativePath
			absPath = filepath.Clean(filepath.Join(rootPath, filepath.FromSlash(relativePath)))
		} else if requireExists {
			return "", "", fmt.Errorf("目标路径不存在")
		}
	}
	if requireExists {
		if _, err := os.Stat(absPath); err != nil {
			return "", "", fmt.Errorf("目标路径不存在")
		}
	}
	return absPath, relativePath, nil
}

type projectIgnoreMatcher struct {
	rootPath   string
	gitignore  *ignore.GitIgnore
	alwaysSkip map[string]bool
}

func newProjectIgnoreMatcher(rootPath string) *projectIgnoreMatcher {
	matcher := &projectIgnoreMatcher{
		rootPath: rootPath,
		alwaysSkip: map[string]bool{
			".git":      true,
			".DS_Store": true,
		},
	}
	gitignorePath := filepath.Join(rootPath, ".gitignore")
	if info, err := os.Stat(gitignorePath); err == nil && !info.IsDir() {
		compiled, compileErr := ignore.CompileIgnoreFile(gitignorePath)
		if compileErr == nil {
			matcher.gitignore = compiled
		}
	}
	return matcher
}

func (m *projectIgnoreMatcher) ShouldIgnore(relativePath string, isDir bool) bool {
	trimmedPath := strings.TrimSpace(filepath.ToSlash(relativePath))
	if trimmedPath == "" {
		return false
	}
	baseName := filepath.Base(trimmedPath)
	if m.alwaysSkip[baseName] {
		return true
	}
	if m.gitignore == nil {
		return false
	}
	if m.gitignore.MatchesPath(trimmedPath) {
		return true
	}
	if isDir && m.gitignore.MatchesPath(trimmedPath+"/") {
		return true
	}
	return false
}

func (s *CompareService) scanManifest(rootPath, absBasePath, relativeBasePath string) ([]ManifestEntry, string, error) {
	ignoreMatcher := newProjectIgnoreMatcher(rootPath)
	_, entries, err := s.walkEntry(absBasePath, relativeBasePath, ignoreMatcher)
	if err != nil {
		return nil, "", err
	}
	if len(entries) == 0 {
		return []ManifestEntry{}, hashBytes([]byte("empty")), nil
	}
	return entries, entries[0].Hash, nil
}

func (s *CompareService) walkEntry(absPath, relativePath string, ignoreMatcher *projectIgnoreMatcher) (ManifestEntry, []ManifestEntry, error) {
	info, err := os.Stat(absPath)
	if err != nil {
		return ManifestEntry{}, nil, err
	}
	name := filepath.Base(absPath)
	if ignoreMatcher != nil && ignoreMatcher.ShouldIgnore(relativePath, info.IsDir()) {
		return ManifestEntry{}, nil, os.ErrNotExist
	}
	if !info.IsDir() {
		content, readErr := os.ReadFile(absPath)
		if readErr != nil {
			return ManifestEntry{}, nil, readErr
		}
		text := isTextContent(content)
		entry := ManifestEntry{RelativePath: relativePath, Name: name, EntryType: "file", Size: info.Size(), ModifyTime: info.ModTime().Unix(), Hash: hashBytes(content), Text: text}
		if text {
			entry.NormalizedHash = normalizedTextHash(content)
		}
		return entry, []ManifestEntry{entry}, nil
	}
	children, err := os.ReadDir(absPath)
	if err != nil {
		return ManifestEntry{}, nil, err
	}
	sort.Slice(children, func(i, j int) bool { return children[i].Name() < children[j].Name() })
	immediateHashes := make([]string, 0, len(children))
	entries := make([]ManifestEntry, 0)
	for _, child := range children {
		childRelative := child.Name()
		if relativePath != "" {
			childRelative = relativePath + "/" + child.Name()
		}
		childEntry, childEntries, childErr := s.walkEntry(filepath.Join(absPath, child.Name()), childRelative, ignoreMatcher)
		if childErr != nil {
			if os.IsNotExist(childErr) {
				continue
			}
			return ManifestEntry{}, nil, childErr
		}
		immediateHashes = append(immediateHashes, childEntry.EntryType+":"+childEntry.Name+":"+manifestEntryCompareHash(childEntry))
		entries = append(entries, childEntries...)
	}
	entry := ManifestEntry{RelativePath: relativePath, Name: name, EntryType: "directory", ModifyTime: info.ModTime().Unix(), Hash: hashBytes([]byte(strings.Join(immediateHashes, "\n")))}
	return entry, append([]ManifestEntry{entry}, entries...), nil
}

func (s *CompareService) currentFileHash(absPath string) (string, error) {
	content, err := os.ReadFile(absPath)
	if err != nil {
		return "", err
	}
	return hashBytes(content), nil
}

func normalizeRelativePath(input string) string {
	trimmed := strings.TrimSpace(strings.ReplaceAll(input, "\\", "/"))
	trimmed = strings.TrimPrefix(trimmed, "/")
	if trimmed == "" || trimmed == "." {
		return ""
	}
	cleaned := filepath.ToSlash(filepath.Clean(trimmed))
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return ""
	}
	return cleaned
}

func hashBytes(content []byte) string {
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
}

func normalizedTextHash(content []byte) string {
	return hashBytes(normalizeLineEndingsBytes(content))
}

func normalizeLineEndingsBytes(content []byte) []byte {
	if len(content) == 0 {
		return content
	}
	normalized := bytes.ReplaceAll(content, []byte("\r\n"), []byte("\n"))
	normalized = bytes.ReplaceAll(normalized, []byte("\r"), []byte("\n"))
	return normalized
}

func manifestEntryCompareHash(entry ManifestEntry) string {
	if entry.EntryType == "file" && entry.Text && strings.TrimSpace(entry.NormalizedHash) != "" {
		return entry.NormalizedHash
	}
	return entry.Hash
}

func fileSideCompareHash(fileSide FileSide) string {
	if fileSide.Text && strings.TrimSpace(fileSide.NormalizedHash) != "" {
		return fileSide.NormalizedHash
	}
	return fileSide.Hash
}

func isTextContent(content []byte) bool {
	if len(content) == 0 {
		return true
	}
	sample := content
	if len(sample) > 8192 {
		sample = sample[:8192]
	}
	suspiciousControls := 0
	for _, b := range sample {
		if b == 0 {
			return false
		}
		if isSuspiciousControlByte(b) {
			suspiciousControls++
		}
	}
	if suspiciousControls == 0 {
		return true
	}
	return suspiciousControls*100 <= len(sample)*10
}

func isSuspiciousControlByte(value byte) bool {
	switch value {
	case '\n', '\r', '\t', '\f', '\b':
		return false
	}
	return value < 0x20 || value == 0x7f
}

func buildDiffLines(left, right string) []DiffLine {
	leftLines := splitLines(left)
	rightLines := splitLines(right)
	if len(leftLines)*len(rightLines) > 2000000 {
		return buildFallbackDiffLines(leftLines, rightLines)
	}
	dp := make([][]int, len(leftLines)+1)
	for i := range dp {
		dp[i] = make([]int, len(rightLines)+1)
	}
	for i := len(leftLines) - 1; i >= 0; i-- {
		for j := len(rightLines) - 1; j >= 0; j-- {
			if leftLines[i] == rightLines[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else if dp[i+1][j] >= dp[i][j+1] {
				dp[i][j] = dp[i+1][j]
			} else {
				dp[i][j] = dp[i][j+1]
			}
		}
	}
	result := make([]DiffLine, 0)
	leftIndex, rightIndex := 0, 0
	leftLineNo, rightLineNo := 1, 1
	for leftIndex < len(leftLines) && rightIndex < len(rightLines) {
		if leftLines[leftIndex] == rightLines[rightIndex] {
			result = append(result, DiffLine{Type: "same", LeftNumber: leftLineNo, RightNumber: rightLineNo, LeftText: leftLines[leftIndex], RightText: rightLines[rightIndex]})
			leftIndex++
			rightIndex++
			leftLineNo++
			rightLineNo++
			continue
		}
		if dp[leftIndex+1][rightIndex] == dp[leftIndex][rightIndex+1] {
			result = append(result, DiffLine{Type: "change", LeftNumber: leftLineNo, RightNumber: rightLineNo, LeftText: leftLines[leftIndex], RightText: rightLines[rightIndex]})
			leftIndex++
			rightIndex++
			leftLineNo++
			rightLineNo++
			continue
		}
		if dp[leftIndex+1][rightIndex] > dp[leftIndex][rightIndex+1] {
			result = append(result, DiffLine{Type: "delete", LeftNumber: leftLineNo, LeftText: leftLines[leftIndex]})
			leftIndex++
			leftLineNo++
			continue
		}
		result = append(result, DiffLine{Type: "add", RightNumber: rightLineNo, RightText: rightLines[rightIndex]})
		rightIndex++
		rightLineNo++
	}
	for leftIndex < len(leftLines) {
		result = append(result, DiffLine{Type: "delete", LeftNumber: leftLineNo, LeftText: leftLines[leftIndex]})
		leftIndex++
		leftLineNo++
	}
	for rightIndex < len(rightLines) {
		result = append(result, DiffLine{Type: "add", RightNumber: rightLineNo, RightText: rightLines[rightIndex]})
		rightIndex++
		rightLineNo++
	}
	return result
}

func buildFallbackDiffLines(leftLines, rightLines []string) []DiffLine {
	maxLen := len(leftLines)
	if len(rightLines) > maxLen {
		maxLen = len(rightLines)
	}
	result := make([]DiffLine, 0, maxLen)
	for index := 0; index < maxLen; index++ {
		line := DiffLine{}
		if index < len(leftLines) {
			line.LeftNumber = index + 1
			line.LeftText = leftLines[index]
		}
		if index < len(rightLines) {
			line.RightNumber = index + 1
			line.RightText = rightLines[index]
		}
		switch {
		case line.LeftText == line.RightText:
			line.Type = "same"
		case line.LeftText == "":
			line.Type = "add"
		case line.RightText == "":
			line.Type = "delete"
		default:
			line.Type = "change"
		}
		result = append(result, line)
	}
	return result
}

func splitLines(content string) []string {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	if normalized == "" {
		return []string{}
	}
	return strings.Split(normalized, "\n")
}
