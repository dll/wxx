package service

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	dbutil "github.com/dll/wxx/server/internal/db"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/google/uuid"
)

// KnowledgePackageService 实现《蔚小芯智能体.md》6.8 的标准知识包协议。
// 包结构固定为 manifest.json + resources.ndjson + attachments/（可选）。
type KnowledgePackageService struct {
	kbSvc      *KBService
	kbRepo     *repository.KBRepo
	db         *sql.DB
	secret     string
	chunkStore *ImportChunkStore
}

func NewKnowledgePackageService(kbSvc *KBService, kbRepo *repository.KBRepo) *KnowledgePackageService {
	return &KnowledgePackageService{
		kbSvc:      kbSvc,
		kbRepo:     kbRepo,
		db:         kbRepo.DB(),
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
	if strings.TrimSpace(s.secret) == "" {
		return nil, nil, fmt.Errorf("知识包 HMAC 密钥未配置，禁止导出")
	}
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
	manifest.SignAlg = "hmac-sha256"
	manifest.Signature = computePackageSignature(s.secret, resourcesShaHex, "", untilCursor)

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
func (s *KnowledgePackageService) ImportPackage(ctx context.Context, zipData []byte, userCtx *model.UserContext, traceID string) (*model.KBImportPackageResponse, error) {
	if strings.TrimSpace(s.secret) == "" {
		return nil, fmt.Errorf("知识包 HMAC 密钥未配置，禁止导入")
	}
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
	if manifest.SignAlg != "hmac-sha256" || strings.TrimSpace(manifest.Signature) == "" {
		return nil, fmt.Errorf("知识包缺少强制 HMAC 签名")
	}
	expected := computePackageSignature(s.secret, manifest.ResourcesSha256, manifest.AttachmentsSha256, manifest.UntilCursor)
	if !hmac.Equal([]byte(manifest.Signature), []byte(expected)) {
		return nil, fmt.Errorf("知识包 HMAC 签名校验失败")
	}

	// package_id 是协议级幂等键。先登记处理中状态，避免并发重放重复执行导入。
	if cached, err := s.getPackageReceipt(manifest.PackageID); err != nil {
		return nil, fmt.Errorf("查询知识包接收记录失败: %w", err)
	} else if cached != nil {
		if cached.Status == "completed" && cached.ResponseJSON != "" {
			var cachedResp model.KBImportPackageResponse
			if err := json.Unmarshal([]byte(cached.ResponseJSON), &cachedResp); err != nil {
				return nil, fmt.Errorf("知识包接收记录损坏: %w", err)
			}
			return &cachedResp, nil
		}
		return nil, fmt.Errorf("知识包正在处理或此前处理失败，禁止重复执行 package_id=%s", manifest.PackageID)
	} else if claimed, err := s.createPackageReceipt(manifest, traceID); err != nil {
		// 并发请求可能已经抢先登记；重新读取并按幂等语义处理。
		if cached, readErr := s.getPackageReceipt(manifest.PackageID); readErr == nil && cached != nil && cached.Status == "completed" && cached.ResponseJSON != "" {
			var cachedResp model.KBImportPackageResponse
			if jsonErr := json.Unmarshal([]byte(cached.ResponseJSON), &cachedResp); jsonErr == nil {
				return &cachedResp, nil
			}
		}
		return nil, fmt.Errorf("登记知识包接收记录失败: %w", err)
	} else if !claimed {
		return nil, fmt.Errorf("知识包正在处理或此前处理失败，禁止重复执行 package_id=%s", manifest.PackageID)
	}

	resp, err := s.kbSvc.ImportResources(ctx, string(ndjsonData), userCtx)
	if err != nil {
		_, _ = s.db.Exec(`UPDATE knowledge_package_receipts SET status='failed', error_message=?, updated_at=CURRENT_TIMESTAMP WHERE package_id=?`, err.Error(), manifest.PackageID)
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

	result := &model.KBImportPackageResponse{
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
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("序列化知识包接收结果失败: %w", err)
	}
	if _, err := s.db.Exec(`UPDATE knowledge_package_receipts SET status='completed', response_json=?, updated_at=CURRENT_TIMESTAMP WHERE package_id=?`, string(encoded), manifest.PackageID); err != nil {
		return nil, fmt.Errorf("更新知识包接收记录失败: %w", err)
	}
	return result, nil
}

type packageReceipt struct {
	Status       string
	ResponseJSON string
	ErrorMessage string
}

func (s *KnowledgePackageService) getPackageReceipt(packageID string) (*packageReceipt, error) {
	var receipt packageReceipt
	err := s.db.QueryRow(`SELECT status, response_json, error_message FROM knowledge_package_receipts WHERE package_id=?`, packageID).Scan(&receipt.Status, &receipt.ResponseJSON, &receipt.ErrorMessage)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &receipt, nil
}

func (s *KnowledgePackageService) createPackageReceipt(manifest model.KnowledgePackageManifest, traceID string) (bool, error) {
	result, err := s.db.Exec(dbutil.InsertIgnore(dbutil.DriverOf(s.db))+` knowledge_package_receipts
		(package_id, producer, signature, trace_id, status, response_json)
		VALUES (?, ?, ?, ?, 'processing', '')`, manifest.PackageID, manifest.Producer, manifest.Signature, traceID)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	return rows == 1, err
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
