package service_test

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	service "neko-tool/internal/service"
)

func TestIsProjectRootRelativePath(t *testing.T) {
	if !service.IsProjectRootRelativePathForTest("") {
		t.Fatal("空路径应视为项目根目录")
	}
	if service.IsProjectRootRelativePathForTest("nested/file.txt") {
		t.Fatal("子路径不应视为项目根目录")
	}
}

func TestCollectAffectedFilePathsFromDisk(t *testing.T) {
	rootDir := t.TempDir()
	nestedDir := filepath.Join(rootDir, "nested")
	deeperDir := filepath.Join(nestedDir, "deeper")
	if err := os.MkdirAll(deeperDir, 0755); err != nil {
		t.Fatalf("创建测试目录失败: %v", err)
	}
	files := []string{
		filepath.Join(rootDir, "root.txt"),
		filepath.Join(nestedDir, "child.txt"),
		filepath.Join(deeperDir, "deep.txt"),
	}
	for _, file := range files {
		if err := os.WriteFile(file, []byte(file), 0644); err != nil {
			t.Fatalf("创建测试文件失败: %v", err)
		}
	}

	info, err := os.Stat(rootDir)
	if err != nil {
		t.Fatalf("读取目录信息失败: %v", err)
	}
	got, err := service.CollectAffectedFilePathsFromDiskForTest(rootDir, "fixtures", info)
	if err != nil {
		t.Fatalf("CollectAffectedFilePathsFromDiskForTest 返回错误: %v", err)
	}
	sort.Strings(got)
	want := []string{
		"fixtures/nested/child.txt",
		"fixtures/nested/deeper/deep.txt",
		"fixtures/root.txt",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("paths = %#v, want %#v", got, want)
	}

	fileInfo, err := os.Stat(files[0])
	if err != nil {
		t.Fatalf("读取文件信息失败: %v", err)
	}
	gotFile, err := service.CollectAffectedFilePathsFromDiskForTest(files[0], "fixtures/root.txt", fileInfo)
	if err != nil {
		t.Fatalf("CollectAffectedFilePathsFromDiskForTest 文件场景返回错误: %v", err)
	}
	if !reflect.DeepEqual(gotFile, []string{"fixtures/root.txt"}) {
		t.Fatalf("file paths = %#v, want %#v", gotFile, []string{"fixtures/root.txt"})
	}
}
