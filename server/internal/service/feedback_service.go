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
}

// NewFeedbackService 创建反馈服务
func NewFeedbackService(feedbackRepo *repository.FeedbackRepo) *FeedbackService {
	return &FeedbackService{feedbackRepo: feedbackRepo}
}

// Submit 提交反馈
func (s *FeedbackService) Submit(userID int64, username string, req *model.FeedbackCreateRequest) (*model.Feedback, error) {
	fb := &model.Feedback{
		FeedbackID: "fb-" + uuid.New().String()[:8],
		UserID:     userID,
		Username:   username,
		MessageID:  req.MessageID,
		ResourceID: req.ResourceID,
		Category:   req.Category,
		Content:    req.Content,
		Status:     "pending",
	}

	id, err := s.feedbackRepo.Create(fb)
	if err != nil {
		return nil, fmt.Errorf("保存反馈失败: %w", err)
	}

	fb.ID = id
	log.Printf("用户反馈已提交 feedback_id=%s category=%s by=%s", fb.FeedbackID, fb.Category, username)
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

// Resolve 处理反馈（标记为已解决或驳回）
func (s *FeedbackService) Resolve(feedbackID, resolvedBy, status string) (*model.Feedback, error) {
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

	if err := s.feedbackRepo.Update(fb); err != nil {
		return nil, fmt.Errorf("更新反馈状态失败: %w", err)
	}

	log.Printf("反馈已处理 feedback_id=%s status=%s by=%s", feedbackID, status, resolvedBy)
	return fb, nil
}
