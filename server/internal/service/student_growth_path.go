package service

import "context"

// GenerateGrowthPath 生成成长路径（基于数字孪生五维快照 + 学业阶段 → 分阶段里程碑 + LLM 总结）。
// 无快照数据时返回 (nil, nil)，由 handler 回落通用 AI 文案。
func (s *StudentService) GenerateGrowthPath(ctx context.Context, userID int64) (*GrowthPathResult, error) {
	if s.twinRepo == nil || s.userRepo == nil {
		return nil, nil
	}
	user, err := s.userRepo.GetByID(userID)
	if err != nil || user == nil {
		return nil, nil
	}
	snap, err := s.twinRepo.GetSnapshot(userID)
	if err != nil {
		return nil, err
	}
	if snap == nil {
		return nil, nil
	}

	dims := map[string]float64{
		"学业": snap.AcademicScore, "能力": snap.AbilityScore, "思想": snap.IdeologicalScore,
		"情感": snap.EmotionalScore, "社交": snap.SocialScore,
	}
	strongest, weakest := "学业", "学业"
	for k, v := range dims {
		if v > dims[strongest] {
			strongest = k
		}
		if v < dims[weakest] {
			weakest = k
		}
	}

	res := &GrowthPathResult{
		UserDisplayName: user.DisplayName,
		CurrentStage:    inferAcademicStage(user.EnrollmentYear),
		AcademicScore:   snap.AcademicScore,
		AbilityScore:    snap.AbilityScore,
		StrongestDim:    strongest,
		WeakestDim:      weakest,
		DataSource:      "real",
	}
	res.Milestones = buildGrowthMilestones(weakest)
	res.Summary = s.buildGrowthSummary(ctx, user.DisplayName, res, weakest)
	return res, nil
}
