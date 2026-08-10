package service

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/dll/wxx/server/internal/auth"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
)

// ExternalAppService 第三方应用中心业务服务
type ExternalAppService struct {
	repo *repository.ExternalAppRepo
}

// NewExternalAppService 创建外部应用服务
func NewExternalAppService(repo *repository.ExternalAppRepo) *ExternalAppService {
	return &ExternalAppService{repo: repo}
}

// parseManifest 解析 manifest JSON；失败返回 nil
func parseManifest(raw string) *model.ExternalAppManifest {
	var m model.ExternalAppManifest
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return nil
	}
	return &m
}

// ListForUser 返回当前用户可见的启用应用（按 visible_to.roles + capabilities 过滤）
func (s *ExternalAppService) ListForUser(role string) ([]model.ExternalAppView, error) {
	apps, err := s.repo.ListEnabled()
	if err != nil {
		return nil, fmt.Errorf("查询应用列表失败: %w", err)
	}

	var views []model.ExternalAppView
	for _, a := range apps {
		m := parseManifest(a.Manifest)
		if m == nil {
			continue // 跳过损坏 manifest
		}
		if !s.visibleTo(a, m, role) {
			continue
		}
		views = append(views, model.ExternalAppView{
			ID:       m.ID,
			Name:     m.Name,
			Icon:     m.Icon,
			Category: m.Category,
			Summary:  m.Summary,
			Version:  m.Version,
			Type:     m.Adapter.Type,
			URL:      m.Adapter.URL,
			OpenIn:   m.Adapter.OpenIn,
		})
	}
	return views, nil
}

// visibleTo 判定应用对当前角色可见
func (s *ExternalAppService) visibleTo(a *model.ExternalApp, m *model.ExternalAppManifest, role string) bool {
	// 明确角色白名单：命中（含角色继承）才可见
	if roles := m.Visible.Roles; len(roles) > 0 {
		ok := false
		for _, r := range roles {
			if auth.RoleMatches(role, r) {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	// 能力 AND 关系
	for _, capName := range m.Visible.Capabilities {
		if !auth.HasCapability(role, auth.Capability(capName)) {
			return false
		}
	}
	return true
}

// ListAdmin 返回管理端全部应用（含启停）
func (s *ExternalAppService) ListAdmin() ([]model.ExternalAppAdminView, error) {
	apps, err := s.repo.ListAll()
	if err != nil {
		return nil, fmt.Errorf("查询应用列表失败: %w", err)
	}
	views := make([]model.ExternalAppAdminView, 0, len(apps))
	for _, a := range apps {
		m := parseManifest(a.Manifest)
		view := model.ExternalAppAdminView{Enabled: a.Enabled, Manifest: a.Manifest}
		if m != nil {
			view.ExternalAppView = model.ExternalAppView{
				ID:       m.ID,
				Name:     m.Name,
				Icon:     m.Icon,
				Category: m.Category,
				Summary:  m.Summary,
				Version:  m.Version,
				Type:     m.Adapter.Type,
				URL:      m.Adapter.URL,
				OpenIn:   m.Adapter.OpenIn,
			}
		} else {
			view.ID = a.ID
		}
		views = append(views, view)
	}
	return views, nil
}

// Create 注册应用（幂等：已存在则更新）
func (s *ExternalAppService) Create(manifestJSON string, enabled *int, createdByID int64) (*model.ExternalAppAdminView, error) {
	m := parseManifest(manifestJSON)
	if m == nil || m.ID == "" {
		return nil, fmt.Errorf("manifest 无效：缺少 id 或 JSON 解析失败")
	}
	if m.Adapter.Type == "" {
		m.Adapter.Type = "external_link"
	}
	if m.Adapter.OpenIn == "" {
		m.Adapter.OpenIn = "_blank"
	}
	if m.Category == "" {
		m.Category = "external"
	}
	norm, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("manifest 规范化失败: %w", err)
	}
	on := 1
	if enabled != nil {
		on = *enabled
		if on != 0 {
			on = 1
		}
	}
	now := time.Now().Unix()
	if _, err := s.repo.Create(m.ID, string(norm), on, createdByID, now); err != nil {
		return nil, fmt.Errorf("注册应用失败: %w", err)
	}
	return s.getAdminView(m.ID)
}

// Update 更新应用（可按需更新 manifest 与启停）
func (s *ExternalAppService) Update(id, manifestJSON string, enabled *int) (*model.ExternalAppAdminView, error) {
	existing, err := s.repo.Get(id)
	if err != nil {
		return nil, fmt.Errorf("查询应用失败: %w", err)
	}
	if existing == nil {
		return nil, fmt.Errorf("应用不存在: %s", id)
	}
	normalized := existing.Manifest
	if manifestJSON != "" {
		m := parseManifest(manifestJSON)
		if m == nil {
			return nil, fmt.Errorf("manifest 无效")
		}
		if m.ID != "" && m.ID != id {
			return nil, fmt.Errorf("manifest.id 与路径 id 不一致")
		}
		// 保持现有 ID
		m.ID = id
		norm, merr := json.Marshal(m)
		if merr != nil {
			return nil, fmt.Errorf("manifest 规范化失败: %w", merr)
		}
		normalized = string(norm)
	}
	if err := s.repo.Update(id, string(normalized), enabled, time.Now().Unix()); err != nil {
		return nil, fmt.Errorf("更新应用失败: %w", err)
	}
	return s.getAdminView(id)
}

// Delete 删除应用
func (s *ExternalAppService) Delete(id string) error {
	if err := s.repo.Delete(id); err != nil {
		return fmt.Errorf("删除应用失败: %w", err)
	}
	return nil
}

// getAdminView 查询并转为管理端视图
func (s *ExternalAppService) getAdminView(id string) (*model.ExternalAppAdminView, error) {
	a, err := s.repo.Get(id)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, fmt.Errorf("应用不存在: %s", id)
	}
	m := parseManifest(a.Manifest)
	view := model.ExternalAppAdminView{Enabled: a.Enabled, Manifest: a.Manifest}
	if m != nil {
		view.ExternalAppView = model.ExternalAppView{
			ID:       m.ID,
			Name:     m.Name,
			Icon:     m.Icon,
			Category: m.Category,
			Summary:  m.Summary,
			Version:  m.Version,
			Type:     m.Adapter.Type,
			URL:      m.Adapter.URL,
			OpenIn:   m.Adapter.OpenIn,
		}
	}
	return &view, nil
}