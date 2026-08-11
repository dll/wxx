package model

// UserPortalCredential 学校门户登录凭证（密码加密存储，绝不明文回显）
type UserPortalCredential struct {
	ID                int64  `json:"id" db:"id"`
	UserID            int64  `json:"user_id" db:"user_id"`
	PortalURL         string `json:"portal_url" db:"portal_url"`
	PortalAccount     string `json:"portal_account" db:"portal_account"`
	PortalPasswordEnc string `json:"-" db:"portal_password_enc"` // AES-GCM 密文，永不输出
	UpdatedAt         string `json:"updated_at" db:"updated_at"`
	CreatedAt         string `json:"created_at" db:"created_at"`
}

// PortalCredentialView 返回给前端的视图（不含密码，仅含绑定状态与账号）
type PortalCredentialView struct {
	UserID        int64  `json:"user_id"`
	PortalURL     string `json:"portal_url"`
	PortalAccount string `json:"portal_account"`
	Bound         bool   `json:"bound"`
	UpdatedAt     string `json:"updated_at"`
}
