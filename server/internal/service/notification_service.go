package service

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/dll/wxx/server/internal/model"
)

type Notification struct {
	ID           int64  `json:"id"`
	Title        string `json:"title"`
	Content      string `json:"content"`
	AuthorName   string `json:"author_name"`
	AudienceType string `json:"audience_type"`
	Status       string `json:"status"`
	PushQQ       bool   `json:"push_qq"`
	PushWechat   bool   `json:"push_wechat"`
	PushResult   string `json:"push_result,omitempty"`
	CreatedAt    string `json:"created_at"`
	PublishedAt  string `json:"published_at,omitempty"`
}

type WebhookConfig struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Channel    string `json:"channel"`
	WebhookURL string `json:"webhook_url"`
	IsActive   bool   `json:"is_active"`
}

type NotificationService struct {
	db            *sql.DB
	qqWebhook     string
	wechatWebhook string
	httpClient    *http.Client
}

func NewNotificationService(db *sql.DB, qqWebhook, wechatWebhook string) *NotificationService {
	return &NotificationService{
		db:            db,
		qqWebhook:     qqWebhook,
		wechatWebhook: wechatWebhook,
		httpClient:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *NotificationService) Create(ctx context.Context, user *model.UserContext, title, content, audienceType string, pushQQ, pushWechat bool) (*Notification, error) {
	if audienceType == "" {
		audienceType = "all"
	}
	result, err := s.db.ExecContext(ctx,
		`INSERT INTO notifications (title, content, author_id, author_name, audience_type, push_qq, push_wechat, status)
		 VALUES (?, ?, ?, ?, ?, ?, ?, 'draft')`,
		title, content, user.UserID, user.DisplayName, audienceType, boolToInt(pushQQ), boolToInt(pushWechat))
	if err != nil {
		return nil, fmt.Errorf("创建通知失败: %w", err)
	}
	id, _ := result.LastInsertId()
	return &Notification{
		ID: id, Title: title, Content: content, AuthorName: user.DisplayName,
		AudienceType: audienceType, Status: "draft",
		PushQQ: pushQQ, PushWechat: pushWechat,
		CreatedAt: time.Now().Format("2006-01-02 15:04:05"),
	}, nil
}

func (s *NotificationService) List(ctx context.Context, user *model.UserContext, page, limit int) ([]Notification, int, error) {
	var total int
	s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM notifications").Scan(&total)

	offset := (page - 1) * limit
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, title, content, author_name, audience_type, status, push_qq, push_wechat,
		        COALESCE(push_result,''), created_at, COALESCE(published_at,'')
		 FROM notifications ORDER BY created_at DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("查询通知列表失败: %w", err)
	}
	defer rows.Close()

	var items []Notification
	for rows.Next() {
		var n Notification
		var pushQQInt, pushWechatInt int
		if err := rows.Scan(&n.ID, &n.Title, &n.Content, &n.AuthorName, &n.AudienceType,
			&n.Status, &pushQQInt, &pushWechatInt, &n.PushResult, &n.CreatedAt, &n.PublishedAt); err != nil {
			continue
		}
		n.PushQQ = pushQQInt == 1
		n.PushWechat = pushWechatInt == 1
		items = append(items, n)
	}
	return items, total, nil
}

func (s *NotificationService) Publish(ctx context.Context, user *model.UserContext, id int64) (*Notification, error) {
	// 读取通知
	var n Notification
	var pushQQInt, pushWechatInt int
	err := s.db.QueryRowContext(ctx,
		`SELECT id, title, content, author_name, audience_type, status, push_qq, push_wechat,
		        COALESCE(push_result,''), created_at, COALESCE(published_at,'')
		 FROM notifications WHERE id = ?`, id).
		Scan(&n.ID, &n.Title, &n.Content, &n.AuthorName, &n.AudienceType,
			&n.Status, &pushQQInt, &pushWechatInt, &n.PushResult, &n.CreatedAt, &n.PublishedAt)
	if err != nil {
		return nil, fmt.Errorf("通知不存在: %w", err)
	}
	n.PushQQ = pushQQInt == 1
	n.PushWechat = pushWechatInt == 1

	if n.Status == "published" {
		return nil, fmt.Errorf("通知已发布，不可重复发布")
	}

	// 执行推送
	pushResults := make(map[string]string)

	if n.PushQQ && s.qqWebhook != "" && s.qqWebhook != "${QQ_WEBHOOK_URL}" {
		resp, err := s.pushToWebhook(s.qqWebhook, n.Title, n.Content, "qq")
		if err != nil {
			pushResults["qq"] = fmt.Sprintf("失败: %v", err)
		} else {
			pushResults["qq"] = resp
		}
	} else if n.PushQQ {
		pushResults["qq"] = "未配置QQ Webhook URL"
	}

	if n.PushWechat && s.wechatWebhook != "" && s.wechatWebhook != "${WECHAT_WEBHOOK_URL}" {
		resp, err := s.pushToWebhook(s.wechatWebhook, n.Title, n.Content, "wechat")
		if err != nil {
			pushResults["wechat"] = fmt.Sprintf("失败: %v", err)
		} else {
			pushResults["wechat"] = resp
		}
	} else if n.PushWechat {
		pushResults["wechat"] = "未配置微信Webhook URL"
	}

	pushResultJSON, _ := json.Marshal(pushResults)
	now := time.Now().Format("2006-01-02 15:04:05")

	_, err = s.db.ExecContext(ctx,
		"UPDATE notifications SET status = 'published', push_result = ?, published_at = ?, updated_at = ? WHERE id = ?",
		string(pushResultJSON), now, now, id)
	if err != nil {
		return nil, fmt.Errorf("更新通知状态失败: %w", err)
	}

	n.Status = "published"
	n.PushResult = string(pushResultJSON)
	n.PublishedAt = now
	return &n, nil
}

func (s *NotificationService) Delete(ctx context.Context, user *model.UserContext, id int64) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM notifications WHERE id = ?", id)
	return err
}

func (s *NotificationService) GetWebhookStatus() []WebhookConfig {
	configs := []WebhookConfig{
		{ID: 1, Name: "QQ群推送", Channel: "qq", WebhookURL: maskURL(s.qqWebhook), IsActive: s.qqWebhook != "" && s.qqWebhook != "${QQ_WEBHOOK_URL}"},
		{ID: 2, Name: "微信群推送", Channel: "wechat", WebhookURL: maskURL(s.wechatWebhook), IsActive: s.wechatWebhook != "" && s.wechatWebhook != "${WECHAT_WEBHOOK_URL}"},
	}
	return configs
}

// pushToWebhook 发送通知到 webhook
// 支持：企业微信机器人、钉钉机器人、通用 webhook
func (s *NotificationService) pushToWebhook(webhookURL, title, content, channel string) (string, error) {
	var body []byte
	var err error

	switch channel {
	case "wechat":
		// 企业微信机器人消息格式
		msg := map[string]interface{}{
			"msgtype": "markdown",
			"markdown": map[string]string{
				"content": fmt.Sprintf("## %s\n%s\n\n---\n*蔚小芯 智能推送*", title, content),
			},
		}
		body, err = json.Marshal(msg)
	case "qq":
		// QQ 群机器人格式（兼容常见实现）
		msg := map[string]interface{}{
			"content": fmt.Sprintf("【%s】\n%s", title, content),
		}
		body, err = json.Marshal(msg)
	default:
		// 通用 JSON 格式
		msg := map[string]interface{}{
			"title":   title,
			"content": content,
			"source":  "蔚小芯",
		}
		body, err = json.Marshal(msg)
	}
	if err != nil {
		return "", fmt.Errorf("序列化消息失败: %w", err)
	}

	resp, err := s.httpClient.Post(webhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("推送请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("推送返回状态码 %d", resp.StatusCode)
	}
	return fmt.Sprintf("推送成功 (%d)", resp.StatusCode), nil
}

func maskURL(url string) string {
	if url == "" || url == "${QQ_WEBHOOK_URL}" || url == "${WECHAT_WEBHOOK_URL}" {
		return "未配置"
	}
	if len(url) > 30 {
		return url[:20] + "..." + url[len(url)-10:]
	}
	return url
}
