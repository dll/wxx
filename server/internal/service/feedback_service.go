package service

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/util"
	"github.com/google/uuid"
)

// FeedbackService 用户反馈业务服务
type FeedbackService struct {
	feedbackRepo   *repository.FeedbackRepo
	userRepo       *repository.UserRepo
	screenshotRepo *repository.FeedbackScreenshotRepo
	db             *sql.DB
	// 可选：用户反馈"回答有误"时回调（用于失效 FAQ 缓存等）
	// messageID 是用户在前端记录的消息 id，由调用方决定如何用它定位原问题
	onAnswerError func(messageID, content string)
}

// NewFeedbackService 创建反馈服务
func NewFeedbackService(feedbackRepo *repository.FeedbackRepo, userRepo *repository.UserRepo, screenshotRepo *repository.FeedbackScreenshotRepo) *FeedbackService {
	return &FeedbackService{feedbackRepo: feedbackRepo, userRepo: userRepo, screenshotRepo: screenshotRepo}
}

// SetDB 设置数据库连接（用于发送站内通知）
func (s *FeedbackService) SetDB(db *sql.DB) {
	s.db = db
}

// SetAnswerErrorHook 注入"回答有误"反馈钩子（如失效 FAQ 缓存）
// 钩子在反馈成功保存后异步执行，不影响反馈提交结果
func (s *FeedbackService) SetAnswerErrorHook(fn func(messageID, content string)) {
	s.onAnswerError = fn
}

// Submit 提交反馈（含可选截图）
func (s *FeedbackService) Submit(userID int64, username string, req *model.FeedbackCreateRequest) (*model.Feedback, error) {
	// 校验 token 中的用户是否仍存在于当前数据库（Vercel /tmp 数据库冷启动后旧 token 可能失效）
	if s.userRepo != nil {
		u, err := s.userRepo.GetByID(userID)
		if err != nil {
			return nil, ErrUserNotFound
		}
		if u == nil {
			return nil, ErrUserNotFound
		}
	}

	fb := &model.Feedback{
		FeedbackID:    "fb-" + uuid.New().String()[:8],
		UserID:        userID,
		Username:      username,
		MessageID:     req.MessageID,
		ResourceID:    req.ResourceID,
		Category:      req.Category,
		Content:       req.Content,
		ScreenshotURL: req.ScreenshotURL,
		Status:        "pending",
	}

	id, err := s.feedbackRepo.Create(fb)
	if err != nil {
		return nil, fmt.Errorf("保存反馈失败: %w", err)
	}

	fb.ID = id
	log.Printf("用户反馈已提交 feedback_id=%s category=%s has_screenshot=%v by=%s",
		fb.FeedbackID, fb.Category, fb.ScreenshotURL != "", username)

	// 记录处理日志
	_ = s.feedbackRepo.AddLog(fb.FeedbackID, "submit", username, "用户提交反馈")

	// 仅 "回答有误" 类反馈触发钩子（异步，不影响响应）
	if req.Category == "answer_error" && s.onAnswerError != nil {
		go s.onAnswerError(req.MessageID, req.Content)
	}
	return fb, nil
}

// List 分页查询反馈列表
func (s *FeedbackService) List(status string, page, pageSize int) ([]*model.Feedback, int, error) {
	offset, _, _ := util.Paginate(page, pageSize)

	items, err := s.feedbackRepo.List(status, offset, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("查询反馈列表失败: %w", err)
	}

	total, err := s.feedbackRepo.Count(status)
	if err != nil {
		return nil, 0, fmt.Errorf("统计反馈总数失败: %w", err)
	}

	return items, total, nil
}

// ListMine 查询指定用户自己提交的反馈（用于"我的反馈"页面）
func (s *FeedbackService) ListMine(userID int64, status string, page, pageSize int) ([]*model.Feedback, int, error) {
	offset, _, _ := util.Paginate(page, pageSize)

	items, err := s.feedbackRepo.ListByUser(userID, status, offset, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("查询我的反馈失败: %w", err)
	}

	total, err := s.feedbackRepo.CountByUser(userID, status)
	if err != nil {
		return nil, 0, fmt.Errorf("统计我的反馈总数失败: %w", err)
	}

	return items, total, nil
}

// Get 获取单条反馈详情
func (s *FeedbackService) Get(feedbackID string) (*model.Feedback, error) {
	fb, err := s.feedbackRepo.GetByFeedbackID(feedbackID)
	if err != nil {
		return nil, fmt.Errorf("查询反馈失败: %w", err)
	}
	return fb, nil
}

// Resolve 处理反馈（标记为处理中/已解决/驳回，含可选回复）
func (s *FeedbackService) Resolve(feedbackID, resolvedBy, status, reply string) (*model.Feedback, error) {
	fb, err := s.feedbackRepo.GetByFeedbackID(feedbackID)
	if err != nil {
		return nil, fmt.Errorf("查询反馈失败: %w", err)
	}
	if fb == nil {
		return nil, fmt.Errorf("反馈不存在: %s", feedbackID)
	}
	// 允许的状态流转：pending→processing, pending→resolved, pending→dismissed, processing→resolved, processing→dismissed
	validTransitions := map[string][]string{
		"pending":    {"processing", "resolved", "dismissed"},
		"processing": {"resolved", "dismissed"},
		"resolved":   {},
		"dismissed":  {},
	}
	allowed, ok := validTransitions[fb.Status]
	if !ok {
		return nil, fmt.Errorf("未知的反馈状态: %s", fb.Status)
	}
	valid := false
	for _, s := range allowed {
		if s == status {
			valid = true
			break
		}
	}
	if !valid {
		return nil, fmt.Errorf("反馈状态为 %s，不可变更为 %s", fb.Status, status)
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	oldStatus := fb.Status
	fb.Status = status
	fb.ResolvedBy = resolvedBy
	fb.ResolvedAt = &now
	if reply != "" {
		fb.Reply = reply
	}

	if err := s.feedbackRepo.Update(fb); err != nil {
		return nil, fmt.Errorf("更新反馈状态失败: %w", err)
	}

	// 记录处理日志
	action := "status_change"
	detail := fmt.Sprintf("状态从 %s 变更为 %s", oldStatus, status)
	if reply != "" {
		detail += fmt.Sprintf("，回复：%s", reply)
	}
	_ = s.feedbackRepo.AddLog(feedbackID, action, resolvedBy, detail)

	log.Printf("反馈已处理 feedback_id=%s status=%s reply=%s by=%s",
		feedbackID, status, reply, resolvedBy)

	// 如果状态变为 resolved，异步发送站内通知
	if status == "resolved" {
		go s.sendResolveNotification(fb)
	}

	return fb, nil
}

// sendResolveNotification 发送反馈解决通知
func (s *FeedbackService) sendResolveNotification(fb *model.Feedback) {
	if s.db == nil {
		return
	}
	title := "您的反馈已解决"
	content := fmt.Sprintf("您提交的反馈（%s）已处理完成，回复：%s", fb.FeedbackID, fb.Reply)
	if fb.Reply == "" {
		content = fmt.Sprintf("您提交的反馈（%s）已处理完成。", fb.FeedbackID)
	}

	_, err := s.db.Exec(
		`INSERT INTO user_notifications (user_id, title, content, type, related_type, related_id, is_read)
		 VALUES (?, ?, ?, 'feedback', 'feedback', ?, 0)`,
		fb.UserID, title, content, fb.ID,
	)
	if err != nil {
		log.Printf("发送反馈解决通知失败: feedback_id=%s user_id=%d err=%v", fb.FeedbackID, fb.UserID, err)
	}
}

// GetStats 获取反馈统计数据
func (s *FeedbackService) GetStats() (*model.FeedbackStats, error) {
	total, err := s.feedbackRepo.Count("")
	if err != nil {
		return nil, fmt.Errorf("统计总数失败: %w", err)
	}

	byStatus, err := s.feedbackRepo.CountByStatus()
	if err != nil {
		return nil, fmt.Errorf("按状态统计失败: %w", err)
	}

	byCategory, err := s.feedbackRepo.CountByCategory()
	if err != nil {
		return nil, fmt.Errorf("按分类统计失败: %w", err)
	}

	weekTrend, err := s.feedbackRepo.WeekTrend()
	if err != nil {
		return nil, fmt.Errorf("获取周趋势失败: %w", err)
	}

	topIssues, err := s.feedbackRepo.TopIssues(10)
	if err != nil {
		return nil, fmt.Errorf("获取热门问题失败: %w", err)
	}

	avgHours, err := s.feedbackRepo.AvgResolveHours()
	if err != nil {
		return nil, fmt.Errorf("获取平均解决时长失败: %w", err)
	}

	return &model.FeedbackStats{
		Total:           total,
		ByStatus:        byStatus,
		ByCategory:      byCategory,
		WeekTrend:       weekTrend,
		TopIssues:       topIssues,
		AvgResolveHours: avgHours,
	}, nil
}

// LinkResource 关联知识库资源
func (s *FeedbackService) LinkResource(feedbackID, resourceID, note, operator string) error {
	fb, err := s.feedbackRepo.GetByFeedbackID(feedbackID)
	if err != nil {
		return fmt.Errorf("查询反馈失败: %w", err)
	}
	if fb == nil {
		return fmt.Errorf("反馈不存在: %s", feedbackID)
	}

	if err := s.feedbackRepo.LinkResource(feedbackID, resourceID, note, operator); err != nil {
		return fmt.Errorf("关联知识资源失败: %w", err)
	}

	// 记录处理日志
	detail := fmt.Sprintf("关联资源 %s", resourceID)
	if note != "" {
		detail += fmt.Sprintf("，备注：%s", note)
	}
	_ = s.feedbackRepo.AddLog(feedbackID, "link_resource", operator, detail)

	log.Printf("反馈已关联知识资源 feedback_id=%s resource_id=%s by=%s",
		feedbackID, resourceID, operator)
	return nil
}

// Rate 满意度评价
func (s *FeedbackService) Rate(feedbackID string, userID int64, rating int, comment string) error {
	fb, err := s.feedbackRepo.GetByFeedbackID(feedbackID)
	if err != nil {
		return fmt.Errorf("查询反馈失败: %w", err)
	}
	if fb == nil {
		return fmt.Errorf("反馈不存在: %s", feedbackID)
	}
	if fb.UserID != userID {
		return fmt.Errorf("无权评价此反馈")
	}
	if fb.Status != "resolved" {
		return fmt.Errorf("仅已解决的反馈可评价")
	}
	if fb.Rating > 0 {
		return fmt.Errorf("该反馈已评价过")
	}

	if rating < 1 || rating > 5 {
		return fmt.Errorf("评分必须在 1-5 之间")
	}

	if err := s.feedbackRepo.UpdateRating(feedbackID, rating, comment); err != nil {
		return fmt.Errorf("保存评分失败: %w", err)
	}

	// 记录处理日志
	detail := fmt.Sprintf("用户评分 %d 星", rating)
	if comment != "" {
		detail += fmt.Sprintf("，评价：%s", comment)
	}
	_ = s.feedbackRepo.AddLog(feedbackID, "rate", fb.Username, detail)

	log.Printf("反馈已评价 feedback_id=%s rating=%d by=user_%d",
		feedbackID, rating, userID)
	return nil
}

// ListLogs 获取反馈处理记录
func (s *FeedbackService) ListLogs(feedbackID string) ([]*model.FeedbackLog, error) {
	logs, err := s.feedbackRepo.ListLogs(feedbackID)
	if err != nil {
		return nil, fmt.Errorf("获取处理记录失败: %w", err)
	}
	return logs, nil
}

// SaveScreenshot 保存截图数据
func (s *FeedbackService) SaveScreenshot(filename, mime, encoded, uploader string, size int64) error {
	return s.screenshotRepo.Save(filename, mime, encoded, uploader, size)
}

// GetScreenshot 按文件名获取截图
func (s *FeedbackService) GetScreenshot(filename string) (dataB64, mime string, err error) {
	return s.screenshotRepo.GetByFilename(filename)
}
