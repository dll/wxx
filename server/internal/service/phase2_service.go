package service

import (
	"encoding/json"

	"github.com/dll/wxx/server/internal/repository"
)

// Phase2Service 阶段二真实业务服务（积分成就 / 问答广场 / 谈心记录）
type Phase2Service struct {
	repo       *repository.Phase2Repo
	checkinRepo *repository.CheckinRepo
}

// NewPhase2Service 创建阶段二服务
func NewPhase2Service(repo *repository.Phase2Repo, checkinRepo *repository.CheckinRepo) *Phase2Service {
	return &Phase2Service{repo: repo, checkinRepo: checkinRepo}
}

// ── 积分成就 ──

// AchievementData 成就返回（对齐前端 AchievementData 模型）
type AchievementData struct {
	TotalPoints    int                      `json:"total_points"`
	Level          int                      `json:"level"`
	LevelName      string                   `json:"level_name"`
	NextLevelPoints int                     `json:"next_level_points"`
	WeeklyRank     int                      `json:"weekly_rank"`
	Badges         []map[string]interface{} `json:"badges"`
	DataSource     string                   `json:"data_source"`
}

// levelFor 根据总分返回等级
func levelFor(total int) (level int, name string, next int) {
	switch {
	case total >= 2500:
		return 4, "钻石", 0
	case total >= 1000:
		return 3, "黄金", 2500
	case total >= 300:
		return 2, "白银", 1000
	default:
		return 1, "青铜", 300
	}
}

// GetAchievements 基于真实积分/打卡/问答数据计算成就
func (s *Phase2Service) GetAchievements(userID int64) (*AchievementData, error) {
	total, _, recent, err := s.repo.GetPointsSummary(userID, 10)
	if err != nil {
		return nil, err
	}

	checkinCount, _ := s.repo.CountSource(userID, "checkin")
	qaCount, _ := s.repo.CountSource(userID, "qa_post")
	answerCount, _ := s.repo.CountSource(userID, "qa_answer")
	partyCount, _ := s.repo.CountSource(userID, "party")

	streak := 0
	if s.checkinRepo != nil {
		streak = s.checkinRepo.CalcStreak(userID)
	}

	level, name, next := levelFor(total)

	badges := []map[string]interface{}{
		{"id": "b1", "name": "学习打卡", "icon": "local_fire_department",
			"description": "完成 1 次学习打卡", "unlocked": checkinCount > 0, "unlocked_at": ""},
		{"id": "b2", "name": "连续打卡7天", "icon": "calendar_month",
			"description": "连续打卡 7 天", "unlocked": streak >= 7, "unlocked_at": ""},
		{"id": "b3", "name": "问答达人", "icon": "forum",
			"description": "发布 1 个问答", "unlocked": qaCount > 0, "unlocked_at": ""},
		{"id": "b4", "name": "热心解答", "icon": "volunteer_activism",
			"description": "回答 1 次他人问题", "unlocked": answerCount > 0, "unlocked_at": ""},
		{"id": "b5", "name": "思想之星", "icon": "auto_awesome",
			"description": "完成思想政治学习", "unlocked": partyCount > 0, "unlocked_at": ""},
	}
	for _, b := range badges {
		if b["unlocked"] == true {
			if len(recent) > 0 {
				b["unlocked_at"] = recent[0].CreatedAt
			}
		}
	}

	return &AchievementData{
		TotalPoints:     total,
		Level:           level,
		LevelName:       name,
		NextLevelPoints: next,
		WeeklyRank:      0,
		Badges:          badges,
		DataSource:      "real",
	}, nil
}

// AddPoints 记分（供打卡/问答等业务在真实操作后调用）
func (s *Phase2Service) AddPoints(userID int64, points int, reason, source string) error {
	return s.repo.AddPoints(userID, points, reason, source)
}

// ── 问答广场 ──

// CreateQAPost 发布问题（发布后自动加分）
func (s *Phase2Service) CreateQAPost(userID int64, title, content, category string) (int64, error) {
	id, err := s.repo.CreateQAPost(userID, title, content, category)
	if err != nil {
		return 0, err
	}
	_ = s.repo.AddPoints(userID, 10, "发布问题："+title, "qa_post")
	return id, nil
}

// ListRealPosts 列出真实帖子（合并知识库 FAQ 由 handler 完成）
func (s *Phase2Service) ListRealPosts(limit int) ([]*repository.QAPost, error) {
	return s.repo.ListQAPosts(limit)
}

// AnswerPost 回答帖子（自动加分）
func (s *Phase2Service) AnswerPost(postID, userID int64, content string) (int64, error) {
	id, err := s.repo.CreateQAAnswer(postID, userID, content)
	if err != nil {
		return 0, err
	}
	_ = s.repo.AddPoints(userID, 15, "回答他人问题", "qa_answer")
	return id, nil
}

// ListAnswers 列出帖子回答
func (s *Phase2Service) ListAnswers(postID int64) ([]map[string]interface{}, error) {
	return s.repo.ListQAAnswers(postID)
}

// GetPost 帖子详情（含浏览数自增）
func (s *Phase2Service) GetPost(postID int64) (*repository.QAPost, error) {
	_ = s.repo.IncrementPostViews(postID)
	return s.repo.GetQAPost(postID)
}

// ── 谈心谈话 ──

// TalkRecordInput 谈心记录输入
type TalkRecordInput struct {
	StudentID   int64    `json:"student_id"`
	StudentName string   `json:"student_name"`
	Topic       string   `json:"topic"`
	Emotion     string   `json:"emotion"`
	Content     string   `json:"content"`
	Summary     string   `json:"summary"`
	FollowUps   []string `json:"follow_ups"`
}

// SaveTalkRecord 保存谈心记录（真实落库）
func (s *Phase2Service) SaveTalkRecord(counselorID int64, in *TalkRecordInput) (int64, error) {
	fu, _ := json.Marshal(in.FollowUps)
	return s.repo.CreateTalkRecord(counselorID, in.StudentID, in.StudentName, in.Topic, in.Emotion, in.Content, in.Summary, string(fu))
}

// ListTalkRecords 列出辅导员的谈心记录
func (s *Phase2Service) ListTalkRecords(counselorID int64, limit int) ([]map[string]interface{}, error) {
	if limit <= 0 {
		limit = 100
	}
	return s.repo.ListTalkRecords(counselorID, limit)
}
