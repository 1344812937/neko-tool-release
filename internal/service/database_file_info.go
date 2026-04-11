package service

import (
	"fmt"
	"os"
	"path/filepath"

	"neko-tool/internal/core/ds/providers"
)

type DatabaseFileInfo struct {
	SizeBytes int64
	SizeLabel string
}

func readPrimaryDatabaseFileInfo() DatabaseFileInfo {
	path := providers.PrimaryDBPath()
	if absolutePath, err := filepath.Abs(path); err == nil {
		path = absolutePath
	}
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DatabaseFileInfo{SizeBytes: 0, SizeLabel: formatDatabaseSize(0)}
		}
		return DatabaseFileInfo{SizeBytes: 0, SizeLabel: "-"}
	}
	sizeBytes := info.Size()
	return DatabaseFileInfo{SizeBytes: sizeBytes, SizeLabel: formatDatabaseSize(sizeBytes)}
}

func formatDatabaseSize(sizeBytes int64) string {
	if sizeBytes <= 0 {
		return "0 MB"
	}
	const (
		mb = 1024 * 1024
		gb = 1024 * 1024 * 1024
	)
	if sizeBytes >= gb {
		return fmt.Sprintf("%.2f GB", float64(sizeBytes)/float64(gb))
	}
	return fmt.Sprintf("%.2f MB", float64(sizeBytes)/float64(mb))
}
