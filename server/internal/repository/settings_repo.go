package repository

import (
	"database/sql"

	dbutil "github.com/dll/wxx/server/internal/db"
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
		"SELECT id, `key`, `value`, description, updated_by, created_at, updated_at " +
			"FROM system_settings ORDER BY id ASC",
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
	stmt := "INSERT INTO system_settings (`key`, `value`, updated_by, updated_at) " +
		"VALUES (?, ?, ?, CURRENT_TIMESTAMP) " +
		"ON CONFLICT(`key`) DO UPDATE SET `value`=excluded.`value`, updated_by=excluded.updated_by, updated_at=CURRENT_TIMESTAMP"
	_, err := r.db.Exec(dbutil.AdaptForDriver(stmt, dbutil.DriverOf(r.db)),
		key, value, updatedBy,
	)
	return err
}

// GetByPrefix 获取指定前缀的配置项（如 feature.）
func (r *SettingsRepo) GetByPrefix(prefix string) (map[string]string, error) {
	rows, err := r.db.Query("SELECT `key`, `value` FROM system_settings WHERE `key` LIKE ?", prefix+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := map[string]string{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err == nil {
			result[k] = v
		}
	}
	return result, rows.Err()
}
