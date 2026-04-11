package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"neko-tool/internal/config"
	"neko-tool/internal/models"
	internalRepo "neko-tool/internal/repository"
	"neko-tool/pkg/core/tx"
	pkgModels "neko-tool/pkg/models"
	"neko-tool/pkg/until"
)

var projectBrowserLog = until.Log

type ProjectBrowserCacheService struct {
	workSpaceService *WorkSpaceService
	projectAccess    *ProjectAccessService
	cacheRepo        *internalRepo.ProjectNodeCacheRepository
	projectSyncLog   *ProjectSyncLogService
	nodeClient       *NodeClientService
	configManager    *config.ApplicationConfigManager
	projectLocks     sync.Map
}

// NewProjectBrowserCacheService 构造项目浏览缓存服务。
func NewProjectBrowserCacheService(workSpaceService *WorkSpaceService, projectAccess *ProjectAccessService, cacheRepo *internalRepo.ProjectNodeCacheRepository, projectSyncLog *ProjectSyncLogService, nodeClient *NodeClientService, configManager *config.ApplicationConfigManager) *ProjectBrowserCacheService {
	return &ProjectBrowserCacheService{
		workSpaceService: workSpaceService,
		projectAccess:    projectAccess,
		cacheRepo:        cacheRepo,
		projectSyncLog:   projectSyncLog,
		nodeClient:       nodeClient,
		configManager:    configManager,
	}
}

// BrowseProject 从缓存中读取项目目录信息，必要时自动触发一次刷新。
func (s *ProjectBrowserCacheService) BrowseProject(ctx context.Context, request ProjectBrowseRequest) (ManifestResult, error) {
	project, err := s.workSpaceService.GetProject(ctx, request.ProjectId)
	if err != nil {
		return ManifestResult{}, err
	}
	if !project.CacheInitialized {
		if _, refreshErr := s.RefreshProject(ctx, request); refreshErr != nil {
			return ManifestResult{}, refreshErr
		}
		project, err = s.workSpaceService.GetProject(ctx, request.ProjectId)
		if err != nil {
			return ManifestResult{}, err
		}
	}
	return s.buildManifestFromCache(ctx, project, request.BasePath, request.Depth)
}

// RefreshProject 重新扫描项目目录并回写缓存后返回浏览结果。
func (s *ProjectBrowserCacheService) RefreshProject(ctx context.Context, request ProjectBrowseRequest) (ManifestResult, error) {
	project, err := s.workSpaceService.GetProject(ctx, request.ProjectId)
	if err != nil {
		return ManifestResult{}, err
	}
	lock := s.getProjectLock(request.ProjectId)
	lock.Lock()
	defer lock.Unlock()
	if err := s.refreshProjectCache(ctx, project); err != nil {
		return ManifestResult{}, err
	}
	project, err = s.workSpaceService.GetProject(ctx, request.ProjectId)
	if err != nil {
		return ManifestResult{}, err
	}
	if verifyErr := s.reconcileProjectLogs(ctx, project); verifyErr != nil {
		projectBrowserLog.Warn("刷新项目后校验修改日志失败: ", verifyErr)
	}
	return s.buildManifestFromCache(ctx, project, request.BasePath, request.Depth)
}

// DeleteProjectPath 删除项目中的文件或目录，并同步更新缓存与本地日志。
func (s *ProjectBrowserCacheService) DeleteProjectPath(ctx context.Context, request ProjectBrowseDeleteRequest) error {
	project, err := s.workSpaceService.GetProject(ctx, request.ProjectId)
	if err != nil {
		return err
	}
	lock := s.getProjectLock(request.ProjectId)
	lock.Lock()
	defer lock.Unlock()
	validatedPath, err := s.projectAccess.ValidateExistingProjectPath(project.Path)
	if err != nil {
		return err
	}
	relativePath := normalizeRelativePath(request.Path)
	if isProjectRootRelativePath(relativePath) {
		return fmt.Errorf("不允许删除项目根目录")
	}
	absPath, _, err := s.resolveProjectRelativePath(validatedPath, relativePath, true)
	if err != nil {
		return err
	}
	if samePath(validatedPath, absPath) {
		return fmt.Errorf("不允许删除项目根目录")
	}
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("目标路径不存在")
		}
		return err
	}
	affectedFiles, err := collectAffectedFilePathsFromDisk(absPath, relativePath, info)
	if err != nil {
		return err
	}
	if info.IsDir() {
		if err := os.RemoveAll(absPath); err != nil {
			return err
		}
	} else {
		if err := os.Remove(absPath); err != nil {
			return err
		}
	}
	if err := s.markPathDeleted(ctx, request.ProjectId, relativePath); err != nil {
		return err
	}
	return s.recordDeletedFileLogs(ctx, project, affectedFiles)
}

// ReadProjectFile 始终从磁盘读取项目文件内容，并在缺失时回写删除标记。
func (s *ProjectBrowserCacheService) ReadProjectFile(ctx context.Context, request ProjectBrowseFileRequest) (FileSide, error) {
	project, err := s.workSpaceService.GetProject(ctx, request.ProjectId)
	if err != nil {
		return FileSide{}, err
	}
	relativePath := normalizeRelativePath(request.Path)
	rootPath := filepath.Clean(project.Path)
	if project.DiskDeleted {
		return FileSide{Exists: false, Deleted: true}, nil
	}
	absPath, _, err := s.resolveProjectRelativePath(rootPath, relativePath, false)
	if err != nil {
		return FileSide{}, err
	}
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			if markErr := s.markPathDeleted(ctx, request.ProjectId, relativePath); markErr != nil {
				projectBrowserLog.Warn("标记文件已删除失败: ", markErr)
			}
			result := FileSide{Exists: false, Deleted: true}
			if verifyErr := s.ensureFileLog(ctx, project, relativePath, result); verifyErr != nil {
				projectBrowserLog.Warn("补充本地删除日志失败: ", verifyErr)
			}
			return result, nil
		}
		return FileSide{}, err
	}
	if info.IsDir() {
		return FileSide{}, fmt.Errorf("目标路径是目录，无法读取文件内容")
	}
	content, err := os.ReadFile(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			if markErr := s.markPathDeleted(ctx, request.ProjectId, relativePath); markErr != nil {
				projectBrowserLog.Warn("标记文件已删除失败: ", markErr)
			}
			result := FileSide{Exists: false, Deleted: true}
			if verifyErr := s.ensureFileLog(ctx, project, relativePath, result); verifyErr != nil {
				projectBrowserLog.Warn("补充本地删除日志失败: ", verifyErr)
			}
			return result, nil
		}
		return FileSide{}, err
	}
	result := FileSide{
		Exists:        true,
		Deleted:       false,
		Hash:          hashBytes(content),
		Text:          isTextContent(content),
		ContentBase64: base64Encode(content),
		Size:          int64(len(content)),
	}
	if result.Text {
		result.NormalizedHash = normalizedTextHash(content)
		result.Content = string(content)
	}
	if verifyErr := s.ensureFileLog(ctx, project, relativePath, result); verifyErr != nil {
		projectBrowserLog.Warn("补充本地修改日志失败: ", verifyErr)
	}
	return result, nil
}

func (s *ProjectBrowserCacheService) refreshProjectCache(ctx context.Context, project models.WorkSpace) error {
	ctx, _, deliver := tx.GetScope(ctx, tx.NewScope)
	defer deliver()
	now := time.Now()
	validatedPath, err := s.projectAccess.ValidateExistingProjectPath(project.Path)
	if err != nil {
		if markErr := s.markProjectDeletedInTx(ctx, derefUint64(project.Id), now); markErr != nil {
			return markErr
		}
		return nil
	}
	entries, rootHash, scanErr := s.scanProjectNodes(validatedPath, now)
	if scanErr != nil {
		return scanErr
	}
	if err := s.cacheRepoFor(ctx).ReplaceProjectNodes(derefUint64(project.Id), entries, now); err != nil {
		return err
	}
	updates := map[string]any{
		"path":              validatedPath,
		"hash":              rootHash,
		"cache_initialized": true,
		"disk_deleted":      false,
		"last_scan_at":      now,
	}
	return s.workSpaceService.Exec(ctx).
		Where("`id` = ? AND `Valid` = ?", derefUint64(project.Id), 1).
		Updates(updates).Error
}

func (s *ProjectBrowserCacheService) markProjectDeletedInTx(ctx context.Context, projectId uint64, scannedAt time.Time) error {
	if err := s.cacheRepoFor(ctx).MarkProjectDeleted(projectId, scannedAt); err != nil {
		return err
	}
	updates := map[string]any{
		"cache_initialized": true,
		"disk_deleted":      true,
		"last_scan_at":      scannedAt,
		"hash":              "",
	}
	return s.workSpaceService.Exec(ctx).
		Where("`id` = ? AND `Valid` = ?", projectId, 1).
		Updates(updates).Error
}

func (s *ProjectBrowserCacheService) markPathDeleted(ctx context.Context, projectId uint64, relativePath string) error {
	ctx, _, deliver := tx.GetScope(ctx, tx.NewScope)
	defer deliver()
	now := time.Now()
	if err := s.cacheRepoFor(ctx).MarkPathDeleted(projectId, relativePath, now); err != nil {
		return err
	}
	rootPath, rootErr := s.projectRootPath(ctx, projectId)
	if rootErr == nil {
		if _, statErr := os.Stat(rootPath); statErr != nil && os.IsNotExist(statErr) {
			return s.markProjectDeletedInTx(ctx, projectId, now)
		}
	}
	return nil
}

func (s *ProjectBrowserCacheService) recordDeletedFileLogs(ctx context.Context, project models.WorkSpace, paths []string) error {
	if s.projectSyncLog == nil {
		return nil
	}
	projectId := derefUint64(project.Id)
	if projectId == 0 {
		return nil
	}
	normalizedPaths := uniqueNormalizedRelativePaths(paths)
	if len(normalizedPaths) == 0 {
		return nil
	}
	latestIndex, err := s.projectSyncLog.ListLatestByTargetProjectAndPaths(ctx, projectId, normalizedPaths)
	if err != nil {
		return err
	}
	localAddress := ""
	if s.configManager != nil {
		localAddress = ResolveWorkstationAddress(s.configManager.GetConfig().NodeConfig.WorkstationAddress, "")
	}
	localNodeName := s.nodeClient.LocalNodeInfo().Name
	for _, path := range normalizedPaths {
		latest, found := latestIndex[path]
		if _, err := s.projectSyncLog.ensureLocalFileLogWithLatest(ctx, project, path, FileSide{Exists: false, Deleted: true}, latest, found, localNodeName, localAddress); err != nil {
			return err
		}
	}
	return nil
}

func samePath(left, right string) bool {
	caseSensitive := filesystemCaseSensitive(left)
	if _, err := os.Stat(left); err != nil {
		caseSensitive = filesystemCaseSensitive(right)
	}
	return pathCompareKeyWithSensitivity(filepath.Clean(left), caseSensitive) == pathCompareKeyWithSensitivity(filepath.Clean(right), caseSensitive)
}

func isProjectRootRelativePath(relativePath string) bool {
	return strings.TrimSpace(relativePath) == ""
}

func collectAffectedFilePathsFromDisk(absPath, relativePath string, info os.FileInfo) ([]string, error) {
	if info == nil {
		return nil, fmt.Errorf("目标路径信息不能为空")
	}
	if !info.IsDir() {
		return []string{relativePath}, nil
	}
	children, err := os.ReadDir(absPath)
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0)
	for _, child := range children {
		childAbsPath := filepath.Join(absPath, child.Name())
		childRelativePath := child.Name()
		if relativePath != "" {
			childRelativePath = relativePath + "/" + child.Name()
		}
		childInfo, statErr := child.Info()
		if statErr != nil {
			return nil, statErr
		}
		childPaths, childErr := collectAffectedFilePathsFromDisk(childAbsPath, childRelativePath, childInfo)
		if childErr != nil {
			return nil, childErr
		}
		paths = append(paths, childPaths...)
	}
	return paths, nil
}

func (s *ProjectBrowserCacheService) buildManifestFromCache(ctx context.Context, project models.WorkSpace, basePath string, depth int) (ManifestResult, error) {
	if depth <= 0 {
		depth = 3
	}
	basePath = normalizeRelativePath(basePath)
	nodes, err := s.cacheRepoFor(ctx).ListSubTree(derefUint64(project.Id), basePath, depth)
	if err != nil {
		return ManifestResult{}, err
	}
	entries := make([]ManifestEntry, 0, len(nodes))
	rootHash := project.Hash
	baseDeleted := false
	for _, node := range nodes {
		entries = append(entries, s.toManifestEntry(node))
		if node.RelativePath == basePath && basePath != "" {
			rootHash = node.Hash
			baseDeleted = node.DiskDeleted
		}
	}
	message := "目录缓存来自数据库，文件内容预览仍实时读取磁盘。"
	if project.DiskDeleted {
		message = "项目目录不存在，当前展示的是数据库中的缓存记录。"
	} else if baseDeleted {
		message = "当前目录已在磁盘上删除，展示的是数据库中的缓存记录。"
	}
	return ManifestResult{
		Project: NodeProject{
			Id:   derefUint64(project.Id),
			Name: project.Name,
			Path: project.Path,
		},
		BasePath:          basePath,
		Depth:             depth,
		RootHash:          rootHash,
		PathCaseSensitive: filesystemCaseSensitive(project.Path),
		Entries:           entries,
		ProjectDeleted:    project.DiskDeleted,
		CacheInitialized:  project.CacheInitialized,
		FromCache:         true,
		Message:           message,
	}, nil
}

func (s *ProjectBrowserCacheService) scanProjectNodes(rootPath string, scannedAt time.Time) ([]models.ProjectNodeCache, string, error) {
	ignoreMatcher := newProjectIgnoreMatcher(rootPath)
	children, rootHash, err := s.scanDirectory(rootPath, "", ignoreMatcher, scannedAt)
	if err != nil {
		return nil, "", err
	}
	return children, rootHash, nil
}

func (s *ProjectBrowserCacheService) scanDirectory(absPath, relativePath string, ignoreMatcher *projectIgnoreMatcher, scannedAt time.Time) ([]models.ProjectNodeCache, string, error) {
	children, err := os.ReadDir(absPath)
	if err != nil {
		return nil, "", err
	}
	sort.Slice(children, func(i, j int) bool { return children[i].Name() < children[j].Name() })
	entries := make([]models.ProjectNodeCache, 0)
	immediateHashes := make([]string, 0, len(children))
	for _, child := range children {
		childRelative := child.Name()
		if relativePath != "" {
			childRelative = relativePath + "/" + child.Name()
		}
		childPath := filepath.Join(absPath, child.Name())
		info, statErr := os.Stat(childPath)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				continue
			}
			return nil, "", statErr
		}
		if ignoreMatcher != nil && ignoreMatcher.ShouldIgnore(childRelative, info.IsDir()) {
			continue
		}
		if info.IsDir() {
			childEntries, childHash, childErr := s.scanDirectory(childPath, childRelative, ignoreMatcher, scannedAt)
			if childErr != nil {
				return nil, "", childErr
			}
			directoryNode := models.ProjectNodeCache{
				BaseModel:    &pkgModels.BaseModel{},
				RelativePath: childRelative,
				ParentPath:   relativePath,
				ProjectId:    0,
				Name:         child.Name(),
				EntryType:    "directory",
				Hash:         childHash,
				Size:         0,
				EntryModTime: info.ModTime().Unix(),
				Depth:        nodePathDepth(childRelative),
				HasChildren:  len(childEntries) > 0,
				DiskDeleted:  false,
				LastScanAt:   &scannedAt,
			}
			immediateHashes = append(immediateHashes, directoryNode.EntryType+":"+directoryNode.Name+":"+directoryNode.Hash)
			entries = append(entries, directoryNode)
			entries = append(entries, childEntries...)
			continue
		}
		content, readErr := os.ReadFile(childPath)
		if readErr != nil {
			return nil, "", readErr
		}
		fileNode := models.ProjectNodeCache{
			BaseModel:    &pkgModels.BaseModel{},
			RelativePath: childRelative,
			ParentPath:   relativePath,
			ProjectId:    0,
			Name:         child.Name(),
			EntryType:    "file",
			Hash:         hashBytes(content),
			Size:         info.Size(),
			EntryModTime: info.ModTime().Unix(),
			Depth:        nodePathDepth(childRelative),
			HasChildren:  false,
			DiskDeleted:  false,
			LastScanAt:   &scannedAt,
		}
		immediateHashes = append(immediateHashes, fileNode.EntryType+":"+fileNode.Name+":"+fileNode.Hash)
		entries = append(entries, fileNode)
	}
	for index := range entries {
		entries[index].ProjectId = 0
	}
	return entries, hashBytes([]byte(strings.Join(immediateHashes, "\n"))), nil
}

func (s *ProjectBrowserCacheService) cacheRepoFor(ctx context.Context) *internalRepo.ProjectNodeCacheRepository {
	if scope, ok := tx.FromCtx(ctx); ok {
		return s.cacheRepo.WithScope(scope)
	}
	return s.cacheRepo
}

func (s *ProjectBrowserCacheService) resolveProjectRelativePath(rootPath, input string, requireExists bool) (string, string, error) {
	relativePath := normalizeRelativePath(input)
	absPath := rootPath
	if relativePath != "" {
		absPath = filepath.Join(rootPath, filepath.FromSlash(relativePath))
	}
	absPath = filepath.Clean(absPath)
	if !isSameOrWithinPath(rootPath, absPath) {
		return "", "", fmt.Errorf("路径超出项目根目录范围")
	}
	if requireExists {
		if _, err := os.Stat(absPath); err != nil {
			return "", "", fmt.Errorf("目标路径不存在")
		}
	}
	return absPath, relativePath, nil
}

func (s *ProjectBrowserCacheService) projectRootPath(ctx context.Context, projectId uint64) (string, error) {
	project, err := s.workSpaceService.GetProject(ctx, projectId)
	if err != nil {
		return "", err
	}
	return filepath.Clean(project.Path), nil
}

func (s *ProjectBrowserCacheService) toManifestEntry(node models.ProjectNodeCache) ManifestEntry {
	return ManifestEntry{
		RelativePath: node.RelativePath,
		Name:         node.Name,
		EntryType:    node.EntryType,
		Size:         node.Size,
		ModifyTime:   node.EntryModTime,
		Hash:         node.Hash,
		HasChildren:  node.HasChildren,
		Deleted:      node.DiskDeleted,
	}
}

func (s *ProjectBrowserCacheService) ensureFileLog(ctx context.Context, project models.WorkSpace, relativePath string, current FileSide) error {
	if s.projectSyncLog == nil {
		return nil
	}
	localAddress := ""
	if s.configManager != nil {
		localAddress = ResolveWorkstationAddress(s.configManager.GetConfig().NodeConfig.WorkstationAddress, "")
	}
	_, err := s.projectSyncLog.EnsureLocalFileLog(ctx, project, relativePath, current, s.nodeClient.LocalNodeInfo().Name, localAddress)
	return err
}

func (s *ProjectBrowserCacheService) reconcileProjectLogs(ctx context.Context, project models.WorkSpace) error {
	if s.projectSyncLog == nil {
		return nil
	}
	projectId := derefUint64(project.Id)
	if projectId == 0 {
		return nil
	}
	nodes, err := s.cacheRepoFor(ctx).ListProjectFiles(projectId)
	if err != nil {
		return err
	}
	latestIndex, err := s.projectSyncLog.ListLatestByTargetProject(ctx, projectId)
	if err != nil {
		return err
	}
	rootPath := filepath.Clean(project.Path)
	localAddress := ""
	if s.configManager != nil {
		localAddress = ResolveWorkstationAddress(s.configManager.GetConfig().NodeConfig.WorkstationAddress, "")
	}
	localNodeName := s.nodeClient.LocalNodeInfo().Name
	for _, node := range nodes {
		current := FileSide{Exists: false, Deleted: true}
		if !node.DiskDeleted {
			absPath, _, pathErr := s.resolveProjectRelativePath(rootPath, node.RelativePath, false)
			if pathErr != nil {
				projectBrowserLog.Warn("刷新后解析文件路径失败: ", pathErr)
				continue
			}
			content, readErr := os.ReadFile(absPath)
			if readErr != nil {
				if !os.IsNotExist(readErr) {
					projectBrowserLog.Warn("刷新后读取文件内容失败: ", readErr)
					continue
				}
			} else {
				current = FileSide{
					Exists:        true,
					Hash:          hashBytes(content),
					Text:          isTextContent(content),
					ContentBase64: base64Encode(content),
					Size:          int64(len(content)),
				}
				if current.Text {
					current.Content = string(content)
				}
			}
		}
		latest, found := latestIndex[normalizeRelativePath(node.RelativePath)]
		if _, verifyErr := s.projectSyncLog.ensureLocalFileLogWithLatest(ctx, project, node.RelativePath, current, latest, found, localNodeName, localAddress); verifyErr != nil {
			projectBrowserLog.Warn("刷新后补充修改日志失败: ", verifyErr)
		}
	}
	return nil
}

func (s *ProjectBrowserCacheService) getProjectLock(projectId uint64) *sync.Mutex {
	lock, _ := s.projectLocks.LoadOrStore(projectId, &sync.Mutex{})
	return lock.(*sync.Mutex)
}

func base64Encode(content []byte) string {
	return base64.StdEncoding.EncodeToString(content)
}

func nodePathDepth(relativePath string) int {
	trimmed := strings.Trim(relativePath, "/")
	if trimmed == "" {
		return 0
	}
	return len(strings.Split(trimmed, "/"))
}

func uniqueNormalizedRelativePaths(paths []string) []string {
	if len(paths) == 0 {
		return nil
	}
	result := make([]string, 0, len(paths))
	seen := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		normalized := normalizeRelativePath(path)
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	return result
}
