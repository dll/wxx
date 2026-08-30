// Package model 心理健康教育（P4-d：从 handler 下沉为共享 DTO）。
package model

// PsychScale 量表列表项
type PsychScale struct {
	ID               int64  `json:"id"`
	ScaleID          string `json:"scale_id"`
	Name             string `json:"name"`
	Abbreviation     string `json:"abbreviation"`
	Category         string `json:"category"`
	Description      string `json:"description"`
	QuestionCount    int    `json:"question_count"`
	EstimatedMinutes int    `json:"estimated_minutes"`
	IsCrisis         int    `json:"is_crisis"`
	CreatedAt        string `json:"created_at"`
}

// PsychScaleDetail 量表详情（含题目 JSON 与计分说明）
type PsychScaleDetail struct {
	ID               int64  `json:"id"`
	ScaleID          string `json:"scale_id"`
	Name             string `json:"name"`
	Abbreviation     string `json:"abbreviation"`
	Category         string `json:"category"`
	Description      string `json:"description"`
	QuestionCount    int    `json:"question_count"`
	EstimatedMinutes int    `json:"estimated_minutes"`
	ScoringMethod    string `json:"scoring_method"`
	Interpretation   string `json:"interpretation"`
	QuestionsJSON    string `json:"questions_json"`
	IsCrisis         int    `json:"is_crisis"`
	Status           string `json:"status"`
	CreatedAt        string `json:"created_at"`
}

// AssessmentRecord 心理测评记录
type AssessmentRecord struct {
	ID            int64   `json:"id"`
	RecordID      string  `json:"record_id"`
	ScaleID       string  `json:"scale_id"`
	ScaleName     string  `json:"scale_name"`
	TotalScore    float64 `json:"total_score"`
	Level         string  `json:"level"`
	ResultSummary string  `json:"result_summary"`
	Suggestion    string  `json:"suggestion"`
	CompletedAt   string  `json:"completed_at"`
	CreatedAt     string  `json:"created_at"`
}

// PsychArticle 心理科普文章列表项
type PsychArticle struct {
	ID         int64  `json:"id"`
	ArticleID  string `json:"article_id"`
	Title      string `json:"title"`
	Category   string `json:"category"`
	Summary    string `json:"summary"`
	CoverImage string `json:"cover_image"`
	Author     string `json:"author"`
	ReadCount  int    `json:"read_count"`
	IsCrisis   int    `json:"is_crisis"`
	Tags       string `json:"tags"`
	CreatedAt  string `json:"created_at"`
}

// PsychArticleDetail 心理科普文章详情
type PsychArticleDetail struct {
	ID         int64  `json:"id"`
	ArticleID  string `json:"article_id"`
	Title      string `json:"title"`
	Category   string `json:"category"`
	Summary    string `json:"summary"`
	Content    string `json:"content"`
	CoverImage string `json:"cover_image"`
	Author     string `json:"author"`
	ReadCount  int    `json:"read_count"`
	IsCrisis   int    `json:"is_crisis"`
	Tags       string `json:"tags"`
	Status     string `json:"status"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

// CrisisHotline 危机热线
type CrisisHotline struct {
	ID          int64  `json:"id"`
	HotlineID   string `json:"hotline_id"`
	Name        string `json:"name"`
	Phone       string `json:"phone"`
	ServiceTime string `json:"service_time"`
	Description string `json:"description"`
	Level       int    `json:"level"`
	CreatedAt   string `json:"created_at"`
}

// MoodDiaryItem 情绪日记条目
type MoodDiaryItem struct {
	ID              int64   `json:"id"`
	DiaryID         string  `json:"diary_id"`
	Date            string  `json:"date"`
	MoodScore       int     `json:"mood_score"`
	MoodTags        string  `json:"mood_tags"`
	Events          string  `json:"events"`
	DiaryContent    string  `json:"diary_content"`
	SleepHours      float64 `json:"sleep_hours"`
	ExerciseMinutes int     `json:"exercise_minutes"`
	SocialLevel     int     `json:"social_level"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}
