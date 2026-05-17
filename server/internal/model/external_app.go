package model

// ExternalApp 第三方应用
type ExternalApp struct {
	ID          int64  `json:"id" db:"id"`
	AppKey      string `json:"app_key" db:"app_key"`
	Name        string `json:"name" db:"name"`
	Description string `json:"description" db:"description"`
	IconURL     string `json:"icon_url" db:"icon_url"`
	AppURL      string `json:"app_url" db:"app_url"`
	Mode        string `json:"mode" db:"mode"` // external_link/webview/reverse_proxy
	Category    string `json:"category" db:"category"`
	Status      string `json:"status" db:"status"` // active/inactive
	CreatedBy   string `json:"created_by" db:"created_by"`
	CreatedAt   string `json:"created_at" db:"created_at"`
	UpdatedAt   string `json:"updated_at" db:"updated_at"`
}
