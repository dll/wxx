// Package model 用户站内通知（P4-d：从 handler 下沉为共享 DTO）。
package model

// UserNotification 站内通知
type UserNotification struct {
	ID          int64  `json:"id"`
	UserID      int64  `json:"user_id"`
	Title       string `json:"title"`
	Content     string `json:"content"`
	Type        string `json:"type"`
	RelatedType string `json:"related_type"`
	RelatedID   int64  `json:"related_id"`
	IsRead      int    `json:"is_read"`
	CreatedAt   string `json:"created_at"`
}
