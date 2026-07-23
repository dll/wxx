package service

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/dll/wxx/server/internal/config"
	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

// 短信验证码存储（开发环境）
var smsCodeStore sync.Map

// AuthService 认证业务服务
type AuthService struct {
	cfg      *config.Config
	userRepo *repository.UserRepo
}

// ErrUserNotFound 用户不存在 sentinel error，调用方可用 errors.Is 识别后单独处理
var ErrUserNotFound = errors.New("用户不存在")

// NewAuthService 创建认证服务
func NewAuthService(cfg *config.Config, userRepo *repository.UserRepo) *AuthService {
	return &AuthService{
		cfg:      cfg,
		userRepo: userRepo,
	}
}

// LoginResult 登录结果
type LoginResult struct {
	Token       string `json:"token"`
	ExpiresIn   int    `json:"expires_in"` // 过期时间（秒）
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
}

// SendCode 发送短信验证码（开发环境：仅日志，后续对接短信通道）
func (s *AuthService) SendCode(phone string) (string, error) {
	if phone == "" {
		return "", fmt.Errorf("手机号不能为空")
	}
	if len(phone) != 11 || phone[0] != '1' {
		return "", fmt.Errorf("手机号格式不正确")
	}
	code := fmt.Sprintf("%06d", rand.Intn(1000000))
	smsCodeStore.Store(phone, code)
	log.Printf("[DEV] 短信验证码 手机=%s code=%s", phone, code)
	return code, nil
}

// VerifyCode 校验短信验证码
func (s *AuthService) VerifyCode(phone, code string) bool {
	if code == "" || phone == "" {
		return false
	}
	// 开发环境：任意 6 位数字可通过（仅校验格式）
	if len(code) == 6 {
		return true
	}
	stored, ok := smsCodeStore.Load(phone)
	if !ok {
		return false
	}
	smsCodeStore.Delete(phone)
	return stored.(string) == code
}

// GuestRegister 游客注册
func (s *AuthService) GuestRegister(displayName, phone, code string) (*LoginResult, error) {
	if displayName == "" {
		return nil, fmt.Errorf("昵称不能为空")
	}
	if phone == "" {
		return nil, fmt.Errorf("手机号不能为空")
	}
	if !s.VerifyCode(phone, code) {
		return nil, fmt.Errorf("验证码错误或已过期")
	}

	// 生成唯一用户名
	username := fmt.Sprintf("guest_%s_%d", phone, time.Now().UnixMilli()%10000)

	user := &model.User{
		Username:    username,
		DisplayName: displayName,
		Role:        "guest",
		OwnerScope:  "college",
		OwnerID:     "default",
		Status:      "pending", // 游客默认为 pending，需审核
	}
	id, err := s.userRepo.Create(user)
	if err != nil {
		return nil, fmt.Errorf("创建游客失败: %w", err)
	}
	user.ID = id

	token, err := middleware.GenerateToken(s.cfg, user)
	if err != nil {
		return nil, fmt.Errorf("签发 token 失败: %w", err)
	}

	return &LoginResult{
		Token:       token,
		ExpiresIn:   s.cfg.JWTExpireHours * 3600,
		DisplayName: user.DisplayName,
		Role:        user.Role,
	}, nil
}

// ApproveGuest 审核通过游客->学生
func (s *AuthService) ApproveGuest(guestID int64, newUsername string) error {
	if newUsername == "" {
		return fmt.Errorf("学号不能为空")
	}

	user, err := s.userRepo.GetByID(guestID)
	if err != nil {
		return fmt.Errorf("查询用户失败: %w", err)
	}
	if user == nil {
		return ErrUserNotFound
	}
	if user.Role != "guest" {
		return fmt.Errorf("该用户不是游客")
	}
	if user.Status != "pending" {
		return fmt.Errorf("该游客状态不是待审核: %s", user.Status)
	}

	// 检查学号是否已被使用
	existing, err := s.userRepo.GetByUsername(newUsername)
	if err != nil {
		return fmt.Errorf("查询学号失败: %w", err)
	}
	if existing != nil {
		return fmt.Errorf("学号 %s 已被使用", newUsername)
	}

	// 更新用户名和角色
	user.Username = newUsername
	user.Role = "student"
	user.Status = "active"
	if err := s.userRepo.UpdateUsernameAndRole(user.ID, newUsername, "student"); err != nil {
		return fmt.Errorf("更新用户信息失败: %w", err)
	}
	if err := s.userRepo.UpdateStatus(user.ID, "active"); err != nil {
		return fmt.Errorf("更新状态失败: %w", err)
	}

	return nil
}

// RejectGuest 拒绝游客申请
func (s *AuthService) RejectGuest(guestID int64) error {
	user, err := s.userRepo.GetByID(guestID)
	if err != nil {
		return fmt.Errorf("查询用户失败: %w", err)
	}
	if user == nil {
		return ErrUserNotFound
	}
	if user.Role != "guest" {
		return fmt.Errorf("该用户不是游客")
	}

	return s.userRepo.UpdateStatus(user.ID, "rejected")
}

// ListPendingGuests 列出待审核游客
func (s *AuthService) ListPendingGuests() ([]*model.User, error) {
	return s.userRepo.ListPendingGuests()
}

// RecordConsent 记录用户同意隐私政策与用户协议
// 当前用户表无 consented 字段，仅记录日志；不因用户缺失而报错
func (s *AuthService) RecordConsent(userID int64) error {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		log.Printf("查询用户失败(consent): %v", err)
		return nil // 不阻塞前端流程
	}
	if user == nil {
		log.Printf("用户不存在(consent): id=%d，跳过记录", userID)
		return nil
	}
	log.Printf("用户同意隐私政策与用户协议: user=%s role=%s", user.Username, user.Role)
	return nil
}

// LoginByUsername 通过用户名登录（开发环境简化登录，生产环境走 SSO）
func (s *AuthService) LoginByUsername(username string, role string, password string) (*LoginResult, error) {
	if username == "" {
		return nil, fmt.Errorf("用户名不能为空")
	}

	// 查询用户
	user, err := s.userRepo.GetByUsername(username)
	if err != nil {
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}

	// 用户不存在 → 返回错误（预发布环境，必须先导入）
	if user == nil {
		return nil, fmt.Errorf("用户不存在，请联系管理员导入账号")
	}

	// 密码验证（预发布环境密码必填）
	if password == "" {
		return nil, fmt.Errorf("密码不能为空")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("密码错误")
	}

	// 签发 JWT
	token, err := middleware.GenerateToken(s.cfg, user)
	if err != nil {
		return nil, fmt.Errorf("签发 token 失败: %w", err)
	}

	return &LoginResult{
		Token:       token,
		ExpiresIn:   s.cfg.JWTExpireHours * 3600,
		DisplayName: user.DisplayName,
		Role:        user.Role,
	}, nil
}

// ChangePassword 用户自助修改密码
func (s *AuthService) ChangePassword(userID int64, oldPassword, newPassword string) error {
	if len(newPassword) < 6 {
		return fmt.Errorf("新密码长度不能少于 6 位")
	}

	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return fmt.Errorf("查询用户失败: %w", err)
	}
	if user == nil {
		return fmt.Errorf("用户不存在")
	}

	// 若已有旧密码，验证旧密码
	if user.PasswordHash != "" {
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword)); err != nil {
			if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
				return fmt.Errorf("旧密码错误")
			}
			return fmt.Errorf("密码验证失败: %w", err)
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("密码加密失败: %w", err)
	}

	if err := s.userRepo.UpdatePassword(userID, string(hash)); err != nil {
		return fmt.Errorf("更新密码失败: %w", err)
	}

	log.Printf("用户修改密码成功: userID=%d", userID)
	return nil
}

// ResetPassword 管理员重置用户密码（仅 sys_admin 可用，调用方已做角色校验）
func (s *AuthService) ResetPassword(adminID int64, targetUserID int64, newPassword string) error {
	if len(newPassword) < 6 {
		return fmt.Errorf("新密码长度不能少于 6 位")
	}

	targetUser, err := s.userRepo.GetByID(targetUserID)
	if err != nil {
		return fmt.Errorf("查询目标用户失败: %w", err)
	}
	if targetUser == nil {
		return fmt.Errorf("目标用户不存在")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("密码加密失败: %w", err)
	}

	if err := s.userRepo.UpdatePassword(targetUserID, string(hash)); err != nil {
		return fmt.Errorf("重置密码失败: %w", err)
	}

	log.Printf("管理员重置密码: adminID=%d targetUser=%s", adminID, targetUser.Username)
	return nil
}

// GetVoiceConfig 获取用户语音开关配置
// 用户不存在时返回 ErrUserNotFound（sentinel），由 handler 决定如何降级
func (s *AuthService) GetVoiceConfig(userID int64) (int, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return 0, fmt.Errorf("查询用户失败: %w", err)
	}
	if user == nil {
		return 0, ErrUserNotFound
	}
	return user.VoiceEnabled, nil
}

// UpdateVoiceConfig 更新用户语音开关
func (s *AuthService) UpdateVoiceConfig(userID int64, enabled int) error {
	if enabled != 0 && enabled != 1 {
		return fmt.Errorf("语音开关值需为 0 或 1")
	}
	if err := s.userRepo.UpdateVoiceEnabled(userID, enabled); err != nil {
		return fmt.Errorf("更新语音配置失败: %w", err)
	}
	log.Printf("用户语音配置已更新: userID=%d voice_enabled=%d", userID, enabled)
	return nil
}
