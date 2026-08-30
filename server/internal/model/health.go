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
