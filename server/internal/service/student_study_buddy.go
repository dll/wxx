package service

import (
	"context"
	"fmt"
	"sort"

	"github.com/dll/wxx/server/internal/model"
)

// StudyBuddyMatch 学友匹配结果。
type StudyBuddyMatch struct {
	Matches     []map[string]interface{} `json:"matches"`
	MatchReason string                   `json:"match_reason"`
	DataSource  string                   `json:"data_source"`
}

// GenerateStudyBuddyMatches 基于真实院系资料匹配学习伙伴，缺少资料时返回安全兜底。
func (s *StudentService) GenerateStudyBuddyMatches(ctx context.Context, userID int64) *StudyBuddyMatch {
	_ = ctx
	if s.userRepo == nil {
		return studyBuddyFallback()
	}
	me, err := s.userRepo.GetByID(userID)
	if err != nil || me == nil || (me.College == "" && me.Major == "" && me.ClassName == "") {
		return studyBuddyFallback()
	}
	q := &model.UserQuery{Role: "student", Status: "active", Limit: 30}
	if me.Major != "" {
		q.Major = me.Major
	} else if me.College != "" {
		q.College = me.College
	}
	peers, _, err := s.userRepo.ListAdvanced(q)
	if err != nil || len(peers) == 0 {
		return studyBuddyFallback()
	}
	matches := make([]map[string]interface{}, 0, 5)
	for _, p := range peers {
		if p.ID == userID {
			continue
		}
		score := 50
		switch {
		case me.ClassName != "" && p.ClassName == me.ClassName:
			score += 40
		case me.Major != "" && p.Major == me.Major:
			score += 30
		case me.College != "" && p.College == me.College:
			score += 15
		}
		if me.EnrollmentYear != "" && p.EnrollmentYear == me.EnrollmentYear {
			score += 15
		}
		if score > 99 {
			score = 99
		}
		reason := "同学院"
		if me.ClassName != "" && p.ClassName == me.ClassName {
			reason = "同班同学"
		} else if me.Major != "" && p.Major == me.Major {
			reason = "同专业"
		}
		matches = append(matches, map[string]interface{}{"name": maskDisplayName(p.DisplayName), "match_score": score, "major": p.Major, "class_name": p.ClassName, "reason": reason})
		if len(matches) >= 5 {
			break
		}
	}
	if len(matches) == 0 {
		return studyBuddyFallback()
	}
	sort.Slice(matches, func(i, j int) bool { return matches[i]["match_score"].(int) > matches[j]["match_score"].(int) })
	return &StudyBuddyMatch{Matches: matches, MatchReason: fmt.Sprintf("基于你的专业「%s」与班级，为你匹配了 %d 位可结伴学习的同学（姓名已脱敏保护隐私）。", me.Major, len(matches)), DataSource: "real"}
}

func studyBuddyFallback() *StudyBuddyMatch {
	return &StudyBuddyMatch{Matches: []map[string]interface{}{{"name": "张*", "match_score": 92, "reason": "同专业", "major": "计算机科学与技术"}, {"name": "李*", "match_score": 85, "reason": "同学院"}}, MatchReason: "暂无足够的同学资料用于精准匹配，以下为示例。完善院系班级信息后可获得真实推荐。", DataSource: "fallback"}
}
