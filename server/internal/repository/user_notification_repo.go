// Package repository 用户站内通知仓库（P4-d：从 user_notification_handler 下沉的 12 处裸 SQL）。
package repository

import (
	"database/sql"
	"log"

	"github.com/dll/wxx/server/internal/model"
)

// UserNotificationRepo 站内通知数据访问层。
type UserNotificationRepo struct {
	db *sql.DB
}

// NewUserNotificationRepo 创建通知仓库。
func NewUserNotificationRepo(db *sql.DB) *UserNotificationRepo {
	return &UserNotificationRepo{db: db}
}

// CountUnread 统计用户未读数。
func (r *UserNotificationRepo) CountUnread(userID int64) (int, error) {
	var unread int
	err := r.db.QueryRow("SELECT COUNT(*) FROM user_notifications WHERE user_id = ? AND is_read = 0", userID).Scan(&unread)
	return unread, err
}

// ListByUser 分页查询用户通知（含总数与未读数）。
func (r *UserNotificationRepo) ListByUser(userID int64, notifType string, page, pageSize int) ([]model.UserNotification, int, int, error) {
	where := []string{"user_id = ?"}
	args := []interface{}{userID}
	if notifType != "" {
		where = append(where, "type = ?")
		args = append(args, notifType)
	}
	whereSQL := buildWhereClause(where)

	unread, err := r.CountUnread(userID)
	if err != nil {
		return nil, 0, 0, err
	}

	var total int
	if err := r.db.QueryRow("SELECT COUNT(*) FROM user_notifications "+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, 0, err
	}

	rows, err := r.db.Query(
		"SELECT id, user_id, title, content, type, related_type, related_id, is_read, created_at "+
			"FROM user_notifications "+whereSQL+" ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?",
		append(args, pageSize, pageOffset(page, pageSize))...,
	)
	if err != nil {
		return nil, 0, 0, err
	}
	defer rows.Close()

	items := make([]model.UserNotification, 0)
	for rows.Next() {
		var n model.UserNotification
		if err := rows.Scan(&n.ID, &n.UserID, &n.Title, &n.Content, &n.Type,
			&n.RelatedType, &n.RelatedID, &n.IsRead, &n.CreatedAt); err != nil {
			log.Printf("[notification] 扫描通知行失败: %v", err)
			continue
		}
		items = append(items, n)
	}
	return items, total, unread, rows.Err()
}

// MarkRead 标记单条已读（返回受影响行数，0=不存在或不属于该用户）。
func (r *UserNotificationRepo) MarkRead(id, userID int64) (int64, error) {
	result, err := r.db.Exec("UPDATE user_notifications SET is_read = 1 WHERE id = ? AND user_id = ?", id, userID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// MarkAllRead 全部已读。
func (r *UserNotificationRepo) MarkAllRead(userID int64) error {
	_, err := r.db.Exec("UPDATE user_notifications SET is_read = 1 WHERE user_id = ? AND is_read = 0", userID)
	return err
}

// ActiveUserIDs 全部启用用户 ID（群发用）。
func (r *UserNotificationRepo) ActiveUserIDs() ([]int64, error) {
	rows, err := r.db.Query("SELECT id FROM users WHERE status = 'active'")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	return ids, rows.Err()
}

// SendBulk 批量插入系统通知，返回成功条数（单条失败记日志不中断）。
func (r *UserNotificationRepo) SendBulk(title, content string, userIDs []int64) int {
	sent := 0
	for _, uid := range userIDs {
		if _, err := r.db.Exec(
			"INSERT INTO user_notifications (user_id, title, content, type, related_type, related_id, is_read) VALUES (?, ?, ?, 'system', '', 0, 0)",
			uid, title, content,
		); err == nil {
			sent++
		} else {
			log.Printf("[WARN] 系统通知写入失败 user_id=%d: %v", uid, err)
		}
	}
	return sent
}

// AdminList 管理端分页列表（含同条件总数）。
func (r *UserNotificationRepo) AdminList(notifType string, page, pageSize int) ([]model.UserNotification, int, error) {
	where := []string{}
	args := []interface{}{}
	if notifType != "" {
		where = append(where, "type = ?")
		args = append(args, notifType)
	}
	whereSQL := buildWhereClause(where)

	var total int
	if err := r.db.QueryRow("SELECT COUNT(*) FROM user_notifications "+whereSQL, args...).Scan(&total); err != nil {
		log.Printf("[WARN] 通知总数统计失败: %v", err)
	}

	rows, err := r.db.Query(
		"SELECT id, user_id, title, content, type, related_type, related_id, is_read, created_at "+
			"FROM user_notifications "+whereSQL+" ORDER BY id DESC LIMIT ? OFFSET ?",
		append(args, pageSize, pageOffset(page, pageSize))...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	list := make([]model.UserNotification, 0)
	for rows.Next() {
		var n model.UserNotification
		if err := rows.Scan(&n.ID, &n.UserID, &n.Title, &n.Content, &n.Type, &n.RelatedType, &n.RelatedID, &n.IsRead, &n.CreatedAt); err == nil {
			list = append(list, n)
		}
	}
	return list, total, rows.Err()
}

// AdminDelete 管理端删除单条（返回受影响行数）。
func (r *UserNotificationRepo) AdminDelete(id int64) (int64, error) {
	res, err := r.db.Exec("DELETE FROM user_notifications WHERE id = ?", id)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// AdminClear 管理端清空全部（返回受影响行数）。
func (r *UserNotificationRepo) AdminClear() (int64, error) {
	res, err := r.db.Exec("DELETE FROM user_notifications")
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
