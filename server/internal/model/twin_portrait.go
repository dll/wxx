package model

// TwinPortrait 数字孪生画像
type TwinPortrait struct {
	ID                int64  `json:"id" db:"id"`
	UserID            int64  `json:"user_id" db:"user_id"`
	PrototypeType     string `json:"prototype_type" db:"prototype_type"` // photo | chao_xing
	PromptVersion     string `json:"prompt_version" db:"prompt_version"`
	ImageBase64       string `json:"image_base64" db:"image_base64"`
	ImageMIME         string `json:"image_mime" db:"image_mime"`
	SourcePhotoBase64 string `json:"source_photo_base64" db:"source_photo_base64"`
	CreatedAt         string `json:"created_at" db:"created_at"`
}

// TwinPortraitView 返回给前端的画像视图（不含原型照片，避免超大响应）
type TwinPortraitView struct {
	ID            int64  `json:"id"`
	UserID        int64  `json:"user_id"`
	PrototypeType string `json:"prototype_type"`
	PromptVersion string `json:"prompt_version"`
	ImageBase64   string `json:"image_base64"`
	ImageMIME     string `json:"image_mime"`
	HasPhoto      bool   `json:"has_photo"`
	CreatedAt     string `json:"created_at"`
}
