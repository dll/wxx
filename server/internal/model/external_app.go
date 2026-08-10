package model

// ══════════════════════════════════════════════════════════════
// 第三方应用接入（external_apps）
// ══════════════════════════════════════════════════════════════

// ExternalApp 第三方应用清单（manifest 整体 JSON 存储，运行时解析）
type ExternalApp struct {
	ID        string `json:"id" db:"id"`
	Manifest  string `json:"manifest" db:"manifest"` // 完整 JSON（见 docs/external-apps.md）
	Enabled   int    `json:"enabled" db:"enabled"`   // 1 启用 0 停用
	CreatedBy int64  `json:"created_by" db:"created_by"`
	CreatedAt int64  `json:"created_at" db:"created_at"`
	UpdatedAt int64  `json:"updated_at" db:"updated_at"`
}