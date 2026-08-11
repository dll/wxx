package service

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
)

// PortalCredentialService 学校门户凭证服务
// 安全策略：密码以 AES-GCM 加密存储（WXX_ENCRYPTION_KEY），
// 查询接口绝不返回明文密码，仅返回绑定状态与账号。
type PortalCredentialService struct {
	repo *repository.PortalCredentialRepo
}

// NewPortalCredentialService 创建门户凭证服务
func NewPortalCredentialService(repo *repository.PortalCredentialRepo) *PortalCredentialService {
	return &PortalCredentialService{repo: repo}
}

// Get 查询绑定状态（不含密码明文）
func (s *PortalCredentialService) Get(userID int64) (*model.PortalCredentialView, error) {
	c, err := s.repo.Get(userID)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return &model.PortalCredentialView{UserID: userID}, nil
	}
	return &model.PortalCredentialView{
		UserID:        c.UserID,
		PortalURL:     c.PortalURL,
		PortalAccount: c.PortalAccount,
		Bound:         c.PortalAccount != "" && c.PortalPasswordEnc != "",
		UpdatedAt:     c.UpdatedAt,
	}, nil
}

// Save 保存凭证（密码加密存储）
func (s *PortalCredentialService) Save(userID int64, portalURL, account, password string) error {
	if strings.TrimSpace(account) == "" {
		return fmt.Errorf("门户账号不能为空")
	}
	if strings.TrimSpace(password) == "" {
		return fmt.Errorf("门户密码不能为空")
	}
	if strings.TrimSpace(portalURL) == "" {
		portalURL = "https://my0.chzu.edu.cn/"
	}
	// 校验 URL 合法性
	u, err := url.Parse(portalURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("门户地址不合法，须为 http(s) 链接")
	}
	return s.repo.Upsert(userID, portalURL, strings.TrimSpace(account), password)
}

// Delete 清除凭证
func (s *PortalCredentialService) Delete(userID int64) error {
	return s.repo.Delete(userID)
}
