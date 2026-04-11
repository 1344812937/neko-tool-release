package service

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

var filesystemCaseSensitivityCache sync.Map

func pathCompareKey(path string) string {
	cleaned := filepath.Clean(path)
	return pathCompareKeyWithSensitivity(cleaned, filesystemCaseSensitive(cleaned))
}

func pathCompareKeyWithSensitivity(path string, caseSensitive bool) string {
	cleaned := filepath.Clean(path)
	if caseSensitive {
		return cleaned
	}
	return strings.ToLower(cleaned)
}

func relativePathCompareKey(path string, caseSensitive bool) string {
	cleaned := normalizeRelativePath(path)
	if caseSensitive {
		return cleaned
	}
	return strings.ToLower(cleaned)
}

func filesystemCaseSensitive(path string) bool {
	cacheKey := nearestExistingPath(filepath.Clean(path))
	if value, ok := filesystemCaseSensitivityCache.Load(cacheKey); ok {
		return value.(bool)
	}
	detected := detectFilesystemCaseSensitivity(cacheKey)
	filesystemCaseSensitivityCache.Store(cacheKey, detected)
	return detected
}

func nearestExistingPath(path string) string {
	probe := filepath.Clean(path)
	for {
		if _, err := os.Stat(probe); err == nil {
			return probe
		}
		next := filepath.Dir(probe)
		if next == probe {
			return probe
		}
		probe = next
	}
}

func detectFilesystemCaseSensitivity(path string) bool {
	if runtime.GOOS == "windows" {
		return false
	}
	probe := nearestExistingPath(path)
	for {
		parent := filepath.Dir(probe)
		if parent == probe {
			break
		}
		baseName := filepath.Base(probe)
		alternativeName := swapCaseCandidate(baseName)
		if alternativeName == "" || alternativeName == baseName {
			probe = parent
			continue
		}
		entries, err := os.ReadDir(parent)
		if err != nil {
			break
		}
		hasBaseExact := false
		hasAlternativeExact := false
		for _, entry := range entries {
			name := entry.Name()
			if name == baseName {
				hasBaseExact = true
			}
			if name == alternativeName {
				hasAlternativeExact = true
			}
		}
		if hasAlternativeExact {
			return true
		}
		if _, err := os.Stat(filepath.Join(parent, alternativeName)); err == nil && hasBaseExact {
			return false
		}
		return true
	}
	switch runtime.GOOS {
	case "darwin":
		return false
	default:
		return true
	}
}

func swapCaseCandidate(name string) string {
	for index, char := range name {
		switch {
		case char >= 'a' && char <= 'z':
			return name[:index] + strings.ToUpper(string(char)) + name[index+len(string(char)):]
		case char >= 'A' && char <= 'Z':
			return name[:index] + strings.ToLower(string(char)) + name[index+len(string(char)):]
		}
	}
	return ""
}

func resolveExistingRelativePathCaseInsensitive(rootPath, relativePath string) (string, bool, bool, error) {
	normalizedPath := normalizeRelativePath(relativePath)
	if normalizedPath == "" {
		return "", true, true, nil
	}
	segments := strings.Split(normalizedPath, "/")
	currentAbsPath := filepath.Clean(rootPath)
	currentRelativePath := ""
	exactMatch := true
	for _, segment := range segments {
		entries, err := os.ReadDir(currentAbsPath)
		if err != nil {
			return "", false, false, err
		}
		exactName := ""
		foldedMatches := make([]string, 0, 1)
		for _, entry := range entries {
			name := entry.Name()
			if name == segment {
				exactName = name
				break
			}
			if strings.EqualFold(name, segment) {
				foldedMatches = append(foldedMatches, name)
			}
		}
		chosenName := exactName
		if chosenName == "" {
			switch len(foldedMatches) {
			case 0:
				return normalizedPath, false, false, nil
			case 1:
				chosenName = foldedMatches[0]
				exactMatch = false
			default:
				return "", false, false, fmt.Errorf("路径 %s 在当前工作站存在大小写冲突，请先整理目录后再操作", normalizedPath)
			}
		}
		if currentRelativePath == "" {
			currentRelativePath = chosenName
		} else {
			currentRelativePath += "/" + chosenName
		}
		currentAbsPath = filepath.Join(currentAbsPath, chosenName)
	}
	return currentRelativePath, true, exactMatch, nil
}

func isSameOrWithinPath(rootPath, targetPath string) bool {
	rootPath = filepath.Clean(rootPath)
	targetPath = filepath.Clean(targetPath)
	relPath, err := filepath.Rel(rootPath, targetPath)
	if err != nil {
		return false
	}
	relPath = filepath.Clean(relPath)
	if relPath == "." {
		return true
	}
	return relPath != ".." && !strings.HasPrefix(relPath, ".."+string(os.PathSeparator))
}

func directoryDisplayName(path string) string {
	cleaned := filepath.Clean(path)
	if runtime.GOOS == "windows" {
		if volume := filepath.VolumeName(cleaned); volume != "" {
			remainder := strings.TrimPrefix(cleaned, volume)
			if remainder == "" || remainder == string(os.PathSeparator) {
				return volume + string(os.PathSeparator)
			}
		}
	}
	baseName := filepath.Base(cleaned)
	if baseName == "." || baseName == string(os.PathSeparator) || baseName == "" {
		return cleaned
	}
	return baseName
}
