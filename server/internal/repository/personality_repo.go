package repository

import (
	"database/sql"
	"fmt"

	dbutil "github.com/dll/wxx/server/internal/db"
)

// PersonalityRepo 学生性格洞察数据访问层
type PersonalityRepo struct {
	db *sql.DB
}

// NewPersonalityRepo 创建性格洞察仓库
func NewPersonalityRepo(db *sql.DB) *PersonalityRepo {
	return &PersonalityRepo{db: db}
}

// PersonalityProfile 性格画像实体
type PersonalityProfile struct {
	ID                int64   `json:"id"`
	UserID            int64   `json:"user_id"`
	VisualScore       float64 `json:"visual_score"`
	AuditoryScore     float64 `json:"auditory_score"`
	ReadingScore      float64 `json:"reading_score"`
	KinestheticScore  float64 `json:"kinesthetic_score"`
	Openness          float64 `json:"openness"`
	Conscientiousness float64 `json:"conscientiousness"`
	Extraversion      float64 `json:"extraversion"`
	Agreeableness     float64 `json:"agreeableness"`
	Neuroticism       float64 `json:"neuroticism"`
	PersonalityType   string  `json:"personality_type"`
	TypeLabel         string  `json:"type_label"`
	Description       string  `json:"description"`
	LearningStyle     string  `json:"learning_style"`
	Strengths         string  `json:"strengths"`          // JSON 数组
	Weaknesses        string  `json:"weaknesses"`         // JSON 数组
	CareerSuggestions string  `json:"career_suggestions"` // JSON 数组
	RawAnswers        string  `json:"raw_answers"`
	DataSource        string  `json:"data_source"`
	ComputedAt        string  `json:"computed_at"`
}

// GetByUserID 读取用户性格画像；无记录返回 (nil, nil)
func (r *PersonalityRepo) GetByUserID(userID int64) (*PersonalityProfile, error) {
	p := &PersonalityProfile{}
	err := r.db.QueryRow(`
		SELECT id, user_id,
		       visual_score, auditory_score, reading_score, kinesthetic_score,
		       openness, conscientiousness, extraversion, agreeableness, neuroticism,
		       personality_type, type_label, description, learning_style,
		       strengths, weaknesses, career_suggestions,
		       raw_answers, data_source, computed_at
		FROM student_personality WHERE user_id = ?`, userID).
		Scan(&p.ID, &p.UserID,
			&p.VisualScore, &p.AuditoryScore, &p.ReadingScore, &p.KinestheticScore,
			&p.Openness, &p.Conscientiousness, &p.Extraversion, &p.Agreeableness, &p.Neuroticism,
			&p.PersonalityType, &p.TypeLabel, &p.Description, &p.LearningStyle,
			&p.Strengths, &p.Weaknesses, &p.CareerSuggestions,
			&p.RawAnswers, &p.DataSource, &p.ComputedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("读取性格画像失败: %w", err)
	}
	return p, nil
}

// Upsert 写入/更新性格画像（按 user_id 唯一）
func (r *PersonalityRepo) Upsert(p *PersonalityProfile) error {
	stmt := `
		INSERT INTO student_personality (
			user_id, visual_score, auditory_score, reading_score, kinesthetic_score,
			openness, conscientiousness, extraversion, agreeableness, neuroticism,
			personality_type, type_label, description, learning_style,
			strengths, weaknesses, career_suggestions,
			raw_answers, data_source, computed_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id) DO UPDATE SET
			visual_score=excluded.visual_score, auditory_score=excluded.auditory_score,
			reading_score=excluded.reading_score, kinesthetic_score=excluded.kinesthetic_score,
			openness=excluded.openness, conscientiousness=excluded.conscientiousness,
			extraversion=excluded.extraversion, agreeableness=excluded.agreeableness,
			neuroticism=excluded.neuroticism,
			personality_type=excluded.personality_type, type_label=excluded.type_label,
			description=excluded.description, learning_style=excluded.learning_style,
			strengths=excluded.strengths, weaknesses=excluded.weaknesses,
			career_suggestions=excluded.career_suggestions,
			raw_answers=excluded.raw_answers, data_source=excluded.data_source,
			computed_at=excluded.computed_at, updated_at=CURRENT_TIMESTAMP`
	_, err := r.db.Exec(dbutil.AdaptForDriver(stmt, dbutil.DriverOf(r.db)),
		p.UserID, p.VisualScore, p.AuditoryScore, p.ReadingScore, p.KinestheticScore,
		p.Openness, p.Conscientiousness, p.Extraversion, p.Agreeableness, p.Neuroticism,
		p.PersonalityType, p.TypeLabel, p.Description, p.LearningStyle,
		p.Strengths, p.Weaknesses, p.CareerSuggestions,
		p.RawAnswers, p.DataSource)
	if err != nil {
		return fmt.Errorf("写入性格画像失败: %w", err)
	}
	return nil
}
