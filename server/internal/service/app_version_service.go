package service

import (
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
)

type AppVersionService struct {
	repo *repository.AppVersionRepo
}

func NewAppVersionService(repo *repository.AppVersionRepo) *AppVersionService {
	return &AppVersionService{repo: repo}
}

// CheckUpdate 检查更新
func (s *AppVersionService) CheckUpdate(platform string, versionCode int) (*model.AppVersion, bool, error) {
	return s.repo.CheckUpdate(platform, versionCode)
}

// GetLatestVersion 获取最新版本
func (s *AppVersionService) GetLatestVersion(platform string) (*model.AppVersion, error) {
	return s.repo.GetLatestVersion(platform)
}

// ListVersions 版本列表
func (s *AppVersionService) ListVersions(page, pageSize int) ([]*model.AppVersion, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.ListAllVersions(page, pageSize)
}

// CreateVersion 创建新版本
func (s *AppVersionService) CreateVersion(v *model.AppVersion) error {
	return s.repo.CreateVersion(v)
}

// UpdateVersion 更新版本
func (s *AppVersionService) UpdateVersion(v *model.AppVersion) error {
	return s.repo.UpdateVersion(v)
}

// DeleteVersion 删除版本
func (s *AppVersionService) DeleteVersion(id int64) error {
	return s.repo.DeleteVersion(id)
}
