package service

import (
	"fmt"
	"log"

	"github.com/dll/wxx/server/internal/config"
	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
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
func (s *AuthService) LoginByUsername(username string) (*LoginResult, error) {
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
		user = &model.User{
			Username:    username,
			DisplayName: username,
			Role:        "student",
			OwnerScope:  "college",
			OwnerID:     "default",
		}
		id, err := s.userRepo.Create(user)
		if err != nil {
			return nil, fmt.Errorf("创建用户失败: %w", err)
		}
		user.ID = id
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
