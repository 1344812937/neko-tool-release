package service_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"neko-tool/internal/models"
	service "neko-tool/internal/service"
)

func TestBuildSyncItemStatus(t *testing.T) {
	tests := []struct {
		name       string
		sourceFile service.FileSide
		targetFile service.FileSide
		wantStatus string
		wantOK     bool
	}{
		{
			name:       "源存在目标不存在时生成左侧新增状态",
			sourceFile: service.FileSide{Exists: true, Hash: "source-hash"},
			targetFile: service.FileSide{Exists: false},
			wantStatus: "left_only",
			wantOK:     true,
		},
		{
			name:       "源不存在目标存在时生成右侧新增状态",
			sourceFile: service.FileSide{Exists: false},
			targetFile: service.FileSide{Exists: true, Hash: "target-hash"},
			wantStatus: "right_only",
			wantOK:     true,
		},
		{
			name:       "两侧内容一致时不生成同步项",
			sourceFile: service.FileSide{Exists: true, Hash: "same-hash"},
			targetFile: service.FileSide{Exists: true, Hash: "same-hash"},
			wantStatus: "",
			wantOK:     false,
		},
		{
			name:       "仅换行风格不同的文本文件视为一致",
			sourceFile: service.FileSide{Exists: true, Hash: "raw-crlf", Text: true, NormalizedHash: "normalized-hash"},
			targetFile: service.FileSide{Exists: true, Hash: "raw-lf", Text: true, NormalizedHash: "normalized-hash"},
			wantStatus: "",
			wantOK:     false,
		},
		{
			name:       "两侧内容不同生成文件变更状态",
			sourceFile: service.FileSide{Exists: true, Hash: "left-hash"},
			targetFile: service.FileSide{Exists: true, Hash: "right-hash"},
			wantStatus: "file_changed",
			wantOK:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStatus, gotOK := service.BuildSyncItemStatusForTest(tt.sourceFile, tt.targetFile)
			if gotStatus != tt.wantStatus {
				t.Fatalf("status = %q, want %q", gotStatus, tt.wantStatus)
			}
			if gotOK != tt.wantOK {
				t.Fatalf("ok = %v, want %v", gotOK, tt.wantOK)
			}
		})
	}
}

func TestCompareManifestMatchesCaseInsensitivePaths(t *testing.T) {
	left := service.ManifestResult{
		PathCaseSensitive: false,
		Entries:           []service.ManifestEntry{{RelativePath: "api/src/dto/file.txt", Name: "file.txt", EntryType: "file", Hash: "same", Size: 1}},
	}
	right := service.ManifestResult{
		PathCaseSensitive: false,
		Entries:           []service.ManifestEntry{{RelativePath: "api/src/DTO/file.txt", Name: "file.txt", EntryType: "file", Hash: "same", Size: 1}},
	}

	result := service.CompareManifestForTest(left, right)
	if result.Summary.SameCount != 1 {
		t.Fatalf("sameCount = %d, want 1", result.Summary.SameCount)
	}
	if len(result.Items) != 0 {
		t.Fatalf("items = %#v, want no diff items", result.Items)
	}
	if result.Summary.Total != 1 {
		t.Fatalf("total = %d, want 1", result.Summary.Total)
	}
}

func TestCompareManifestKeepsAmbiguousCaseSensitiveEntries(t *testing.T) {
	left := service.ManifestResult{
		PathCaseSensitive: true,
		Entries: []service.ManifestEntry{
			{RelativePath: "api/src/dto/file.txt", Name: "file.txt", EntryType: "file", Hash: "left-a", Size: 1},
			{RelativePath: "api/src/DTO/file.txt", Name: "file.txt", EntryType: "file", Hash: "left-b", Size: 1},
		},
	}
	right := service.ManifestResult{
		PathCaseSensitive: false,
		Entries:           []service.ManifestEntry{{RelativePath: "api/src/dTo/file.txt", Name: "file.txt", EntryType: "file", Hash: "right", Size: 1}},
	}

	result := service.CompareManifestForTest(left, right)
	if result.Summary.LeftOnly != 2 {
		t.Fatalf("leftOnly = %d, want 2", result.Summary.LeftOnly)
	}
	if result.Summary.RightOnly != 1 {
		t.Fatalf("rightOnly = %d, want 1", result.Summary.RightOnly)
	}
	if len(result.Items) != 3 {
		t.Fatalf("len(items) = %d, want 3", len(result.Items))
	}
}

func TestCompareManifestIgnoresLineEndingOnlyDifference(t *testing.T) {
	left := service.ManifestResult{
		PathCaseSensitive: true,
		Entries: []service.ManifestEntry{{
			RelativePath:   "api/src/demo.txt",
			Name:           "demo.txt",
			EntryType:      "file",
			Hash:           "raw-crlf",
			Text:           true,
			NormalizedHash: "normalized-hash",
			Size:           12,
		}},
	}
	right := service.ManifestResult{
		PathCaseSensitive: true,
		Entries: []service.ManifestEntry{{
			RelativePath:   "api/src/demo.txt",
			Name:           "demo.txt",
			EntryType:      "file",
			Hash:           "raw-lf",
			Text:           true,
			NormalizedHash: "normalized-hash",
			Size:           11,
		}},
	}

	result := service.CompareManifestForTest(left, right)
	if result.Summary.SameCount != 1 {
		t.Fatalf("sameCount = %d, want 1", result.Summary.SameCount)
	}
	if len(result.Items) != 0 {
		t.Fatalf("items = %#v, want no diff items", result.Items)
	}
	if result.Summary.Total != 1 {
		t.Fatalf("total = %d, want 1", result.Summary.Total)
	}
}

func TestBuildFileSideTreatsNonUTF8TextAsText(t *testing.T) {
	left := service.BuildFileSideFromContentForTest([]byte{0xC4, 0xE3, 0xBA, 0xC3, '\r', '\n'})
	right := service.BuildFileSideFromContentForTest([]byte{0xC4, 0xE3, 0xBA, 0xC3, '\n'})

	if !left.Text || !right.Text {
		t.Fatalf("非 UTF-8 文本内容应被识别为文本: left=%v right=%v", left.Text, right.Text)
	}
	status, ok := service.BuildSyncItemStatusForTest(left, right)
	if ok {
		t.Fatalf("status = %q, want no diff item", status)
	}
}

func TestBuildFileSideKeepsBinaryContentAsBinary(t *testing.T) {
	file := service.BuildFileSideFromContentForTest([]byte{0x00, 0x01, 0x02, 0x03})
	if file.Text {
		t.Fatal("包含 NUL 的内容应继续识别为二进制")
	}
	if file.NormalizedHash != "" {
		t.Fatalf("normalizedHash = %q, want empty", file.NormalizedHash)
	}
}

func TestBuildProjectSyncLogSnapshotsStoresReversePatchForTextChange(t *testing.T) {
	before := service.BuildFileSideFromContentForTest([]byte("line-1\nline-2\n"))
	after := service.BuildFileSideFromContentForTest([]byte("line-1\nline-3\n"))

	beforeSnapshot, afterSnapshot, diffAlgorithm, err := service.BuildProjectSyncLogSnapshotsForTest(before, after)
	if err != nil {
		t.Fatalf("BuildProjectSyncLogSnapshotsForTest 返回错误: %v", err)
	}
	if beforeSnapshot.StorageKind != "compressed_reverse_patch" {
		t.Fatalf("before storage = %q, want compressed_reverse_patch", beforeSnapshot.StorageKind)
	}
	if afterSnapshot.StorageKind != "compressed_full_text" {
		t.Fatalf("after storage = %q, want compressed_full_text", afterSnapshot.StorageKind)
	}
	if diffAlgorithm != "diff_match_patch" {
		t.Fatalf("diffAlgorithm = %q, want diff_match_patch", diffAlgorithm)
	}
	detail, err := service.BuildProjectSyncLogDetailForTest(models.ProjectSyncLog{
		BeforeExists:      true,
		BeforeHash:        before.Hash,
		BeforeEncoding:    beforeSnapshot.Encoding,
		BeforeStorageKind: beforeSnapshot.StorageKind,
		BeforeContentSize: beforeSnapshot.ContentSize,
		BeforeContent:     beforeSnapshot.Content,
		AfterHash:         after.Hash,
		AfterEncoding:     afterSnapshot.Encoding,
		AfterStorageKind:  afterSnapshot.StorageKind,
		AfterContentSize:  afterSnapshot.ContentSize,
		AfterContent:      afterSnapshot.Content,
		DiffAlgorithm:     diffAlgorithm,
	})
	if err != nil {
		t.Fatalf("BuildProjectSyncLogDetailForTest 返回错误: %v", err)
	}
	if detail.BeforeContent != before.Content {
		t.Fatalf("before content = %q, want %q", detail.BeforeContent, before.Content)
	}
	if detail.AfterContent != after.Content {
		t.Fatalf("after content = %q, want %q", detail.AfterContent, after.Content)
	}
}

func TestBuildProjectSyncLogSnapshotsCompressesCreateTextContent(t *testing.T) {
	after := service.BuildFileSideFromContentForTest([]byte("hello\nworld\n"))

	_, afterSnapshot, _, err := service.BuildProjectSyncLogSnapshotsForTest(service.FileSide{}, after)
	if err != nil {
		t.Fatalf("BuildProjectSyncLogSnapshotsForTest 返回错误: %v", err)
	}
	if afterSnapshot.StorageKind != "compressed_full_text" {
		t.Fatalf("after storage = %q, want compressed_full_text", afterSnapshot.StorageKind)
	}
	detail, err := service.BuildProjectSyncLogDetailForTest(models.ProjectSyncLog{
		AfterHash:        after.Hash,
		AfterEncoding:    afterSnapshot.Encoding,
		AfterStorageKind: afterSnapshot.StorageKind,
		AfterContentSize: afterSnapshot.ContentSize,
		AfterContent:     afterSnapshot.Content,
	})
	if err != nil {
		t.Fatalf("BuildProjectSyncLogDetailForTest 返回错误: %v", err)
	}
	if detail.AfterContent != after.Content {
		t.Fatalf("after content = %q, want %q", detail.AfterContent, after.Content)
	}
}

func TestBuildProjectSyncLogSnapshotsDropsLargeTextContent(t *testing.T) {
	largeContent := strings.Repeat("a", 15*1024*1024+1)
	after := service.BuildFileSideFromContentForTest([]byte(largeContent))

	_, afterSnapshot, diffAlgorithm, err := service.BuildProjectSyncLogSnapshotsForTest(service.FileSide{}, after)
	if err != nil {
		t.Fatalf("BuildProjectSyncLogSnapshotsForTest 返回错误: %v", err)
	}
	if afterSnapshot.StorageKind != "hash_only" {
		t.Fatalf("after storage = %q, want hash_only", afterSnapshot.StorageKind)
	}
	if afterSnapshot.OmittedReason != "size_limit" {
		t.Fatalf("after omitted reason = %q, want size_limit", afterSnapshot.OmittedReason)
	}
	if afterSnapshot.Content != "" {
		t.Fatalf("after content = %q, want empty", afterSnapshot.Content)
	}
	if diffAlgorithm != "" {
		t.Fatalf("diffAlgorithm = %q, want empty", diffAlgorithm)
	}
}

func TestBuildProjectSyncLogSnapshotsDropsBinaryContent(t *testing.T) {
	before := service.BuildFileSideFromContentForTest([]byte{0x00, 0x01, 0x02})
	after := service.BuildFileSideFromContentForTest([]byte{0x00, 0x01, 0x03})

	beforeSnapshot, afterSnapshot, _, err := service.BuildProjectSyncLogSnapshotsForTest(before, after)
	if err != nil {
		t.Fatalf("BuildProjectSyncLogSnapshotsForTest 返回错误: %v", err)
	}
	if beforeSnapshot.StorageKind != "hash_only" || afterSnapshot.StorageKind != "hash_only" {
		t.Fatalf("binary snapshots = %#v %#v, want hash_only", beforeSnapshot, afterSnapshot)
	}
	if beforeSnapshot.OmittedReason != "binary" || afterSnapshot.OmittedReason != "binary" {
		t.Fatalf("binary omitted reasons = %q %q, want binary", beforeSnapshot.OmittedReason, afterSnapshot.OmittedReason)
	}
}

func TestBuildProjectSyncLogDetailSupportsLegacyRows(t *testing.T) {
	detail, err := service.BuildProjectSyncLogDetailForTest(models.ProjectSyncLog{
		BeforeExists:   true,
		BeforeEncoding: "text",
		BeforeContent:  "legacy-before",
		AfterEncoding:  "text",
		AfterContent:   "legacy-after",
	})
	if err != nil {
		t.Fatalf("BuildProjectSyncLogDetailForTest 返回错误: %v", err)
	}
	if detail.BeforeStorageKind != "legacy_full" {
		t.Fatalf("before storage kind = %q, want legacy_full", detail.BeforeStorageKind)
	}
	if detail.AfterStorageKind != "legacy_full" {
		t.Fatalf("after storage kind = %q, want legacy_full", detail.AfterStorageKind)
	}
	if detail.BeforeContent != "legacy-before" || detail.AfterContent != "legacy-after" {
		t.Fatalf("legacy content mismatch: before=%q after=%q", detail.BeforeContent, detail.AfterContent)
	}
}

func TestShouldKeepLatestProjectSyncLogHandlesZeroValueProject(t *testing.T) {
	keep, keepID := service.ShouldKeepLatestProjectSyncLogForTest(models.WorkSpace{}, models.ProjectSyncLog{})
	if keep {
		t.Fatal("keep = true, want false")
	}
	if keepID != 0 {
		t.Fatalf("keepID = %d, want 0", keepID)
	}
}

func TestResolveExistingRelativePathCaseInsensitive(t *testing.T) {
	rootDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(rootDir, "api", "src", "dto"), 0755); err != nil {
		t.Fatalf("创建测试目录失败: %v", err)
	}
	filePath := filepath.Join(rootDir, "api", "src", "dto", "file.txt")
	if err := os.WriteFile(filePath, []byte("demo"), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	resolvedPath, exists, exactMatch, err := service.ResolveExistingRelativePathCaseInsensitiveForTest(rootDir, "api/src/DTO/file.txt")
	if err != nil {
		t.Fatalf("ResolveExistingRelativePathCaseInsensitiveForTest 返回错误: %v", err)
	}
	if !exists {
		t.Fatal("exists = false, want true")
	}
	if exactMatch {
		t.Fatal("exactMatch = true, want false")
	}
	if resolvedPath != "api/src/dto/file.txt" {
		t.Fatalf("resolvedPath = %q, want %q", resolvedPath, "api/src/dto/file.txt")
	}
}
