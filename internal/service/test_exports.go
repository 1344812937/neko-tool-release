package service

import (
	"os"

	"neko-tool/internal/models"
)

// BuildSyncItemStatusForTest 暴露同步项状态判定，供统一测试目录中的单元测试复用。
func BuildSyncItemStatusForTest(sourceFile, targetFile FileSide) (string, bool) {
	return buildSyncItemStatus(sourceFile, targetFile)
}

// IsProjectRootRelativePathForTest 暴露项目根目录判定，供统一测试目录中的单元测试复用。
func IsProjectRootRelativePathForTest(relativePath string) bool {
	return isProjectRootRelativePath(relativePath)
}

// CollectAffectedFilePathsFromDiskForTest 暴露磁盘文件枚举逻辑，供统一测试目录中的单元测试复用。
func CollectAffectedFilePathsFromDiskForTest(absPath, relativePath string, info os.FileInfo) ([]string, error) {
	return collectAffectedFilePathsFromDisk(absPath, relativePath, info)
}

// CompareManifestForTest 暴露目录清单比较逻辑，供统一测试目录中的单元测试复用。
func CompareManifestForTest(left, right ManifestResult) CompareResult {
	return (&CompareService{}).compareManifest(left, right)
}

// ResolveExistingRelativePathCaseInsensitiveForTest 暴露大小写不敏感路径解析逻辑，供统一测试目录中的单元测试复用。
func ResolveExistingRelativePathCaseInsensitiveForTest(rootPath, relativePath string) (string, bool, bool, error) {
	return resolveExistingRelativePathCaseInsensitive(rootPath, relativePath)
}

// BuildFileSideFromContentForTest 暴露文件内容分析逻辑，供统一测试目录中的单元测试复用。
func BuildFileSideFromContentForTest(content []byte) FileSide {
	return buildFileSideFromContent(content)
}

type ProjectSyncLogSnapshotForTest struct {
	Encoding      string
	Content       string
	StorageKind   string
	ContentSize   int64
	OmittedReason string
}

// BuildProjectSyncLogSnapshotsForTest 暴露日志快照构建逻辑，供统一测试目录中的单元测试复用。
func BuildProjectSyncLogSnapshotsForTest(beforeFile, afterFile FileSide) (ProjectSyncLogSnapshotForTest, ProjectSyncLogSnapshotForTest, string, error) {
	beforeSnapshot, afterSnapshot, diffAlgorithm, err := buildProjectSyncLogSnapshots(beforeFile, afterFile)
	return ProjectSyncLogSnapshotForTest(beforeSnapshot), ProjectSyncLogSnapshotForTest(afterSnapshot), diffAlgorithm, err
}

// BuildProjectSyncLogDetailForTest 暴露日志详情重建逻辑，供统一测试目录中的单元测试复用。
func BuildProjectSyncLogDetailForTest(entry models.ProjectSyncLog) (ProjectSyncLogDetail, error) {
	return buildProjectSyncLogDetail(entry)
}

// ShouldKeepLatestProjectSyncLogForTest 暴露站点日志清理的保留判定逻辑，供统一测试目录中的单元测试复用。
func ShouldKeepLatestProjectSyncLogForTest(project models.WorkSpace, latest models.ProjectSyncLog) (bool, uint64) {
	return (&CompareService{}).shouldKeepLatestProjectSyncLog(project, latest)
}
