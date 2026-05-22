package service

import (
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
	feedbackRepo *repository.FeedbackRepo
	// 可选：用户反馈"回答有误"时回调（用于失效 FAQ 缓存等）
	// messageID 是用户在前端记录的消息 id，由调用方决定如何用它定位原问题
	onAnswerError func(messageID, content string)
}

// NewFeedbackService 创建反馈服务
func NewFeedbackService(feedbackRepo *repository.FeedbackRepo) *FeedbackService {
	return &FeedbackService{feedbackRepo: feedbackRepo}
}

// SetAnswerErrorHook 注入"回答有误"反馈钩子（如失效 FAQ 缓存）
// 钩子在反馈成功保存后异步执行，不影响反馈提交结果
func (s *FeedbackService) SetAnswerErrorHook(fn func(messageID, content string)) {
	s.onAnswerError = fn
}

// Submit 提交反馈（含可选截图）
func (s *FeedbackService) Submit(userID int64, username string, req *model.FeedbackCreateRequest) (*model.Feedback, error) {
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

// Resolve 处理反馈（标记为已解决或驳回，含可选回复）
func (s *FeedbackService) Resolve(feedbackID, resolvedBy, status, reply string) (*model.Feedback, error) {
	fb, err := s.feedbackRepo.GetByFeedbackID(feedbackID)
	if err != nil {
		return nil, fmt.Errorf("查询反馈失败: %w", err)
	}
	if fb == nil {
		return nil, fmt.Errorf("反馈不存在: %s", feedbackID)
	}
	if fb.Status != "pending" {
		return nil, fmt.Errorf("反馈状态为 %s，不可重复处理", fb.Status)
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	fb.Status = status
	fb.ResolvedBy = resolvedBy
	fb.ResolvedAt = &now
	fb.Reply = reply

	if err := s.feedbackRepo.Update(fb); err != nil {
		return nil, fmt.Errorf("更新反馈状态失败: %w", err)
	}

	log.Printf("反馈已处理 feedback_id=%s status=%s reply=%s by=%s",
		feedbackID, status, reply, resolvedBy)
	return fb, nil
}
