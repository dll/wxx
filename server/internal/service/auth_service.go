package service

import (
	"errors"
	"fmt"
	"log"

	"github.com/dll/wxx/server/internal/config"
	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

// AuthService 认证业务服务
type AuthService struct {
	cfg      *config.Config
	userRepo *repository.UserRepo
}

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

	// 用户不存在时自动创建（开发环境）
	if user == nil {
		if role == "" {
			role = "student"
		}
		user = &model.User{
			Username:    username,
			DisplayName: username,
			Role:        role,
			OwnerScope:  "college",
			OwnerID:     "default",
		}
		id, err := s.userRepo.Create(user)
		if err != nil {
			return nil, fmt.Errorf("创建用户失败: %w", err)
		}
		user.ID = id
	}

	// 若用户已设置密码，验证密码
	if user.PasswordHash != "" {
		if password == "" {
			return nil, fmt.Errorf("密码不能为空")
		}
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
			if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
				return nil, fmt.Errorf("密码错误")
			}
			return nil, fmt.Errorf("密码验证失败: %w", err)
		}
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
func (s *AuthService) GetVoiceConfig(userID int64) (int, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return 0, fmt.Errorf("查询用户失败: %w", err)
	}
	if user == nil {
		return 0, fmt.Errorf("用户不存在")
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
