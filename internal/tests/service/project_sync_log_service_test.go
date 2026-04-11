package service_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"neko-tool/internal/models"
	internalRepo "neko-tool/internal/repository"
	service "neko-tool/internal/service"
	"neko-tool/pkg/core/tx"
	pkgModels "neko-tool/pkg/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestListLatestByTargetProjectSelectsNewestLogPerPath(t *testing.T) {
	ctx := context.Background()
	svc, _ := newTestProjectSyncLogService(t)
	projectId := uint64(1001)
	otherProjectId := uint64(1002)
	baseTime := time.Date(2026, 4, 9, 10, 0, 0, 0, time.UTC)

	seedProjectSyncLog(t, svc, models.ProjectSyncLog{
		BaseModel:       newTestBaseModel(101, baseTime.Add(1*time.Minute)),
		TargetProjectId: projectId,
		RelativePath:    "dir/a.txt",
		AfterHash:       "a-old",
		OperatedAt:      timePtr(baseTime.Add(1 * time.Minute)),
	})
	seedProjectSyncLog(t, svc, models.ProjectSyncLog{
		BaseModel:       newTestBaseModel(102, baseTime.Add(2*time.Minute)),
		TargetProjectId: projectId,
		RelativePath:    "dir/a.txt",
		AfterHash:       "a-new",
		OperatedAt:      timePtr(baseTime.Add(2 * time.Minute)),
	})
	seedProjectSyncLog(t, svc, models.ProjectSyncLog{
		BaseModel:       newTestBaseModel(103, baseTime.Add(3*time.Minute)),
		TargetProjectId: projectId,
		RelativePath:    "dir/b.txt",
		AfterHash:       "b-old",
		OperatedAt:      timePtr(baseTime.Add(3 * time.Minute)),
	})
	seedProjectSyncLog(t, svc, models.ProjectSyncLog{
		BaseModel:       newTestBaseModel(104, baseTime.Add(4*time.Minute)),
		TargetProjectId: projectId,
		RelativePath:    "dir/b.txt",
		AfterHash:       "b-new",
		OperatedAt:      timePtr(baseTime.Add(3 * time.Minute)),
	})
	seedProjectSyncLog(t, svc, models.ProjectSyncLog{
		BaseModel:       newTestBaseModel(105, baseTime.Add(5*time.Minute)),
		TargetProjectId: projectId,
		RelativePath:    "dir/c.txt",
		AfterHash:       "c-old",
		OperatedAt:      timePtr(baseTime.Add(5 * time.Minute)),
	})
	seedProjectSyncLog(t, svc, models.ProjectSyncLog{
		BaseModel:       newTestBaseModel(106, baseTime.Add(5*time.Minute)),
		TargetProjectId: projectId,
		RelativePath:    "dir/c.txt",
		AfterHash:       "c-new",
		OperatedAt:      timePtr(baseTime.Add(5 * time.Minute)),
	})
	seedProjectSyncLog(t, svc, models.ProjectSyncLog{
		BaseModel:       newTestBaseModel(107, baseTime.Add(10*time.Minute)),
		TargetProjectId: otherProjectId,
		RelativePath:    "dir/a.txt",
		AfterHash:       "other-project",
		OperatedAt:      timePtr(baseTime.Add(10 * time.Minute)),
	})

	latest, err := svc.ListLatestByTargetProject(ctx, projectId)
	if err != nil {
		t.Fatalf("ListLatestByTargetProject 返回错误: %v", err)
	}
	if len(latest) != 3 {
		t.Fatalf("len(latest) = %d, want 3", len(latest))
	}
	assertLatestHash(t, latest, "dir/a.txt", "a-new")
	assertLatestHash(t, latest, "dir/b.txt", "b-new")
	assertLatestHash(t, latest, "dir/c.txt", "c-new")
}

func TestListLatestByTargetProjectAndPathsBatchesAndFiltersPaths(t *testing.T) {
	ctx := context.Background()
	svc, _ := newTestProjectSyncLogService(t)
	projectId := uint64(2001)
	baseTime := time.Date(2026, 4, 9, 12, 0, 0, 0, time.UTC)
	const totalPaths = 240

	queryPaths := make([]string, 0, totalPaths+2)
	for index := 0; index < totalPaths; index++ {
		path := fmt.Sprintf("dir/file-%03d.txt", index)
		seedProjectSyncLog(t, svc, models.ProjectSyncLog{
			BaseModel:       newTestBaseModel(uint64(3000+index), baseTime.Add(time.Duration(index)*time.Second)),
			TargetProjectId: projectId,
			RelativePath:    path,
			AfterHash:       fmt.Sprintf("hash-%03d", index),
			OperatedAt:      timePtr(baseTime.Add(time.Duration(index) * time.Second)),
		})
		queryPaths = append(queryPaths, path)
	}
	queryPaths = append(queryPaths, " dir/file-001.txt ", "missing.txt")

	latest, err := svc.ListLatestByTargetProjectAndPaths(ctx, projectId, queryPaths)
	if err != nil {
		t.Fatalf("ListLatestByTargetProjectAndPaths 返回错误: %v", err)
	}
	if len(latest) != totalPaths {
		t.Fatalf("len(latest) = %d, want %d", len(latest), totalPaths)
	}
	assertLatestHash(t, latest, "dir/file-000.txt", "hash-000")
	assertLatestHash(t, latest, "dir/file-239.txt", "hash-239")
	if _, exists := latest["missing.txt"]; exists {
		t.Fatal("missing path should not exist in latest map")
	}
}

func TestEnsureLocalFileLogKeepsExistingBehavior(t *testing.T) {
	ctx := context.Background()
	svc, _ := newTestProjectSyncLogService(t)
	projectId := uint64(3001)
	project := models.WorkSpace{
		BaseModel: &pkgModels.BaseModel{Id: uint64Ptr(projectId)},
		Name:      "demo-project",
	}
	current := service.BuildFileSideFromContentForTest([]byte("hello\nworld\n"))

	created, err := svc.EnsureLocalFileLog(ctx, project, "demo.txt", current, "本地节点", "")
	if err != nil {
		t.Fatalf("首次 EnsureLocalFileLog 返回错误: %v", err)
	}
	if !created {
		t.Fatal("首次调用应创建日志")
	}

	created, err = svc.EnsureLocalFileLog(ctx, project, "demo.txt", current, "本地节点", "")
	if err != nil {
		t.Fatalf("第二次 EnsureLocalFileLog 返回错误: %v", err)
	}
	if created {
		t.Fatal("内容未变化时不应重复创建日志")
	}

	rows, err := svc.ListByTargetProjectAndPath(ctx, projectId, "demo.txt")
	if err != nil {
		t.Fatalf("ListByTargetProjectAndPath 返回错误: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(rows))
	}
	if rows[0].ChangeType != "local_snapshot" {
		t.Fatalf("changeType = %q, want local_snapshot", rows[0].ChangeType)
	}
}

func newTestProjectSyncLogService(t *testing.T) (*service.ProjectSyncLogService, *internalRepo.ProjectSyncLogRepository) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "project-sync-log.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	ds := tx.CreateDataSource("test", db)
	repo := internalRepo.NewProjectSyncLogRepository(ds)
	return service.NewProjectSyncLogService(repo), repo
}

func seedProjectSyncLog(t *testing.T, svc *service.ProjectSyncLogService, entry models.ProjectSyncLog) {
	t.Helper()
	if entry.BaseModel == nil {
		entry.BaseModel = &pkgModels.BaseModel{}
	}
	if entry.OperatedAt == nil {
		now := time.Now()
		entry.OperatedAt = &now
	}
	if err := svc.RecordLog(context.Background(), entry); err != nil {
		t.Fatalf("写入测试日志失败: %v", err)
	}
}

func newTestBaseModel(id uint64, modifyTime time.Time) *pkgModels.BaseModel {
	createTime := modifyTime.Add(-1 * time.Minute)
	return &pkgModels.BaseModel{
		Id:         uint64Ptr(id),
		Valid:      1,
		CreateTime: &createTime,
		ModifyTime: &modifyTime,
	}
}

func assertLatestHash(t *testing.T, latest map[string]models.ProjectSyncLog, relativePath, wantHash string) {
	t.Helper()
	row, exists := latest[relativePath]
	if !exists {
		t.Fatalf("latest map missing path %q", relativePath)
	}
	if row.AfterHash != wantHash {
		t.Fatalf("path %q afterHash = %q, want %q", relativePath, row.AfterHash, wantHash)
	}
}

func timePtr(value time.Time) *time.Time {
	return &value
}

func uint64Ptr(value uint64) *uint64 {
	return &value
}
