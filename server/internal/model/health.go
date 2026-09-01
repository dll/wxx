// Package model 学生身体健康（P4-d：从 handler 下沉为共享 DTO）。
package model

// HealthBasicInfo 身体基本信息（每用户一行）
type HealthBasicInfo struct {
	ID               int64   `json:"id"`
	HeightCm         float64 `json:"height_cm"`
	WeightKg         float64 `json:"weight_kg"`
	BloodType        string  `json:"blood_type"`
	VisionLeft       string  `json:"vision_left"`
	VisionRight      string  `json:"vision_right"`
	Allergies        string  `json:"allergies"`
	PastIllness      string  `json:"past_illness"`
	FamilyHistory    string  `json:"family_history"`
	EmergencyContact string  `json:"emergency_contact"`
	EmergencyPhone   string  `json:"emergency_phone"`
	UpdatedAt        string  `json:"updated_at"`
}

// HealthCheckup 体检记录
type HealthCheckup struct {
	ID          int64    `json:"id"`
	CheckupDate string   `json:"checkup_date"`
	Hospital    string   `json:"hospital"`
	Conclusion  string   `json:"conclusion"`
	Details     string   `json:"details"`
	Attachments []string `json:"attachments"`
	CreatedAt   string   `json:"created_at"`
}

// HealthRecord 病历记录
type HealthRecord struct {
	ID          int64    `json:"id"`
	RecordDate  string   `json:"record_date"`
	Hospital    string   `json:"hospital"`
	Department  string   `json:"department"`
	Diagnosis   string   `json:"diagnosis"`
	Treatment   string   `json:"treatment"`
	Attachments []string `json:"attachments"`
	CreatedAt   string   `json:"created_at"`
}

// HealthDailyItem 日常健康记录（趋势图）
type HealthDailyItem struct {
	ID         int64   `json:"id"`
	RecordDate string  `json:"record_date"`
	HeightCm   float64 `json:"height_cm"`
	WeightKg   float64 `json:"weight_kg"`
	Systolic   int     `json:"systolic"`
	Diastolic  int     `json:"diastolic"`
	HeartRate  int     `json:"heart_rate"`
	Note       string  `json:"note"`
	CreatedAt  string  `json:"created_at"`
}

// HealthActivityItem 健康活动（含关注/报名统计与当前用户状态）
type HealthActivityItem struct {
	ActivityID     string `json:"activity_id"`
	Title          string `json:"title"`
	Category       string `json:"category"`
	Description    string `json:"description"`
	StartAt        string `json:"start_at"`
	EndAt          string `json:"end_at"`
	Venue          string `json:"venue"`
	Organizer      string `json:"organizer"`
	Capacity       int    `json:"capacity"`
	SignupDeadline string `json:"signup_deadline"`
	Status         string `json:"status"`
	CreatorRole    string `json:"creator_role"`
	// 关注/报名统计与当前用户状态
	FavoriteCount int  `json:"favorite_count"`
	SignupCount   int  `json:"signup_count"`
	IsFavorite    bool `json:"is_favorite"`
	IsSignup      bool `json:"is_signup"`
}

// ActivityReviewRow 活动复盘单行（报名/到场原始数）
type ActivityReviewRow struct {
	ActivityID  string  `json:"activity_id"`
	Title       string  `json:"title"`
	Category    string  `json:"category"`
	Venue       string  `json:"venue"`
	Organizer   string  `json:"organizer"`
	Status      string  `json:"status"`
	SignupCount int     `json:"signup_count"`
	AttendCount int     `json:"attend_count"`
	AttendRate  float64 `json:"attend_rate"`
}

// ActivitySignup 活动报名/到场名单条目
type ActivitySignup struct {
	UserID    int    `json:"user_id"`
	Name      string `json:"name"` // display_name 优先，回退 username
	Username  string `json:"username"`
	Attended  bool   `json:"attended"`
	CreatedAt string `json:"created_at"`
}
