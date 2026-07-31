package model

// CampusStep 校园报到步骤（动态管理，替代前端硬编码常量）
type CampusStep struct {
	ID          int64   `json:"id"`
	CampusID    string  `json:"campus_id"` // huifeng | langya
	StepOrder   int     `json:"step_order"`
	Title       string  `json:"title"`
	Location    string  `json:"location"`
	Lat         float64 `json:"lat"`
	Lng         float64 `json:"lng"`
	Duration    string  `json:"duration"`
	Task        string  `json:"task"`
	Materials   string  `json:"materials"`
	Contact     string  `json:"contact"`
	Note        string  `json:"note"`
	IconName    string  `json:"icon_name"`
	Status      string  `json:"status"` // draft | pending_review | published
	CreatedBy   *int64  `json:"created_by,omitempty"`
	ReviewedBy  *int64  `json:"reviewed_by,omitempty"`
	PublishedAt string  `json:"published_at,omitempty"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

// CampusStepRequest 创建/更新步骤请求体
type CampusStepRequest struct {
	CampusID  string  `json:"campus_id"  binding:"required,oneof=huifeng langya"`
	StepOrder int     `json:"step_order" binding:"required,min=1"`
	Title     string  `json:"title"      binding:"required"`
	Location  string  `json:"location"   binding:"required"`
	Lat       float64 `json:"lat"        binding:"required"`
	Lng       float64 `json:"lng"        binding:"required"`
	Duration  string  `json:"duration"`
	Task      string  `json:"task"`
	Materials string  `json:"materials"`
	Contact   string  `json:"contact"`
	Note      string  `json:"note"`
	IconName  string  `json:"icon_name"`
}
