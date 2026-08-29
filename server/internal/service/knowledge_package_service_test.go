package service_test

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"strings"
	"testing"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/service"
	"github.com/dll/wxx/server/internal/testutil"
	"github.com/stretchr/testify/require"
)

// testAdminUserCtx 返回系统管理员上下文（scope 校验恒通过，用于导入类测试）
func testAdminUserCtx() *model.UserContext {
	return &model.UserContext{
		Username:   "importer",
		Role:       "sys_admin",
		Status:     "active",
		OwnerScope: "school",
		OwnerID:    "all",
	}
}

func TestKnowledgePackageRoundTrip(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()
	kbRepo := repository.NewKBRepo(db)
	kbSvc := service.NewKBService(kbRepo, db)
	pkgSvc := service.NewKnowledgePackageService(kbSvc, kbRepo)
	pkgSvc.SetHMACSecret("test-hmac-secret")

	_, err := kbRepo.Create(&model.KBResource{
		ResourceID:   "policy_test_001",
		ResourceType: "Policy",
		OwnerScope:   "school",
		OwnerID:      "",
		RoleScope:    `["student"]`,
		Version:      "20260801-v1",
		Status:       "published",
		Title:        "测试政策",
		Summary:      "测试摘要",
		Content:      "测试正文",
		UpdatedBy:    "tester",
	})
	require.NoError(t, err)

	zipData, manifest, err := pkgSvc.ExportPackage(context.Background(), "", "", "school", "", 10)
	require.NoError(t, err)
	require.NotEmpty(t, zipData)
	require.Equal(t, 1, manifest.ResourceCount)
	require.NotEmpty(t, manifest.ResourcesSha256)
	require.NotEmpty(t, manifest.Signature)

	db2 := testutil.NewTestDB(t)
	defer db2.Close()
	kbRepo2 := repository.NewKBRepo(db2)
	kbSvc2 := service.NewKBService(kbRepo2, db2)
	pkgSvc2 := service.NewKnowledgePackageService(kbSvc2, kbRepo2)
	pkgSvc2.SetHMACSecret("test-hmac-secret")

	resp, err := pkgSvc2.ImportPackage(context.Background(), zipData, testAdminUserCtx(), "trace-1")
	require.NoError(t, err)
	require.Equal(t, 1, resp.AppliedCount)
	require.Equal(t, 0, resp.IgnoredCount)
	require.Equal(t, 0, resp.ConflictCount)
}

func TestKnowledgePackageRejectsTamperedHash(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()
	kbRepo := repository.NewKBRepo(db)
	kbSvc := service.NewKBService(kbRepo, db)
	pkgSvc := service.NewKnowledgePackageService(kbSvc, kbRepo)
	pkgSvc.SetHMACSecret("test-hmac-secret")
	_, err := kbRepo.Create(&model.KBResource{
		ResourceID: "policy_test_002", ResourceType: "Policy", OwnerScope: "school",
		RoleScope: `["student"]`, Version: "v1", Status: "published",
		Title: "测试政策二", Summary: "摘要二", Content: "测试正文二", UpdatedBy: "tester",
	})
	require.NoError(t, err)

	zipData, _, err := pkgSvc.ExportPackage(context.Background(), "", "", "school", "", 10)
	require.NoError(t, err)

	// 篡改 resources.ndjson 内容并保持 manifest 不变，导入必须被 hash 校验拦截。
	tampered := rewriteZipResource(t, zipData, "resources.ndjson", "测试", "测试x")

	_, err = pkgSvc.ImportPackage(context.Background(), tampered, testAdminUserCtx(), "trace-2")
	require.Error(t, err)
}

func TestKnowledgePackageRejectsMissingSignature(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()
	kbRepo := repository.NewKBRepo(db)
	kbSvc := service.NewKBService(kbRepo, db)
	pkgSvc := service.NewKnowledgePackageService(kbSvc, kbRepo)
	pkgSvc.SetHMACSecret("test-hmac-secret")

	_, err := kbRepo.Create(&model.KBResource{
		ResourceID: "policy_test_unsigned", ResourceType: "Policy", OwnerScope: "school",
		RoleScope: `[]`, Version: "v1", Status: "published",
		Title: "无签名政策", Summary: "摘要", Content: "正文", UpdatedBy: "tester",
	})
	require.NoError(t, err)
	zipData, _, err := pkgSvc.ExportPackage(context.Background(), "", "", "school", "", 10)
	require.NoError(t, err)

	unsigned := rewriteZipResource(t, zipData, "manifest.json", `"signature":`, `"signature_removed":`)
	_, err = pkgSvc.ImportPackage(context.Background(), unsigned, testAdminUserCtx(), "trace-unsigned")
	require.ErrorContains(t, err, "缺少强制 HMAC 签名")
}

func TestKnowledgePackageReplayReturnsOriginalResult(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()
	kbRepo := repository.NewKBRepo(db)
	kbSvc := service.NewKBService(kbRepo, db)
	pkgSvc := service.NewKnowledgePackageService(kbSvc, kbRepo)
	pkgSvc.SetHMACSecret("test-hmac-secret")

	_, err := kbRepo.Create(&model.KBResource{
		ResourceID: "policy_test_replay", ResourceType: "Policy", OwnerScope: "school",
		RoleScope: `[]`, Version: "v1", Status: "published",
		Title: "重放政策", Summary: "摘要", Content: "正文", UpdatedBy: "tester",
	})
	require.NoError(t, err)
	zipData, manifest, err := pkgSvc.ExportPackage(context.Background(), "", "", "school", "", 10)
	require.NoError(t, err)

	first, err := pkgSvc.ImportPackage(context.Background(), zipData, testAdminUserCtx(), "trace-replay-1")
	require.NoError(t, err)
	second, err := pkgSvc.ImportPackage(context.Background(), zipData, testAdminUserCtx(), "trace-replay-2")
	require.NoError(t, err)
	require.Equal(t, manifest.PackageID, second.PackageID)
	require.Equal(t, first.AppliedCount, second.AppliedCount)
	require.Equal(t, "trace-replay-1", second.TraceID)
	var receipts int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM knowledge_package_receipts WHERE package_id=?`, manifest.PackageID).Scan(&receipts))
	require.Equal(t, 1, receipts)
}

func TestKnowledgePackageRequiresHMACSecret(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()
	kbRepo := repository.NewKBRepo(db)
	kbSvc := service.NewKBService(kbRepo, db)
	pkgSvc := service.NewKnowledgePackageService(kbSvc, kbRepo)
	_, _, err := pkgSvc.ExportPackage(context.Background(), "", "", "school", "", 10)
	require.ErrorContains(t, err, "HMAC")
}

func TestKnowledgeImportChunkResume(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()
	kbRepo := repository.NewKBRepo(db)
	kbSvc := service.NewKBService(kbRepo, db)
	pkgSvc := service.NewKnowledgePackageService(kbSvc, kbRepo)
	pkgSvc.SetHMACSecret("test-hmac-secret")
	_, err := kbRepo.Create(&model.KBResource{
		ResourceID: "policy_chunk_001", ResourceType: "Policy", OwnerScope: "school",
		RoleScope: `["student"]`, Version: "v1", Status: "published",
		Title: "分片测试政策", Summary: "摘要", Content: "分片测试正文", UpdatedBy: "tester",
	})
	require.NoError(t, err)
	zipData, _, err := pkgSvc.ExportPackage(context.Background(), "", "", "school", "", 10)
	require.NoError(t, err)

	sum := sha256.Sum256(zipData)
	targetDB := testutil.NewTestDB(t)
	defer targetDB.Close()
	targetRepo := repository.NewKBRepo(targetDB)
	targetSvc := service.NewKBService(targetRepo, targetDB)
	targetPkgSvc := service.NewKnowledgePackageService(targetSvc, targetRepo)
	targetPkgSvc.SetHMACSecret("test-hmac-secret")

	initResp, err := targetPkgSvc.InitChunkUpload(3, hex.EncodeToString(sum[:]))
	require.NoError(t, err)
	require.NotEmpty(t, initResp.UploadID)
	require.Equal(t, 3, initResp.TotalChunks)

	chunkSize := (len(zipData) + 2) / 3
	require.NoError(t, targetPkgSvc.PutChunk(initResp.UploadID, 2, zipData[2*chunkSize:], ""))
	status, err := targetPkgSvc.ChunkStatus(initResp.UploadID)
	require.NoError(t, err)
	require.Equal(t, []int{2}, status.ReceivedChunks)
	require.Equal(t, []int{0, 1}, status.MissingChunks)

	require.NoError(t, targetPkgSvc.PutChunk(initResp.UploadID, 0, zipData[:chunkSize], ""))
	require.NoError(t, targetPkgSvc.PutChunk(initResp.UploadID, 1, zipData[chunkSize:2*chunkSize], ""))
	resp, err := targetPkgSvc.CompleteChunkUpload(context.Background(), initResp.UploadID, testAdminUserCtx(), "trace-chunk")
	require.NoError(t, err)
	require.Equal(t, 1, resp.AppliedCount)
}

func rewriteZipResource(t *testing.T, zipData []byte, target, old, new string) []byte {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	require.NoError(t, err)
	buf := &bytes.Buffer{}
	zw := zip.NewWriter(buf)
	for _, f := range zr.File {
		rc, err := f.Open()
		require.NoError(t, err)
		data, err := io.ReadAll(rc)
		require.NoError(t, err)
		rc.Close()
		if f.Name == target {
			data = []byte(strings.ReplaceAll(string(data), old, new))
		}
		w, err := zw.Create(f.Name)
		require.NoError(t, err)
		_, err = w.Write(data)
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())
	return buf.Bytes()
}
