// Package model 就业指导（P4-d：从 handler 下沉为共享 DTO）。
package model

// CareerPolicy 就业政策列表项
type CareerPolicy struct {
	ID          int64   `json:"id"`
	PolicyID    string  `json:"policy_id"`
	Title       string  `json:"title"`
	Category    string  `json:"category"`
	Level       string  `json:"level"`
	Source      string  `json:"source"`
	Summary     string  `json:"summary"`
	PublishDate *string `json:"publish_date"`
	Tags        string  `json:"tags"`
	ViewCount   int     `json:"view_count"`
	CreatedAt   string  `json:"created_at"`
}

// CareerPolicyDetail 就业政策详情
type CareerPolicyDetail struct {
	ID            int64   `json:"id"`
	PolicyID      string  `json:"policy_id"`
	Title         string  `json:"title"`
	Category      string  `json:"category"`
	Level         string  `json:"level"`
	Source        string  `json:"source"`
	Content       string  `json:"content"`
	Summary       string  `json:"summary"`
	PublishDate   *string `json:"publish_date"`
	EffectiveDate *string `json:"effective_date"`
	ExpiryDate    *string `json:"expiry_date"`
	Tags          string  `json:"tags"`
	Status        string  `json:"status"`
	ViewCount     int     `json:"view_count"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

// JobPosting 招聘信息列表项
type JobPosting struct {
	ID           int64   `json:"id"`
	JobID        string  `json:"job_id"`
	CompanyName  string  `json:"company_name"`
	CompanyLogo  string  `json:"company_logo"`
	PositionName string  `json:"position_name"`
	PositionType string  `json:"position_type"`
	Industry     string  `json:"industry"`
	SalaryMin    int     `json:"salary_min"`
	SalaryMax    int     `json:"salary_max"`
	SalaryUnit   string  `json:"salary_unit"`
	Location     string  `json:"location"`
	Education    string  `json:"education"`
	Deadline     *string `json:"deadline"`
	ViewCount    int     `json:"view_count"`
	ApplyCount   int     `json:"apply_count"`
	CreatedAt    string  `json:"created_at"`
}

// JobPostingDetail 职位详情
type JobPostingDetail struct {
	ID               int64   `json:"id"`
	JobID            string  `json:"job_id"`
	CompanyName      string  `json:"company_name"`
	CompanyLogo      string  `json:"company_logo"`
	CompanyIntro     string  `json:"company_intro"`
	PositionName     string  `json:"position_name"`
	PositionType     string  `json:"position_type"`
	Industry         string  `json:"industry"`
	SalaryMin        int     `json:"salary_min"`
	SalaryMax        int     `json:"salary_max"`
	SalaryUnit       string  `json:"salary_unit"`
	Location         string  `json:"location"`
	Education        string  `json:"education"`
	MajorRequirement string  `json:"major_requirement"`
	Description      string  `json:"description"`
	Requirement      string  `json:"requirement"`
	Benefits         string  `json:"benefits"`
	ApplicationURL   string  `json:"application_url"`
	Deadline         *string `json:"deadline"`
	Source           string  `json:"source"`
	Status           string  `json:"status"`
	ViewCount        int     `json:"view_count"`
	ApplyCount       int     `json:"apply_count"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
}

// InfoSession 宣讲会
type InfoSession struct {
	ID              int64  `json:"id"`
	SessionID       string `json:"session_id"`
	CompanyName     string `json:"company_name"`
	CompanyLogo     string `json:"company_logo"`
	Title           string `json:"title"`
	Date            string `json:"date"`
	TimeStart       string `json:"time_start"`
	TimeEnd         string `json:"time_end"`
	Location        string `json:"location"`
	Campus          string `json:"campus"`
	Description     string `json:"description"`
	RegistrationURL string `json:"registration_url"`
	Capacity        int    `json:"capacity"`
	RegisteredCount int    `json:"registered_count"`
	CreatedAt       string `json:"created_at"`
}

// InterviewQuestion 面试题
type InterviewQuestion struct {
	ID         int64  `json:"id"`
	QuestionID string `json:"question_id"`
	Category   string `json:"category"`
	Industry   string `json:"industry"`
	Position   string `json:"position"`
	Question   string `json:"question"`
	AnswerHint string `json:"answer_hint"`
	Keywords   string `json:"keywords"`
	Difficulty int    `json:"difficulty"`
	CreatedAt  string `json:"created_at"`
}
