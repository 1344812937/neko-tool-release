package service

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
	"time"

	"neko-tool/internal/models"

	"github.com/sergi/go-diff/diffmatchpatch"
)

const (
	maxProjectSyncLogTextBytes = 15 * 1024 * 1024

	logContentEncodingNone   = "none"
	logContentEncodingText   = "text"
	logContentEncodingBinary = "base64"

	logStorageKindNone                   = "none"
	logStorageKindFullText               = "full_text"
	logStorageKindCompressedFullText     = "compressed_full_text"
	logStorageKindReversePatch           = "reverse_patch"
	logStorageKindCompressedReversePatch = "compressed_reverse_patch"
	logStorageKindHashOnly               = "hash_only"
	logStorageKindLegacyFull             = "legacy_full"

	logOmittedReasonBinary    = "binary"
	logOmittedReasonSizeLimit = "size_limit"

	logDiffAlgorithmDMP = "diff_match_patch"
)

type logContentSnapshot struct {
	Encoding      string
	Content       string
	StorageKind   string
	ContentSize   int64
	OmittedReason string
}

type logContentMeta struct {
	Encoding      string
	StorageKind   string
	ContentSize   int64
	OmittedReason string
}

func buildProjectSyncLogSnapshots(beforeFile, afterFile FileSide) (logContentSnapshot, logContentSnapshot, string, error) {
	beforeExists := beforeFile.Exists
	afterExists := afterFile.Exists
	if !beforeExists && !afterExists {
		return buildNoneLogSnapshot(), buildNoneLogSnapshot(), "", nil
	}
	if !shouldStoreFullLogContent(beforeFile, afterFile) {
		reason := resolveProjectSyncLogOmittedReason(beforeFile, afterFile)
		return buildHashOnlyLogSnapshot(beforeFile, reason), buildHashOnlyLogSnapshot(afterFile, reason), "", nil
	}
	if beforeExists && afterExists {
		reversePatch, err := buildReversePatch(afterFile.Content, beforeFile.Content)
		if err != nil {
			return logContentSnapshot{}, logContentSnapshot{}, "", err
		}
		return logContentSnapshot{
				Encoding:    logContentEncodingText,
				Content:     compressProjectSyncLogText(reversePatch),
				StorageKind: logStorageKindCompressedReversePatch,
				ContentSize: beforeFile.Size,
			}, logContentSnapshot{
				Encoding:    logContentEncodingText,
				Content:     compressProjectSyncLogText(afterFile.Content),
				StorageKind: logStorageKindCompressedFullText,
				ContentSize: afterFile.Size,
			}, logDiffAlgorithmDMP, nil
	}
	if beforeExists {
		return buildFullTextLogSnapshot(beforeFile), buildNoneLogSnapshot(), "", nil
	}
	return buildNoneLogSnapshot(), buildFullTextLogSnapshot(afterFile), "", nil
}

func buildProjectSyncLogDetail(entry models.ProjectSyncLog) (ProjectSyncLogDetail, error) {
	logID, createTime, modifyTime, sortValue, valid := extractProjectSyncLogBaseFields(entry)
	beforeMeta := normalizeLogContentMeta(entry.BeforeEncoding, entry.BeforeStorageKind, entry.BeforeContent, entry.BeforeContentSize, entry.BeforeOmittedReason)
	afterMeta := normalizeLogContentMeta(entry.AfterEncoding, entry.AfterStorageKind, entry.AfterContent, entry.AfterContentSize, entry.AfterOmittedReason)
	beforeContent, err := restoreProjectSyncLogContent(beforeMeta, entry.BeforeContent, afterMeta, entry.AfterContent, entry.DiffAlgorithm)
	if err != nil {
		return ProjectSyncLogDetail{}, err
	}
	afterContent := visibleProjectSyncLogContent(afterMeta, entry.AfterContent)
	return ProjectSyncLogDetail{
		Id:                  logID,
		CreateTime:          createTime,
		ModifyTime:          modifyTime,
		Valid:               valid,
		ChangeType:          entry.ChangeType,
		ScopeType:           entry.ScopeType,
		RelativePath:        entry.RelativePath,
		SourceNodeId:        entry.SourceNodeId,
		SourceNodeName:      entry.SourceNodeName,
		SourceProjectId:     entry.SourceProjectId,
		SourceProjectName:   entry.SourceProjectName,
		TargetNodeId:        entry.TargetNodeId,
		TargetNodeName:      entry.TargetNodeName,
		TargetProjectId:     entry.TargetProjectId,
		TargetProjectName:   entry.TargetProjectName,
		ExecutorNodeName:    entry.ExecutorNodeName,
		ExecutorNodeAddress: entry.ExecutorNodeAddress,
		OperatorIP:          entry.OperatorIP,
		BeforeExists:        entry.BeforeExists,
		BeforeHash:          entry.BeforeHash,
		BeforeEncoding:      beforeMeta.Encoding,
		BeforeContent:       beforeContent,
		BeforeStorageKind:   beforeMeta.StorageKind,
		BeforeContentSize:   beforeMeta.ContentSize,
		BeforeOmittedReason: beforeMeta.OmittedReason,
		AfterHash:           entry.AfterHash,
		AfterEncoding:       afterMeta.Encoding,
		AfterContent:        afterContent,
		AfterStorageKind:    afterMeta.StorageKind,
		AfterContentSize:    afterMeta.ContentSize,
		AfterOmittedReason:  afterMeta.OmittedReason,
		DiffAlgorithm:       normalizeDiffAlgorithm(entry.DiffAlgorithm),
		OperatedAt:          entry.OperatedAt,
		Sort:                sortValue,
	}, nil
}

func shouldStoreFullLogContent(beforeFile, afterFile FileSide) bool {
	if beforeFile.Exists && !isProjectSyncLogTextEligible(beforeFile) {
		return false
	}
	if afterFile.Exists && !isProjectSyncLogTextEligible(afterFile) {
		return false
	}
	return true
}

func isProjectSyncLogTextEligible(fileSide FileSide) bool {
	return fileSide.Text && fileSide.Size <= maxProjectSyncLogTextBytes
}

func resolveProjectSyncLogOmittedReason(beforeFile, afterFile FileSide) string {
	if (beforeFile.Exists && !beforeFile.Text) || (afterFile.Exists && !afterFile.Text) {
		return logOmittedReasonBinary
	}
	return logOmittedReasonSizeLimit
}

func buildNoneLogSnapshot() logContentSnapshot {
	return logContentSnapshot{Encoding: logContentEncodingNone, StorageKind: logStorageKindNone}
}

func buildFullTextLogSnapshot(fileSide FileSide) logContentSnapshot {
	return logContentSnapshot{
		Encoding:    logContentEncodingText,
		Content:     compressProjectSyncLogText(fileSide.Content),
		StorageKind: logStorageKindCompressedFullText,
		ContentSize: fileSide.Size,
	}
}

func buildHashOnlyLogSnapshot(fileSide FileSide, omittedReason string) logContentSnapshot {
	if !fileSide.Exists {
		return buildNoneLogSnapshot()
	}
	encoding := logContentEncodingText
	if !fileSide.Text {
		encoding = logContentEncodingBinary
	}
	return logContentSnapshot{
		Encoding:      encoding,
		StorageKind:   logStorageKindHashOnly,
		ContentSize:   fileSide.Size,
		OmittedReason: omittedReason,
	}
}

func buildReversePatch(afterContent, beforeContent string) (string, error) {
	dmp := diffmatchpatch.New()
	patches := dmp.PatchMake(afterContent, beforeContent)
	return dmp.PatchToText(patches), nil
}

func restoreReversePatch(afterContent, reversePatch string) (string, error) {
	dmp := diffmatchpatch.New()
	patches, err := dmp.PatchFromText(reversePatch)
	if err != nil {
		return "", err
	}
	restored, results := dmp.PatchApply(patches, afterContent)
	for _, ok := range results {
		if !ok {
			return "", fmt.Errorf("日志差异补丁回放失败")
		}
	}
	return restored, nil
}

func normalizeLogContentMeta(encoding, storageKind, content string, contentSize int64, omittedReason string) logContentMeta {
	meta := logContentMeta{
		Encoding:      strings.TrimSpace(encoding),
		StorageKind:   strings.TrimSpace(storageKind),
		ContentSize:   contentSize,
		OmittedReason: strings.TrimSpace(omittedReason),
	}
	if meta.Encoding == "" {
		meta.Encoding = logContentEncodingNone
	}
	if meta.StorageKind == "" {
		switch meta.Encoding {
		case logContentEncodingNone:
			meta.StorageKind = logStorageKindNone
		case logContentEncodingText, logContentEncodingBinary:
			meta.StorageKind = logStorageKindLegacyFull
		default:
			meta.StorageKind = logStorageKindLegacyFull
		}
	}
	if meta.ContentSize == 0 {
		switch meta.StorageKind {
		case logStorageKindFullText, logStorageKindCompressedFullText, logStorageKindLegacyFull, logStorageKindReversePatch, logStorageKindCompressedReversePatch:
			meta.ContentSize = int64(len(content))
		}
	}
	return meta
}

func visibleProjectSyncLogContent(meta logContentMeta, content string) string {
	switch meta.StorageKind {
	case logStorageKindFullText, logStorageKindLegacyFull:
		return content
	case logStorageKindCompressedFullText:
		decoded, err := decompressProjectSyncLogText(content)
		if err != nil {
			return ""
		}
		return decoded
	default:
		return ""
	}
}

func restoreProjectSyncLogContent(meta logContentMeta, rawContent string, peerMeta logContentMeta, peerContent, diffAlgorithm string) (string, error) {
	switch meta.StorageKind {
	case logStorageKindFullText, logStorageKindLegacyFull:
		return rawContent, nil
	case logStorageKindCompressedFullText:
		return decompressProjectSyncLogText(rawContent)
	case logStorageKindReversePatch:
		if normalizeDiffAlgorithm(diffAlgorithm) != logDiffAlgorithmDMP {
			return "", fmt.Errorf("不支持的日志差异算法: %s", diffAlgorithm)
		}
		if peerMeta.StorageKind != logStorageKindFullText && peerMeta.StorageKind != logStorageKindCompressedFullText && peerMeta.StorageKind != logStorageKindLegacyFull {
			return "", fmt.Errorf("日志缺少可回放的完整内容")
		}
		return restoreReversePatch(visibleProjectSyncLogContent(peerMeta, peerContent), rawContent)
	case logStorageKindCompressedReversePatch:
		if normalizeDiffAlgorithm(diffAlgorithm) != logDiffAlgorithmDMP {
			return "", fmt.Errorf("不支持的日志差异算法: %s", diffAlgorithm)
		}
		if peerMeta.StorageKind != logStorageKindFullText && peerMeta.StorageKind != logStorageKindCompressedFullText && peerMeta.StorageKind != logStorageKindLegacyFull {
			return "", fmt.Errorf("日志缺少可回放的完整内容")
		}
		decodedPatch, err := decompressProjectSyncLogText(rawContent)
		if err != nil {
			return "", err
		}
		return restoreReversePatch(visibleProjectSyncLogContent(peerMeta, peerContent), decodedPatch)
	default:
		return "", nil
	}
}

func normalizeDiffAlgorithm(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return logDiffAlgorithmDMP
	}
	return trimmed
}

func logEntryCompareHash(encoding, storageKind, content, hash string) string {
	if strings.TrimSpace(encoding) == logContentEncodingText {
		meta := normalizeLogContentMeta(encoding, storageKind, content, 0, "")
		if meta.StorageKind == logStorageKindFullText || meta.StorageKind == logStorageKindCompressedFullText || meta.StorageKind == logStorageKindLegacyFull {
			visible := visibleProjectSyncLogContent(meta, content)
			if visible != "" {
				return normalizedTextHash([]byte(visible))
			}
		}
	}
	return hash
}

func compressProjectSyncLogText(content string) string {
	if content == "" {
		return ""
	}
	var buffer bytes.Buffer
	writer := gzip.NewWriter(&buffer)
	_, _ = writer.Write([]byte(content))
	_ = writer.Close()
	return base64.StdEncoding.EncodeToString(buffer.Bytes())
}

func decompressProjectSyncLogText(content string) (string, error) {
	if strings.TrimSpace(content) == "" {
		return "", nil
	}
	compressed, err := base64.StdEncoding.DecodeString(content)
	if err != nil {
		return "", err
	}
	reader, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return "", err
	}
	defer reader.Close()
	decoded, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

func logEntryContentExists(encoding, storageKind string) bool {
	meta := normalizeLogContentMeta(encoding, storageKind, "", 0, "")
	return meta.StorageKind != logStorageKindNone
}

func newProjectSyncLogDetailFromRow(row models.ProjectSyncLog) (ProjectSyncLogDetail, error) {
	return buildProjectSyncLogDetail(row)
}

func extractProjectSyncLogBaseFields(entry models.ProjectSyncLog) (uint64, *time.Time, *time.Time, *int, int) {
	if entry.BaseModel == nil {
		return 0, nil, nil, nil, 0
	}
	return derefUint64(entry.Id), entry.CreateTime, entry.ModifyTime, entry.Sort, entry.Valid
}

type ProjectSyncLogDetail struct {
	Id                  uint64     `json:"id"`
	Valid               int        `json:"valid"`
	Sort                *int       `json:"sort"`
	CreateTime          *time.Time `json:"createTime"`
	ModifyTime          *time.Time `json:"modifyTime"`
	ChangeType          string     `json:"changeType"`
	ScopeType           string     `json:"scopeType"`
	RelativePath        string     `json:"relativePath"`
	SourceNodeId        uint64     `json:"sourceNodeId"`
	SourceNodeName      string     `json:"sourceNodeName"`
	SourceProjectId     uint64     `json:"sourceProjectId"`
	SourceProjectName   string     `json:"sourceProjectName"`
	TargetNodeId        uint64     `json:"targetNodeId"`
	TargetNodeName      string     `json:"targetNodeName"`
	TargetProjectId     uint64     `json:"targetProjectId"`
	TargetProjectName   string     `json:"targetProjectName"`
	ExecutorNodeName    string     `json:"executorNodeName"`
	ExecutorNodeAddress string     `json:"executorNodeAddress"`
	OperatorIP          string     `json:"operatorIP"`
	BeforeExists        bool       `json:"beforeExists"`
	BeforeHash          string     `json:"beforeHash"`
	BeforeEncoding      string     `json:"beforeEncoding"`
	BeforeContent       string     `json:"beforeContent"`
	BeforeStorageKind   string     `json:"beforeStorageKind"`
	BeforeContentSize   int64      `json:"beforeContentSize"`
	BeforeOmittedReason string     `json:"beforeOmittedReason"`
	AfterHash           string     `json:"afterHash"`
	AfterEncoding       string     `json:"afterEncoding"`
	AfterContent        string     `json:"afterContent"`
	AfterStorageKind    string     `json:"afterStorageKind"`
	AfterContentSize    int64      `json:"afterContentSize"`
	AfterOmittedReason  string     `json:"afterOmittedReason"`
	DiffAlgorithm       string     `json:"diffAlgorithm"`
	OperatedAt          *time.Time `json:"operatedAt"`
}
