package repository

import (
	"database/sql"

	dbutil "github.com/dll/wxx/server/internal/db"
	"github.com/dll/wxx/server/internal/model"
)

// PortalCredentialRepo 学校门户凭证数据访问（密码 AES-GCM 加密）
type PortalCredentialRepo struct {
	db *sql.DB
}

// NewPortalCredentialRepo 创建门户凭证 repo
func NewPortalCredentialRepo(db *sql.DB) *PortalCredentialRepo {
	return &PortalCredentialRepo{db: db}
}

// Get 查询某用户的门户凭证
func (r *PortalCredentialRepo) Get(userID int64) (*model.UserPortalCredential, error) {
	var c model.UserPortalCredential
	err := r.db.QueryRow(
		"SELECT id, user_id, portal_url, portal_account, portal_password_enc, updated_at, created_at "+
			"FROM user_portal_credentials WHERE user_id = ?", userID,
	).Scan(&c.ID, &c.UserID, &c.PortalURL, &c.PortalAccount, &c.PortalPasswordEnc,
		&c.UpdatedAt, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// Upsert 保存（密码加密后存储）
func (r *PortalCredentialRepo) Upsert(userID int64, portalURL, account, password string) error {
	enc, err := encrypt(password)
	if err != nil {
		return err
	}
	stmt := `
		INSERT INTO user_portal_credentials (user_id, portal_url, portal_account, portal_password_enc)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			portal_url = excluded.portal_url,
			portal_account = excluded.portal_account,
			portal_password_enc = excluded.portal_password_enc,
			updated_at = CURRENT_TIMESTAMP`
	_, err = r.db.Exec(dbutil.AdaptForDriver(stmt, dbutil.DriverOf(r.db)),
		userID, portalURL, account, enc)
	return err
}

// Delete 删除凭证
func (r *PortalCredentialRepo) Delete(userID int64) error {
	_, err := r.db.Exec("DELETE FROM user_portal_credentials WHERE user_id = ?", userID)
	return err
}

// DecryptPortalPassword 解密门户密码（供代理服务登录会话使用；仅内存态，不落日志）
func DecryptPortalPassword(cipherHex string) (string, error) {
	return decrypt(cipherHex)
}
