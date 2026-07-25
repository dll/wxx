package repository

import (
	"database/sql"
	"fmt"
	"strconv"
)

// TwinRepo 个人数字孪生数据访问层
//
// 职责：
//  1. 读写 student_profile_snapshot 五维快照；
//  2. 从既有业务表实时聚合五维原始指标（成绩/竞赛/党建/情感/社团），
//     供 service 计算五维分数。
//
// 注意：student_grades.user_id 为 TEXT，其余业务表 user_id 多为 INTEGER，
// 聚合时按各表实际类型传参。
type TwinRepo struct {
	db *sql.DB
}

// NewTwinRepo 创建数字孪生仓库
func NewTwinRepo(db *sql.DB) *TwinRepo {
	return &TwinRepo{db: db}
}

// TwinRawMetrics 五维原始指标（未归一化），由各业务表聚合而来
type TwinRawMetrics struct {
	// 学业
	AvgGPA        float64 // 平均绩点
	AvgScore      float64 // 平均分
	PassRate      float64 // 通过率 0-1
	CourseCount   int     // 成绩记录数
	CreditsEarned float64 // 已获学分

	// 能力
	CompetitionCount int // 竞赛报名数
	AwardCount       int // 获奖数（已提交作品且通过）
	PlanCount        int // 学习规划数
	PlanDoneCount    int // 已完成规划数

	// 思想
	PartyStageRank   int // 党建阶段序（0=无 1=申请 ... 5=正式党员）
	PartyStudyCount  int // 党建学习记录数
	PartyStudyMinute int // 党建学习总时长（分钟）

	// 情感
	EmotionLogCount int     // 情感日志数
	AvgEmotionScore float64 // 平均情绪分（原始，越高越负面）
	HighRiskCount   int     // 高风险次数

	// 社交
	ClubCount        int // 加入社团数
	ActivityRegCount int // 活动报名数
}

// ClassPercentileBasis 班级匿名百分位计算基准（课程学情看板用）
type ClassPercentileBasis struct {
	ClassAvgGPA float64
	ClassSize   int
}

// AggregateRawMetrics 从各业务表聚合某学生的五维原始指标。
// userID 为整型主键；对 user_id 为 TEXT 的表（student_grades）用字符串形式查询。
func (r *TwinRepo) AggregateRawMetrics(userID int64) (*TwinRawMetrics, error) {
	m := &TwinRawMetrics{}
	uidStr := strconv.FormatInt(userID, 10)

	// ── 学业：student_grades（user_id 为 TEXT）──
	// COALESCE 保证无成绩时返回 0 而非 NULL 扫描失败
	err := r.db.QueryRow(`
		SELECT
			COALESCE(AVG(gpa), 0),
			COALESCE(AVG(score), 0),
			COALESCE(SUM(passed) * 1.0 / NULLIF(COUNT(*), 0), 0),
			COUNT(*),
			COALESCE(SUM(credits_earned), 0)
		FROM student_grades WHERE user_id = ?`, uidStr).
		Scan(&m.AvgGPA, &m.AvgScore, &m.PassRate, &m.CourseCount, &m.CreditsEarned)
	if err != nil {
		return nil, fmt.Errorf("聚合学业指标失败: %w", err)
	}

	// ── 能力：竞赛报名 + 获奖（作品已提交视为参与，award 字段非空视为获奖）──
	if err := r.db.QueryRow(`
		SELECT COUNT(*),
		       COALESCE(SUM(CASE WHEN status = 'awarded' THEN 1 ELSE 0 END), 0)
		FROM competition_registrations WHERE user_id = ?`, userID).
		Scan(&m.CompetitionCount, &m.AwardCount); err != nil {
		// 表可能字段不同，容错为 0 不阻断
		m.CompetitionCount, m.AwardCount = 0, 0
	}

	// 学习规划完成度（status 枚举：draft/submitted/approved/in_progress/completed）
	if err := r.db.QueryRow(`
		SELECT COUNT(*),
		       COALESCE(SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END), 0)
		FROM student_plans WHERE user_id = ?`, userID).
		Scan(&m.PlanCount, &m.PlanDoneCount); err != nil {
		m.PlanCount, m.PlanDoneCount = 0, 0
	}

	// ── 思想：党建进度 + 学习记录 ──
	var stage sql.NullString
	if err := r.db.QueryRow(
		`SELECT status FROM party_progress WHERE user_id = ? ORDER BY updated_at DESC LIMIT 1`,
		userID).Scan(&stage); err == nil && stage.Valid {
		m.PartyStageRank = partyStageRank(stage.String)
	}
	if err := r.db.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(duration), 0)
		FROM party_study_records WHERE user_id = ?`, userID).
		Scan(&m.PartyStudyCount, &m.PartyStudyMinute); err != nil {
		m.PartyStudyCount, m.PartyStudyMinute = 0, 0
	}

	// ── 情感：emotion_logs（score 越高越负面，risk_level=high 计高风险）──
	if err := r.db.QueryRow(`
		SELECT COUNT(*), COALESCE(AVG(score), 0),
		       COALESCE(SUM(CASE WHEN risk_level = 'high' THEN 1 ELSE 0 END), 0)
		FROM emotion_logs WHERE user_id = ?`, userID).
		Scan(&m.EmotionLogCount, &m.AvgEmotionScore, &m.HighRiskCount); err != nil {
		m.EmotionLogCount, m.AvgEmotionScore, m.HighRiskCount = 0, 0, 0
	}

	// ── 社交：社团成员 + 活动报名 ──
	if err := r.db.QueryRow(
		`SELECT COUNT(*) FROM club_members WHERE user_id = ?`, userID).
		Scan(&m.ClubCount); err != nil {
		m.ClubCount = 0
	}
	if err := r.db.QueryRow(
		`SELECT COUNT(*) FROM club_activity_registrations WHERE user_id = ?`, userID).
		Scan(&m.ActivityRegCount); err != nil {
		m.ActivityRegCount = 0
	}

	return m, nil
}

// partyStageRank 把党建状态码映射为 0-5 序，用于思想维度打分
func partyStageRank(status string) int {
	switch status {
	case "applicant":
		return 1
	case "activist":
		return 2
	case "development":
		return 3
	case "probation":
		return 4
	case "member":
		return 5
	default:
		return 0
	}
}

// TwinSnapshot 数字孪生快照实体（对应 student_profile_snapshot 表）
type TwinSnapshot struct {
	UserID           int64   `json:"user_id"`
	OwnerScope       string  `json:"owner_scope"`
	OwnerID          string  `json:"owner_id"`
	College          string  `json:"college"`
	Major            string  `json:"major"`
	ClassName        string  `json:"class_name"`
	AcademicScore    float64 `json:"academic_score"`
	AbilityScore     float64 `json:"ability_score"`
	IdeologicalScore float64 `json:"ideological_score"`
	EmotionalScore   float64 `json:"emotional_score"`
	SocialScore      float64 `json:"social_score"`
	AIInterpretation string  `json:"ai_interpretation"`
	GapAnalysis      string  `json:"gap_analysis"`
	StageAdvice      string  `json:"stage_advice"`
	ComputedAt       string  `json:"computed_at"`
}

// GetSnapshot 读取某学生的最近快照；无快照返回 (nil, nil)
func (r *TwinRepo) GetSnapshot(userID int64) (*TwinSnapshot, error) {
	s := &TwinSnapshot{}
	err := r.db.QueryRow(`
		SELECT user_id, owner_scope, owner_id, college, major, class_name,
		       academic_score, ability_score, ideological_score, emotional_score, social_score,
		       ai_interpretation, gap_analysis, stage_advice, computed_at
		FROM student_profile_snapshot WHERE user_id = ?`, userID).
		Scan(&s.UserID, &s.OwnerScope, &s.OwnerID, &s.College, &s.Major, &s.ClassName,
			&s.AcademicScore, &s.AbilityScore, &s.IdeologicalScore, &s.EmotionalScore, &s.SocialScore,
			&s.AIInterpretation, &s.GapAnalysis, &s.StageAdvice, &s.ComputedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("读取数字孪生快照失败: %w", err)
	}
	return s, nil
}

// UpsertSnapshot 写入/更新快照（按 user_id 唯一）
func (r *TwinRepo) UpsertSnapshot(s *TwinSnapshot) error {
	_, err := r.db.Exec(`
		INSERT INTO student_profile_snapshot (
			user_id, owner_scope, owner_id, college, major, class_name,
			academic_score, ability_score, ideological_score, emotional_score, social_score,
			ai_interpretation, gap_analysis, stage_advice, computed_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))
		ON CONFLICT(user_id) DO UPDATE SET
			owner_scope=excluded.owner_scope, owner_id=excluded.owner_id,
			college=excluded.college, major=excluded.major, class_name=excluded.class_name,
			academic_score=excluded.academic_score, ability_score=excluded.ability_score,
			ideological_score=excluded.ideological_score, emotional_score=excluded.emotional_score,
			social_score=excluded.social_score, ai_interpretation=excluded.ai_interpretation,
			gap_analysis=excluded.gap_analysis, stage_advice=excluded.stage_advice,
			computed_at=excluded.computed_at, updated_at=datetime('now')`,
		s.UserID, s.OwnerScope, s.OwnerID, s.College, s.Major, s.ClassName,
		s.AcademicScore, s.AbilityScore, s.IdeologicalScore, s.EmotionalScore, s.SocialScore,
		s.AIInterpretation, s.GapAnalysis, s.StageAdvice)
	if err != nil {
		return fmt.Errorf("写入数字孪生快照失败: %w", err)
	}
	return nil
}

// GetClassBasis 计算班级平均 GPA 与规模，供课程学情看板的匿名百分位
func (r *TwinRepo) GetClassBasis(className string) (*ClassPercentileBasis, error) {
	if className == "" {
		return &ClassPercentileBasis{}, nil
	}
	b := &ClassPercentileBasis{}
	// student_grades.user_id 为 TEXT，需与 users.id 转换比对
	err := r.db.QueryRow(`
		SELECT COALESCE(AVG(g.gpa), 0), COUNT(DISTINCT g.user_id)
		FROM student_grades g
		JOIN users u ON g.user_id = CAST(u.id AS TEXT)
		WHERE u.class_name = ?`, className).
		Scan(&b.ClassAvgGPA, &b.ClassSize)
	if err != nil {
		return nil, fmt.Errorf("计算班级基准失败: %w", err)
	}
	return b, nil
}

// ListSnapshotsByScope 按归属聚合快照（辅导员看板/学院大屏复用）
func (r *TwinRepo) ListSnapshotsByScope(ownerScope, ownerID, college, className string, limit int) ([]*TwinSnapshot, error) {
	query := `
		SELECT user_id, owner_scope, owner_id, college, major, class_name,
		       academic_score, ability_score, ideological_score, emotional_score, social_score,
		       ai_interpretation, gap_analysis, stage_advice, computed_at
		FROM student_profile_snapshot WHERE 1=1`
	var args []interface{}
	if ownerScope != "" {
		query += " AND owner_scope = ?"
		args = append(args, ownerScope)
	}
	if ownerID != "" {
		query += " AND owner_id = ?"
		args = append(args, ownerID)
	}
	if college != "" {
		query += " AND college = ?"
		args = append(args, college)
	}
	if className != "" {
		query += " AND class_name = ?"
		args = append(args, className)
	}
	query += " ORDER BY computed_at DESC"
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("按 scope 查询快照失败: %w", err)
	}
	defer rows.Close()

	var list []*TwinSnapshot
	for rows.Next() {
		s := &TwinSnapshot{}
		if err := rows.Scan(&s.UserID, &s.OwnerScope, &s.OwnerID, &s.College, &s.Major, &s.ClassName,
			&s.AcademicScore, &s.AbilityScore, &s.IdeologicalScore, &s.EmotionalScore, &s.SocialScore,
			&s.AIInterpretation, &s.GapAnalysis, &s.StageAdvice, &s.ComputedAt); err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, rows.Err()
}
