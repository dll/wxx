package handler

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/gin-gonic/gin"
)

// ── QR 登录会话管理（内存存储，Vercel 热实例间共享）──

type qrSession struct {
	SessionID string    `json:"session_id"`
	Status    string    `json:"status"` // pending | scanned | confirmed
	Token     string    `json:"token,omitempty"`
	ExpiresAt time.Time `json:"expires_at"`
	// PollSecret 仅发放给创建会话的桌面浏览器，绝不写入二维码。
	// 安全修复 S-03：拉取令牌必须持有该密钥，防止扫码者凭 sessionID 轮询窃取令牌。
	PollSecret string `json:"-"`
}

type qrSessionStore struct {
	mu          sync.RWMutex
	data        map[string]*qrSession
	cleanupOnce sync.Once
}

var qrStore = &qrSessionStore{data: make(map[string]*qrSession)}

// generateSessionID 用 crypto/rand 生成强随机 session ID
func generateSessionID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "qr_" + hex.EncodeToString(b), nil
}

// generatePollSecret 生成拉取令牌用的强随机轮询密钥（不进二维码）
func generatePollSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// constantTimeEqual 定长比较，避免时序侧信道
func constantTimeEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// CreateQRSession 创建 QR 登录会话 POST /api/v1/auth/qr-login
// 安全：sessionID 必须由服务端生成，忽略客户端传入的值
func CreateQRSession(c *gin.Context) {
	sessionID, err := generateSessionID()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code: 500, Message: "生成会话 ID 失败",
		})
		return
	}

	pollSecret, err := generatePollSecret()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code: 500, Message: "生成轮询密钥失败",
		})
		return
	}

	qrStore.mu.Lock()
	qrStore.data[sessionID] = &qrSession{
		SessionID:  sessionID,
		Status:     "pending",
		ExpiresAt:  time.Now().Add(5 * time.Minute),
		PollSecret: pollSecret,
	}
	qrStore.mu.Unlock()

	// 启动单例后台清理 goroutine（仅首次调用启动）
	qrStore.cleanupOnce.Do(qrStore.startCleanup)

	// 安全修复 S-03：poll_secret 只返回给创建会话的桌面浏览器，
	// 二维码内只含 session_id；拉取令牌时必须回传 poll_secret。
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "QR 会话已创建",
		"data": gin.H{
			"session_id":  sessionID,
			"poll_secret": pollSecret,
			"expires_in":  300,
		},
	})
}

// GetQRSessionStatus 查询 QR 会话状态 GET /api/v1/auth/qr-status
// 安全修复 S-03：仅在提供正确 poll_secret 时才下发 token，防止扫码者凭 session_id 窃取令牌。
func GetQRSessionStatus(c *gin.Context) {
	sessionID := c.Query("session")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    400,
			Message: "缺少 session 参数",
		})
		return
	}
	pollSecret := c.Query("poll_secret")

	qrStore.mu.RLock()
	s, ok := qrStore.data[sessionID]
	qrStore.mu.RUnlock()

	if !ok || time.Now().After(s.ExpiresAt) {
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "success",
			"data":    gin.H{"status": "expired"},
		})
		return
	}

	// token 仅在 poll_secret 校验通过时下发；否则只暴露状态，绝不泄露令牌。
	token := ""
	if s.Status == "confirmed" && pollSecret != "" && constantTimeEqual(pollSecret, s.PollSecret) {
		token = s.Token
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    gin.H{"status": s.Status, "token": token},
	})
}

// ConfirmQRSession 确认 QR 登录（手机端扫码后调用） POST /api/v1/auth/qr-confirm
func ConfirmQRSession(c *gin.Context) {
	var req struct {
		SessionID string `json:"session_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code: 400, Message: "参数校验失败: " + err.Error(),
		})
		return
	}

	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "请先在手机上登录"})
		return
	}

	// 提取并校验 Bearer Token
	authHeader := c.GetHeader("Authorization")
	token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	if token == "" || token == authHeader {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "无效的认证 Token"})
		return
	}

	qrStore.mu.Lock()
	s, ok := qrStore.data[req.SessionID]
	if !ok {
		qrStore.mu.Unlock()
		c.JSON(http.StatusNotFound, model.ErrorResponse{Code: 404, Message: "会话不存在或已过期"})
		return
	}
	if time.Now().After(s.ExpiresAt) {
		qrStore.mu.Unlock()
		c.JSON(http.StatusGone, model.ErrorResponse{Code: 410, Message: "会话已过期"})
		return
	}
	// 幂等防护：已确认的会话不允许覆盖
	if s.Status == "confirmed" {
		qrStore.mu.Unlock()
		c.JSON(http.StatusConflict, model.ErrorResponse{Code: 409, Message: "会话已确认，请勿重复操作"})
		return
	}
	s.Status = "confirmed"
	s.Token = token
	qrStore.mu.Unlock()

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "登录已确认"})
}

// ScanQRSession 标记 QR 已扫描 PUT /api/v1/auth/qr-scan
func ScanQRSession(c *gin.Context) {
	var req struct {
		SessionID string `json:"session_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code: 400, Message: "参数校验失败",
		})
		return
	}

	qrStore.mu.Lock()
	s, ok := qrStore.data[req.SessionID]
	if ok && s.Status == "pending" {
		s.Status = "scanned"
	}
	qrStore.mu.Unlock()

	if !ok {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Code: 404, Message: "会话不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已标记为已扫描"})
}

// startCleanup 启动后台周期清理（每分钟一次），由 sync.Once 保证仅启动一次
func (s *qrSessionStore) startCleanup() {
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			s.cleanup()
		}
	}()
}

// cleanup 删除过期会话：先用读锁找出待删 ID，再用写锁短暂持有删除
// 避免在大量过期会话场景下长时间持写锁阻塞 Get/Confirm
func (s *qrSessionStore) cleanup() {
	now := time.Now()

	s.mu.RLock()
	expired := make([]string, 0, len(s.data)/4+1)
	for id, session := range s.data {
		if now.After(session.ExpiresAt) {
			expired = append(expired, id)
		}
	}
	s.mu.RUnlock()

	if len(expired) == 0 {
		return
	}

	s.mu.Lock()
	for _, id := range expired {
		// 二次确认（其他 goroutine 可能已经 confirm 了）
		if session, ok := s.data[id]; ok && now.After(session.ExpiresAt) {
			delete(s.data, id)
		}
	}
	s.mu.Unlock()
}
