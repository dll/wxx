package handler

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/gin-gonic/gin"
)

// 私有文件存储层约束：
//   - 仅写本地受控目录（默认 .uploads/vopc 下按项目隔离）；
//   - object_key 由加密安全随机数生成（32 字节 hex），不可猜、不携带原始文件名；
//   - 数据库只存 object_key 与元数据，不存磁盘绝对路径；
//   - 对外只返回 object_key，绝不返回可猜磁盘路径或可静态访问 URL。
//
// 真实云对象存储 / 病毒扫描 / 签名 URL 等强外部能力标注 [blocked]：需要真实云凭据，
// 本批次不做伪造。默认 storage_status=pending，表示文件已落盘但尚未经外部扫描确认。

const (
	// vopcMaxFileBytes 私有文件上限（默认 20 MB）。
	vopcMaxFileBytes = 20 << 20
	// vopcMaxFileNameRunes 原始文件名最大长度（按字符计，用于展示）。
	vopcMaxFileNameRunes = 255
)

// vopcAllowedMimeTypes 私有文件 MIME 白名单。仅允许常见、低风险的文档/图片/归档/代码类。
var vopcAllowedMimeTypes = setOf(
	"application/pdf",
	"text/plain",
	"text/markdown",
	"text/csv",
	"application/msword",
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	"application/vnd.ms-excel",
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	"application/vnd.ms-powerpoint",
	"application/vnd.openxmlformats-officedocument.presentationml.presentation",
	"application/json",
	"application/zip",
	"application/gzip",
	"application/x-tar",
	"image/png",
	"image/jpeg",
	"image/gif",
	"image/webp",
)

// validVOPCStorageStatus 受控文件存储状态白名单。
var validVOPCStorageStatus = setOf("pending", "ready", "scan_ok", "scan_failed")

// SetUploadDir 注入私有文件受控存储根目录（默认 ".uploads/vopc"）。
// 仅测试/装配层调用；生产由 app.go 传入，缺省使用仓库内受 gitignore 保护的 .uploads/vopc。
func (h *VOPCHandler) SetUploadDir(dir string) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		dir = ".uploads/vopc"
	}
	h.uploadDir = dir
}

// resolveUploadDir 懒初始化受控存储根目录，返回项目隔离子目录绝对路径。
func (h *VOPCHandler) resolveUploadDir(projectID int64) (string, error) {
	root := h.uploadDir
	if root == "" {
		root = ".uploads/vopc"
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	dir := filepath.Join(abs, strconv.FormatInt(projectID, 10))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

// newObjectKey 生成 32 字节加密安全随机的 hex key，不可猜。
func newObjectKey() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func sha256Hex(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// safeFileName 仅用于展示原始文件名；仅取 filepath.Base 去路径、拒绝空与危险字符，
// 但绝不据此构造磁盘路径（磁盘落盘用 object_key）。
func safeFileName(name string) (string, bool) {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "\\", "/")
	name = filepath.Base(name)
	if name == "." || name == ".." || name == "" {
		return "", false
	}
	if utf8.RuneCountInString(name) > vopcMaxFileNameRunes {
		return "", false
	}
	if strings.ContainsAny(name, "\r\n\x00") {
		return "", false
	}
	return name, true
}

// UploadFile POST /api/v1/vopc/projects/:id/files
// 项目写权限 + 学院准入 + 大小/MIME 白名单 + 落盘受控目录 + 不可猜 object_key。
// 返回 object_key（不返回可猜 URL / 磁盘路径）。
func (h *VOPCHandler) UploadFile(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	u := middleware.GetUserContext(c)
	if u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未认证"})
		return
	}
	fh, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请选择要上传的私有文件"})
		return
	}
	if fh.Size <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "空文件不允许上传"})
		return
	}
	if fh.Size > vopcMaxFileBytes {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"code": 413, "message": fmt.Sprintf("文件超过大小上限 %d MB", vopcMaxFileBytes>>20)})
		return
	}
	// MIME 白名单：优先用 multipart 头声明，再以扩展名兜底判断；均不在白名单即拒绝。
	declared := strings.ToLower(strings.TrimSpace(fh.Header.Get("Content-Type")))
	if declared == "" || declared == "application/octet-stream" {
		declared = detectMimeByExt(fh.Filename)
	}
	if !vopcAllowedMimeTypes[declared] {
		c.JSON(http.StatusUnsupportedMediaType, gin.H{"code": 415, "message": "不支持的文件类型：" + declared})
		return
	}
	displayName, okName := safeFileName(fh.Filename)
	if !okName {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "文件名无效"})
		return
	}

	// 事务 + 审计 + 落盘需原子：先开启事务校验权限，落盘成功后写记录并提交。
	tx, err := h.db.Begin()
	if err != nil {
		serverError(c, "文件上传失败")
		return
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	var owner int64
	if err := tx.QueryRow(`SELECT owner_user_id FROM vopc_projects WHERE id=?`, id).Scan(&owner); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "项目不存在或无权操作"})
		} else {
			serverError(c, "文件上传失败")
		}
		return
	}
	allowed, err := projectPolicy(tx, id, u.UserID, owner, "manage")
	if err != nil || !allowed {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "项目不存在或无权操作"})
		return
	}
	if blocked, msg := projectBlockedForWrite(tx, id); blocked {
		c.JSON(http.StatusConflict, gin.H{"code": 409, "message": msg})
		return
	}

	objectKey, err := newObjectKey()
	if err != nil {
		serverError(c, "文件上传失败")
		return
	}
	dir, err := h.resolveUploadDir(id)
	if err != nil {
		serverError(c, "文件上传失败")
		return
	}
	diskPath := filepath.Join(dir, objectKey)

	// 流式写盘（受控目录，object_key 无路径成分，杜绝路径穿越）。
	if err := persistMultipartFile(fh, diskPath); err != nil {
		_ = os.Remove(diskPath)
		serverError(c, "文件写入失败")
		return
	}
	sum, err := fileSHA256(diskPath)
	if err != nil {
		_ = os.Remove(diskPath)
		serverError(c, "文件校验失败")
		return
	}

	res, err := tx.Exec(`INSERT INTO vopc_files(project_id,uploader_user_id,object_key,file_name,mime_type,size_bytes,checksum,storage_status) VALUES(?,?,?,?,?,?,?,'pending')`,
		id, u.UserID, objectKey, displayName, declared, fh.Size, sum)
	if err != nil {
		_ = os.Remove(diskPath)
		serverError(c, "文件记录写入失败")
		return
	}
	fileID, _ := res.LastInsertId()
	if err := writeEvent(tx, id, u.UserID, "file.uploaded", "", "pending", fmt.Sprintf("私有文件 #%d（%s）", fileID, displayName)); err != nil {
		_ = os.Remove(diskPath)
		serverError(c, "审计写入失败")
		return
	}
	if err := tx.Commit(); err != nil {
		_ = os.Remove(diskPath)
		serverError(c, "文件上传失败")
		return
	}
	committed = true

	c.JSON(http.StatusCreated, gin.H{
		"code": 0,
		"data": gin.H{
			"id":             fileID,
			"object_key":     objectKey,
			"file_name":      displayName,
			"mime_type":      declared,
			"size_bytes":     fh.Size,
			"checksum":       sum,
			"storage_status": "pending",
		},
	})
}

// DownloadFile GET /api/v1/vopc/projects/:id/files/:key
// 服务端流式返回；下载前复核学院准入 + 项目成员/角色 + 字段权限；未授权 404。
// 不暴露真实磁盘路径。
func (h *VOPCHandler) DownloadFile(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	key := strings.TrimSpace(c.Param("key"))
	if !validObjectKey(key) {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "文件不存在或无权访问"})
		return
	}
	u := middleware.GetUserContext(c)
	if u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未认证"})
		return
	}
	tx, err := h.db.Begin()
	if err != nil {
		serverError(c, "文件下载失败")
		return
	}
	defer tx.Rollback()

	var owner int64
	if err := tx.QueryRow(`SELECT owner_user_id FROM vopc_projects WHERE id=?`, id).Scan(&owner); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "文件不存在或无权访问"})
		return
	}
	allowed, err := projectPolicy(tx, id, u.UserID, owner, "read")
	if err != nil || !allowed {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "文件不存在或无权访问"})
		return
	}
	// 字段级复核：复核学院准入（与 CollegeAccess 中间件一致性，防令牌伪造旁路）。
	if u.OwnerScope != "college" || !strings.EqualFold(u.OwnerID, h.collegeID) || u.Role == "guest" || u.Status != "active" {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "文件不存在或无权访问"})
		return
	}
	var fileName, mimeType, storageStatus, sum string
	var size int64
	err = tx.QueryRow(`SELECT file_name,mime_type,size_bytes,storage_status,checksum FROM vopc_files WHERE project_id=? AND object_key=?`, id, key).Scan(&fileName, &mimeType, &size, &storageStatus, &sum)
	if errors.Is(err, sql.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "文件不存在或无权访问"})
		return
	}
	if err != nil {
		serverError(c, "文件下载失败")
		return
	}
	tx.Rollback()

	dir, err := h.resolveUploadDir(id)
	if err != nil {
		serverError(c, "文件下载失败")
		return
	}
	diskPath := filepath.Join(dir, key)
	f, err := os.Open(diskPath)
	if err != nil {
		// 记录存在但磁盘缺失：视为不存在，避免泄露状态。
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "文件不存在或无权访问"})
		return
	}
	defer f.Close()

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", urlPathEscape(fileName)))
	if mimeType != "" && mimeType != "application/octet-stream" {
		c.Header("Content-Type", mimeType)
	} else {
		c.Header("Content-Type", "application/octet-stream")
	}
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("X-Checksum-Sha256", sum)
	c.Header("Content-Length", strconv.FormatInt(size, 10))
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, f)
}

func validObjectKey(key string) bool {
	if len(key) != 64 {
		return false
	}
	for _, r := range key {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			return false
		}
	}
	return true
}

func urlPathEscape(s string) string {
	// RFC 5987 filename* 需要 percent-encoding 非 ASCII 与保留字符。
	const hexDigits = "0123456789ABCDEF"
	var b strings.Builder
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("-._~", r) {
			b.WriteRune(r)
			continue
		}
		for _, by := range []byte(string(r)) {
			b.WriteByte('%')
			b.WriteByte(hexDigits[by>>4])
			b.WriteByte(hexDigits[by&0x0f])
		}
	}
	return b.String()
}

func persistMultipartFile(fh *multipart.FileHeader, diskPath string) error {
	src, err := fh.Open()
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := os.OpenFile(diskPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(dst, src); err != nil {
		_ = dst.Close()
		_ = os.Remove(diskPath)
		return err
	}
	if err := dst.Close(); err != nil {
		_ = os.Remove(diskPath)
		return err
	}
	return nil
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	return sha256Hex(f)
}

func detectMimeByExt(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".pdf":
		return "application/pdf"
	case ".txt":
		return "text/plain"
	case ".md", ".markdown":
		return "text/markdown"
	case ".csv":
		return "text/csv"
	case ".doc":
		return "application/msword"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".xls":
		return "application/vnd.ms-excel"
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".ppt":
		return "application/vnd.ms-powerpoint"
	case ".pptx":
		return "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	case ".json":
		return "application/json"
	case ".zip":
		return "application/zip"
	case ".gz", ".gzip":
		return "application/gzip"
	case ".tar":
		return "application/x-tar"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}
