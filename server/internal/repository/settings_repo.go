package repository

import (
	"database/sql"

	"github.com/dll/wxx/server/internal/model"
)

// SettingsRepo 系统配置数据访问
type SettingsRepo struct {
	db *sql.DB
}

// NewSettingsRepo 创建系统配置 repo
func NewSettingsRepo(db *sql.DB) *SettingsRepo {
	return &SettingsRepo{db: db}
}

// GetAll 获取所有配置项
func (r *SettingsRepo) GetAll() ([]*model.SystemSetting, error) {
	rows, err := r.db.Query(
		`SELECT id, key, value, description, updated_by, created_at, updated_at
		 FROM system_settings ORDER BY id ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var settings []*model.SystemSetting
	for rows.Next() {
		s := &model.SystemSetting{}
		if err := rows.Scan(&s.ID, &s.Key, &s.Value, &s.Description,
			&s.UpdatedBy, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		settings = append(settings, s)
	}
	return settings, rows.Err()
}

// Upsert 插入或更新配置项
func (r *SettingsRepo) Upsert(key, value, updatedBy string) error {
	_, err := r.db.Exec(
		`INSERT INTO system_settings (key, value, updated_by, updated_at)
		 VALUES (?, ?, ?, datetime('now'))
		 ON CONFLICT(key) DO UPDATE SET value=excluded.value, updated_by=excluded.updated_by, updated_at=datetime('now')`,
		key, value, updatedBy,
	)
	return err
}
