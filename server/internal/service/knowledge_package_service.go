package service

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/google/uuid"
)

// KnowledgePackageService 实现《蔚小芯智能体.md》6.8 的标准知识包协议。
// 包结构固定为 manifest.json + resources.ndjson + attachments/（可选）。
type KnowledgePackageService struct {
	kbSvc      *KBService
	kbRepo     *repository.KBRepo
	secret     string
	chunkStore *ImportChunkStore
}

func NewKnowledgePackageService(kbSvc *KBService, kbRepo *repository.KBRepo) *KnowledgePackageService {
	return &KnowledgePackageService{
		kbSvc:      kbSvc,
		kbRepo:     kbRepo,
		chunkStore: NewImportChunkStore(),
	}
}

func (s *KnowledgePackageService) SetHMACSecret(secret string) {
	s.secret = secret
}

// ExportPackage 生成标准 zip 知识包，并返回 manifest 供 handler 记录/调试。
func (s *KnowledgePackageService) ExportPackage(
	ctx context.Context,
	resourceType, sinceCursor, callerScope, callerOwnerID string,
	limit int,
) ([]byte, *model.KnowledgePackageManifest, error) {
	if limit <= 0 {
		limit = 1000
	}
	if limit > 5000 {
		limit = 5000
	}
	exportType := "full"
	if sinceCursor != "" {
		exportType = "delta"
	}

	resources, nextCursor, err := s.kbRepo.ListSincePage(resourceType, sinceCursor, callerScope, callerOwnerID, limit)
	if err != nil {
		return nil, nil, fmt.Errorf("知识包导出查询失败: %w", err)
	}

	var ndjsonBuf bytes.Buffer
	for _, r := range resources {
		line, err := json.Marshal(r)
		if err != nil {
			return nil, nil, fmt.Errorf("知识包资源序列化失败: %w", err)
		}
		ndjsonBuf.Write(line)
		ndjsonBuf.WriteByte('\n')
	}
	resourcesSha := sha256.Sum256(ndjsonBuf.Bytes())
	resourcesShaHex := hex.EncodeToString(resourcesSha[:])

	untilCursor := sinceCursor
	if len(resources) > 0 {
		last := resources[len(resources)-1]
		untilCursor = last.UpdatedAt + "|" + last.ResourceID
	}

	manifest := &model.KnowledgePackageManifest{
		PackageID:         "pkg_" + strings.ReplaceAll(uuid.NewString()[:8], "-", ""),
		Producer:          "weixiaoxin",
		SchemaVersion:     "1.0",
		ExportType:        exportType,
		OwnerScope:        callerScope,
		OwnerID:           callerOwnerID,
		SinceCursor:       sinceCursor,
		UntilCursor:       untilCursor,
		NextCursor:        nextCursor,
		HasMore:           nextCursor != "",
		ExportBatchID:     "batch_" + uuid.NewString()[:12],
		GeneratedAt:       time.Now().UTC().Format(time.RFC3339),
		ResourceCount:     len(resources),
		HashAlg:           "sha256",
		ResourcesSha256:   resourcesShaHex,
		AttachmentsSha256: "",
	}
	if s.secret != "" {
		manifest.SignAlg = "hmac-sha256"
		manifest.Signature = computePackageSignature(s.secret, resourcesShaHex, "", untilCursor)
	}

	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, nil, err
	}

	zipBuf := &bytes.Buffer{}
	zw := zip.NewWriter(zipBuf)
	if err := writeZipEntry(zw, "manifest.json", manifestBytes); err != nil {
		return nil, nil, err
	}
	if err := writeZipEntry(zw, "resources.ndjson", ndjsonBuf.Bytes()); err != nil {
		return nil, nil, err
	}
	if err := writeZipEntry(zw, "attachments/README.txt", []byte("标准知识包附件目录（可选）。\n")); err != nil {
		return nil, nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, nil, err
	}

	return zipBuf.Bytes(), manifest, nil
}

// ImportPackage 校验并导入标准 zip 知识包。
func (s *KnowledgePackageService) ImportPackage(ctx context.Context, zipData []byte, username, traceID string) (*model.KBImportPackageResponse, error) {
	zr, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("知识包不是有效 zip: %w", err)
	}

	var manifest model.KnowledgePackageManifest
	var ndjsonData []byte
	foundManifest := false
	foundResources := false
	for _, f := range zr.File {
		switch f.Name {
		case "manifest.json":
			data, err := readZipFile(f)
			if err != nil {
				return nil, err
			}
			if err := json.Unmarshal(data, &manifest); err != nil {
				return nil, fmt.Errorf("manifest.json 解析失败: %w", err)
			}
			foundManifest = true
		case "resources.ndjson":
			ndjsonData, err = readZipFile(f)
			if err != nil {
				return nil, err
			}
			foundResources = true
		}
	}
	if !foundManifest {
		return nil, fmt.Errorf("知识包缺少 manifest.json")
	}
	if !foundResources {
		return nil, fmt.Errorf("知识包缺少 resources.ndjson")
	}

	sum := sha256.Sum256(ndjsonData)
	if hex.EncodeToString(sum[:]) != manifest.ResourcesSha256 {
		return nil, fmt.Errorf("resources.ndjson hash 校验失败")
	}
	if s.secret != "" && manifest.Signature != "" {
		expected := computePackageSignature(s.secret, manifest.ResourcesSha256, manifest.AttachmentsSha256, manifest.UntilCursor)
		if !hmac.Equal([]byte(manifest.Signature), []byte(expected)) {
			return nil, fmt.Errorf("知识包 HMAC 签名校验失败")
		}
	}

	resp, err := s.kbSvc.ImportResources(ctx, string(ndjsonData), username)
	if err != nil {
		return nil, fmt.Errorf("知识包导入落库失败: %w", err)
	}

	ignored := resp.Skipped - resp.Conflict
	if ignored < 0 {
		ignored = 0
	}
	warnings := make([]string, 0)
	for _, r := range resp.Data {
		if r.Action == "skipped" && r.Message != "" {
			warnings = append(warnings, r.ResourceID+": "+r.Message)
		}
	}

	return &model.KBImportPackageResponse{
		Code:          0,
		Message:       "导入完成",
		PackageID:     manifest.PackageID,
		ReceivedCount: resp.Total,
		AppliedCount:  resp.Created + resp.Updated,
		IgnoredCount:  ignored,
		ConflictCount: resp.Conflict,
		UntilCursor:   manifest.UntilCursor,
		Warnings:      warnings,
		TraceID:       traceID,
	}, nil
}

func writeZipEntry(zw *zip.Writer, name string, data []byte) error {
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func readZipFile(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

func computePackageSignature(secret, resourcesSha, attachmentsSha, untilCursor string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(resourcesSha))
	mac.Write([]byte(attachmentsSha))
	mac.Write([]byte(untilCursor))
	return hex.EncodeToString(mac.Sum(nil))
}
