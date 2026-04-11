package service

import (
	"fmt"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"neko-tool/internal/config"

	"github.com/ncruces/zenity"
)

type RuntimeCapabilities struct {
	AccessMode                    string   `json:"accessMode"`
	NativeDirectoryPickerEnabled  bool     `json:"nativeDirectoryPickerEnabled"`
	BrowserDirectoryPickerEnabled bool     `json:"browserDirectoryPickerEnabled"`
	AllowedRoots                  []string `json:"allowedRoots"`
	Message                       string   `json:"message"`
}

type DirectoryEntry struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	HasChildren bool   `json:"hasChildren"`
}

type ProjectAccessService struct {
	configManager *config.ApplicationConfigManager
}

// NewProjectAccessService 构造项目目录访问能力服务。
func NewProjectAccessService(configManager *config.ApplicationConfigManager) *ProjectAccessService {
	return &ProjectAccessService{configManager: configManager}
}

func (s *ProjectAccessService) projectConfig() config.ProjectAccess {
	cfg := s.configManager.GetConfig()
	return cfg.ProjectAccess
}

func (s *ProjectAccessService) normalizedAllowedRoots() []string {
	cfg := s.projectConfig()
	roots := make([]string, 0, len(cfg.AllowedRoots))
	seen := map[string]bool{}
	for _, root := range cfg.AllowedRoots {
		if strings.TrimSpace(root) == "" {
			continue
		}
		absPath, err := filepath.Abs(root)
		if err != nil {
			continue
		}
		absPath = filepath.Clean(absPath)
		compareKey := pathCompareKey(absPath)
		if !seen[compareKey] {
			seen[compareKey] = true
			roots = append(roots, absPath)
		}
	}
	sort.Strings(roots)
	return roots
}

// GetAllowedRoots 返回配置中声明的允许访问根目录列表。
func (s *ProjectAccessService) GetAllowedRoots() []string {
	return s.normalizedAllowedRoots()
}

func (s *ProjectAccessService) browserRoots(clientIP string) []string {
	if s.IsLocalRequest(clientIP) {
		return s.localBrowseRoots()
	}
	return s.normalizedAllowedRoots()
}

func (s *ProjectAccessService) localBrowseRoots() []string {
	roots := make([]string, 0)
	if runtime.GOOS == "windows" {
		for drive := 'A'; drive <= 'Z'; drive++ {
			candidate := fmt.Sprintf("%c:/", drive)
			if info, err := os.Stat(candidate); err == nil && info.IsDir() {
				roots = append(roots, filepath.Clean(candidate))
			}
		}
		return roots
	}
	root := string(os.PathSeparator)
	if info, err := os.Stat(root); err == nil && info.IsDir() {
		roots = append(roots, filepath.Clean(root))
	}
	return roots
}

// IsLocalRequest 判断当前请求来源是否为本机回环地址。
func (s *ProjectAccessService) IsLocalRequest(clientIP string) bool {
	ip := net.ParseIP(strings.TrimSpace(clientIP))
	if ip == nil {
		return false
	}
	return ip.IsLoopback()
}

func (s *ProjectAccessService) supportsNativePicker() bool {
	switch runtime.GOOS {
	case "darwin", "windows":
		return zenity.IsAvailable()
	case "linux":
		return (os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != "") && zenity.IsAvailable()
	default:
		return false
	}
}

// CanUseNativePicker 判断当前访问是否允许调用系统目录选择器。
func (s *ProjectAccessService) CanUseNativePicker(clientIP string) bool {
	cfg := s.projectConfig()
	return cfg.EnableNativeLocalPicker && s.IsLocalRequest(clientIP) && s.supportsNativePicker()
}

// GetCapabilities 返回当前访问来源对应的目录访问能力描述。
func (s *ProjectAccessService) GetCapabilities(clientIP string) RuntimeCapabilities {
	roots := s.browserRoots(clientIP)
	nativeEnabled := s.CanUseNativePicker(clientIP)
	accessMode := "remote"
	message := "当前访问来源为远程，添加项目时只能在配置的白名单目录中浏览。"
	if s.IsLocalRequest(clientIP) {
		accessMode = "local"
		if nativeEnabled {
			message = "当前访问来源为本机，可使用系统目录选择器；如需浏览器分栏选择，也不受白名单限制。"
		} else {
			message = "当前访问来源为本机，系统目录选择器不可用，可通过浏览器目录浏览任意服务器本机目录。"
		}
	}
	if !s.IsLocalRequest(clientIP) && len(roots) == 0 {
		message = "远程访问添加项目前，请先在配置中设置允许浏览的白名单目录。"
	}
	return RuntimeCapabilities{
		AccessMode:                    accessMode,
		NativeDirectoryPickerEnabled:  nativeEnabled,
		BrowserDirectoryPickerEnabled: len(roots) > 0,
		AllowedRoots:                  roots,
		Message:                       message,
	}
}

func (s *ProjectAccessService) pathAllowed(absPath string, roots []string) bool {
	for _, root := range roots {
		if isSameOrWithinPath(root, absPath) {
			return true
		}
	}
	return false
}

func (s *ProjectAccessService) validateProjectPath(input string, roots []string, enforceRoots bool) (string, error) {
	if strings.TrimSpace(input) == "" {
		return "", fmt.Errorf("项目路径不能为空")
	}
	absPath, err := filepath.Abs(input)
	if err != nil {
		return "", fmt.Errorf("解析项目路径失败")
	}
	absPath = filepath.Clean(absPath)
	info, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("项目路径不存在")
	}
	if !info.IsDir() {
		return "", fmt.Errorf("项目路径必须是目录")
	}
	if enforceRoots && !s.pathAllowed(absPath, roots) {
		return "", fmt.Errorf("项目路径不在允许的根目录范围内")
	}
	if !s.projectConfig().FollowSymlink {
		stat, err := os.Lstat(absPath)
		if err == nil && stat.Mode()&fs.ModeSymlink != 0 {
			return "", fmt.Errorf("当前配置不允许使用符号链接目录")
		}
	}
	return absPath, nil
}

// ValidateProjectPathForCreate 校验新建项目时输入的目录是否可访问。
func (s *ProjectAccessService) ValidateProjectPathForCreate(input string, clientIP string) (string, error) {
	if s.IsLocalRequest(clientIP) {
		return s.validateProjectPath(input, nil, false)
	}
	roots := s.normalizedAllowedRoots()
	if len(roots) == 0 {
		return "", fmt.Errorf("远程访问添加项目前，请先配置允许浏览的白名单目录")
	}
	return s.validateProjectPath(input, roots, true)
}

// ValidateExistingProjectPath 校验已存在项目记录中的路径是否仍然有效。
func (s *ProjectAccessService) ValidateExistingProjectPath(input string) (string, error) {
	return s.validateProjectPath(input, nil, false)
}

// ListDirectories 按目录层级返回可供前端选择的子目录列表。
func (s *ProjectAccessService) ListDirectories(parent string, clientIP string) ([]DirectoryEntry, error) {
	roots := s.browserRoots(clientIP)
	if strings.TrimSpace(parent) == "" {
		entries := make([]DirectoryEntry, 0, len(roots))
		for _, root := range roots {
			info, err := os.Stat(root)
			if err != nil || !info.IsDir() {
				continue
			}
			entries = append(entries, DirectoryEntry{
				Name:        directoryDisplayName(root),
				Path:        root,
				HasChildren: s.directoryHasChildren(root),
			})
		}
		return entries, nil
	}

	absPath, err := filepath.Abs(parent)
	if err != nil {
		return nil, fmt.Errorf("解析目录失败")
	}
	absPath = filepath.Clean(absPath)
	if !s.pathAllowed(absPath, roots) {
		return nil, fmt.Errorf("目录不在允许的根目录范围内")
	}
	children, err := os.ReadDir(absPath)
	if err != nil {
		return nil, fmt.Errorf("读取目录失败")
	}
	entries := make([]DirectoryEntry, 0)
	for _, child := range children {
		if !child.IsDir() {
			continue
		}
		if !s.projectConfig().FollowSymlink && child.Type()&fs.ModeSymlink != 0 {
			continue
		}
		childPath := filepath.Join(absPath, child.Name())
		entries = append(entries, DirectoryEntry{
			Name:        child.Name(),
			Path:        childPath,
			HasChildren: s.directoryHasChildren(childPath),
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})
	return entries, nil
}

func (s *ProjectAccessService) directoryHasChildren(dirPath string) bool {
	children, err := os.ReadDir(dirPath)
	if err != nil {
		return false
	}
	for _, child := range children {
		if child.IsDir() {
			return true
		}
	}
	return false
}
