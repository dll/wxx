package repository

import (
	"database/sql"
	"fmt"

	dbutil "github.com/dll/wxx/server/internal/db"
	"github.com/dll/wxx/server/internal/model"
)

// ExternalAppRepo 第三方应用清单数据访问
type ExternalAppRepo struct {
	db *sql.DB
}

// NewExternalAppRepo 创建第三方应用 repo
func NewExternalAppRepo(db *sql.DB) *ExternalAppRepo {
	return &ExternalAppRepo{db: db}
}

// ListEnabled 列出启用的应用（应用中心用户可见候选）
func (r *ExternalAppRepo) ListEnabled() ([]*model.ExternalApp, error) {
	rows, err := r.db.Query(`
		SELECT id, manifest, enabled, created_by, created_at, updated_at
		FROM external_apps
		WHERE enabled = 1
		ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanApps(rows)
}

// ListAll 列出全部应用（管理端）
func (r *ExternalAppRepo) ListAll() ([]*model.ExternalApp, error) {
	rows, err := r.db.Query(`
		SELECT id, manifest, enabled, created_by, created_at, updated_at
		FROM external_apps
		ORDER BY enabled DESC, created_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanApps(rows)
}

func scanApps(rows *sql.Rows) ([]*model.ExternalApp, error) {
	var list []*model.ExternalApp
	for rows.Next() {
		var a model.ExternalApp
		if err := rows.Scan(&a.ID, &a.Manifest, &a.Enabled, &a.CreatedBy, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, &a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

// Get 按 ID 查询应用
func (r *ExternalAppRepo) Get(id string) (*model.ExternalApp, error) {
	var a model.ExternalApp
	err := r.db.QueryRow(`
		SELECT id, manifest, enabled, created_by, created_at, updated_at
		FROM external_apps WHERE id = ?
	`, id).Scan(&a.ID, &a.Manifest, &a.Enabled, &a.CreatedBy, &a.CreatedAt, &a.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// Create 注册应用（manifest + created_by），返回是否新建成功（已存在则更新）
func (r *ExternalAppRepo) Create(id, manifest string, enabled int, createdBy int64, now int64) (bool, error) {
	stmt := `
		INSERT INTO external_apps (id, manifest, enabled, created_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			manifest = excluded.manifest,
			enabled = excluded.enabled,
			updated_at = excluded.updated_at`
	res, err := r.db.Exec(dbutil.AdaptForDriver(stmt, dbutil.DriverOf(r.db)),
		id, manifest, enabled, createdBy, now, now)
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

// Update 更新 manifest 与启停状态
func (r *ExternalAppRepo) Update(id, manifest string, enabled *int, now int64) error {
	query := `UPDATE external_apps SET updated_at = ?`
	args := []any{now}
	if manifest != "" {
		query += `, manifest = ?`
		args = append(args, manifest)
	}
	if enabled != nil {
		query += `, enabled = ?`
		args = append(args, *enabled)
	}
	query += ` WHERE id = ?`
	args = append(args, id)
	_, err := r.db.Exec(query, args...)
	return err
}

// Delete 删除应用（物理删除）
func (r *ExternalAppRepo) Delete(id string) error {
	_, err := r.db.Exec(`DELETE FROM external_apps WHERE id = ?`, id)
	return err
}

// EnsureCount 供测试使用的行数统计
func (r *ExternalAppRepo) Count() (int, error) {
	var n int
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM external_apps`).Scan(&n); err != nil {
		return 0, fmt.Errorf("统计外部应用数量失败: %w", err)
	}
	return n, nil
}
