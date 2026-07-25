package repository

import (
	"database/sql"

	"github.com/dll/wxx/server/internal/model"
)

type AppVersionRepo struct {
	db *sql.DB
}

func NewAppVersionRepo(db *sql.DB) *AppVersionRepo {
	return &AppVersionRepo{db: db}
}

// GetLatestVersion 获取指定平台的最新版本
func (r *AppVersionRepo) GetLatestVersion(platform string) (*model.AppVersion, error) {
	var v model.AppVersion
	err := r.db.QueryRow(`
		SELECT id, version_code, version_name, platform, title, changelog,
		       download_url, force_update, status, created_at, updated_at
		FROM app_versions
		WHERE status = 1 AND (platform = ? OR platform = 'all')
		ORDER BY version_code DESC
		LIMIT 1
	`, platform).Scan(
		&v.ID, &v.VersionCode, &v.VersionName, &v.Platform, &v.Title,
		&v.Changelog, &v.DownloadURL, &v.ForceUpdate, &v.Status,
		&v.CreatedAt, &v.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// CheckUpdate 检查是否有新版本
func (r *AppVersionRepo) CheckUpdate(platform string, currentVersionCode int) (*model.AppVersion, bool, error) {
	latest, err := r.GetLatestVersion(platform)
	if err != nil {
		return nil, false, err
	}
	if latest == nil {
		return nil, false, nil
	}
	hasUpdate := latest.VersionCode > currentVersionCode
	return latest, hasUpdate, nil
}

// ListAllVersions 列出所有版本（管理用）
func (r *AppVersionRepo) ListAllVersions(page, pageSize int) ([]*model.AppVersion, int, error) {
	var total int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM app_versions`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	rows, err := r.db.Query(`
		SELECT id, version_code, version_name, platform, title, changelog,
		       download_url, force_update, status, created_at, updated_at
		FROM app_versions
		ORDER BY version_code DESC
		LIMIT ? OFFSET ?
	`, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []*model.AppVersion
	for rows.Next() {
		var v model.AppVersion
		err := rows.Scan(
			&v.ID, &v.VersionCode, &v.VersionName, &v.Platform, &v.Title,
			&v.Changelog, &v.DownloadURL, &v.ForceUpdate, &v.Status,
			&v.CreatedAt, &v.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		list = append(list, &v)
	}
	return list, total, nil
}

// CreateVersion 创建新版本
func (r *AppVersionRepo) CreateVersion(v *model.AppVersion) error {
	result, err := r.db.Exec(`
		INSERT INTO app_versions (version_code, version_name, platform, title, changelog, download_url, force_update, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, v.VersionCode, v.VersionName, v.Platform, v.Title, v.Changelog, v.DownloadURL, v.ForceUpdate, v.Status)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	v.ID = id
	return nil
}

// UpdateVersion 更新版本信息
func (r *AppVersionRepo) UpdateVersion(v *model.AppVersion) error {
	_, err := r.db.Exec(`
		UPDATE app_versions
		SET version_code=?, version_name=?, platform=?, title=?, changelog=?,
		    download_url=?, force_update=?, status=?, updated_at=CURRENT_TIMESTAMP
		WHERE id=?
	`, v.VersionCode, v.VersionName, v.Platform, v.Title, v.Changelog,
		v.DownloadURL, v.ForceUpdate, v.Status, v.ID)
	return err
}

// DeleteVersion 删除版本
func (r *AppVersionRepo) DeleteVersion(id int64) error {
	_, err := r.db.Exec(`DELETE FROM app_versions WHERE id=?`, id)
	return err
}
