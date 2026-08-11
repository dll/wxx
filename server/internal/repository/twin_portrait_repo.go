package repository

import (
	"database/sql"

	"github.com/dll/wxx/server/internal/model"
)

// TwinPortraitRepo 数字孪生画像数据访问
type TwinPortraitRepo struct {
	db *sql.DB
}

// NewTwinPortraitRepo 创建画像 repo
func NewTwinPortraitRepo(db *sql.DB) *TwinPortraitRepo {
	return &TwinPortraitRepo{db: db}
}

const twinPortraitCols = `id, user_id, prototype_type, prompt_version, image_base64, image_mime, source_photo_base64, created_at`

// GetByUserAndType 按用户与原型类型查询画像
func (r *TwinPortraitRepo) GetByUserAndType(userID int64, prototypeType string) (*model.TwinPortrait, error) {
	var p model.TwinPortrait
	var src sql.NullString
	err := r.db.QueryRow(
		"SELECT "+twinPortraitCols+" FROM twin_portraits WHERE user_id = ? AND prototype_type = ?",
		userID, prototypeType,
	).Scan(&p.ID, &p.UserID, &p.PrototypeType, &p.PromptVersion, &p.ImageBase64,
		&p.ImageMIME, &src, &p.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	p.SourcePhotoBase64 = src.String
	return &p, nil
}

// ListByUser 列出用户全部画像
func (r *TwinPortraitRepo) ListByUser(userID int64) ([]*model.TwinPortrait, error) {
	rows, err := r.db.Query(
		"SELECT "+twinPortraitCols+" FROM twin_portraits WHERE user_id = ? ORDER BY id ASC",
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*model.TwinPortrait
	for rows.Next() {
		p := &model.TwinPortrait{}
		var src sql.NullString
		if err := rows.Scan(&p.ID, &p.UserID, &p.PrototypeType, &p.PromptVersion, &p.ImageBase64,
			&p.ImageMIME, &src, &p.CreatedAt); err != nil {
			return nil, err
		}
		p.SourcePhotoBase64 = src.String
		list = append(list, p)
	}
	return list, rows.Err()
}

// Upsert 插入或覆盖画像（同用户+同类型幂等）
func (r *TwinPortraitRepo) Upsert(p *model.TwinPortrait) (int64, error) {
	res, err := r.db.Exec(`
		INSERT INTO twin_portraits (user_id, prototype_type, prompt_version, image_base64, image_mime, source_photo_base64)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, prototype_type) DO UPDATE SET
			prompt_version = excluded.prompt_version,
			image_base64 = excluded.image_base64,
			image_mime = excluded.image_mime,
			source_photo_base64 = excluded.source_photo_base64,
			created_at = datetime('now')`,
		p.UserID, p.PrototypeType, p.PromptVersion, p.ImageBase64, p.ImageMIME, p.SourcePhotoBase64)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}
