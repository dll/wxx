// Package model 学习计划与校历（P4-d：从 handler 下沉为共享 DTO）。
package model

// AcademicCalendar 学期校历
type AcademicCalendar struct {
	ID           int64  `json:"id"`
	AcademicYear int    `json:"academic_year"`
	Semester     int    `json:"semester"`
	SemesterCode string `json:"semester_code"`
	SemesterName string `json:"semester_name"`
	StartDate    string `json:"start_date"`
	EndDate      string `json:"end_date"`
	RegisterDate string `json:"register_date,omitempty"`
	TotalWeeks   int    `json:"total_weeks"`
	WeekStartDay string `json:"week_start_day"`
	Status       string `json:"status"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

// CalendarEvent 校历事件
type CalendarEvent struct {
	ID             int64  `json:"id"`
	SemesterCode   string `json:"semester_code"`
	EventName      string `json:"event_name"`
	EventType      string `json:"event_type"`
	StartDate      string `json:"start_date"`
	EndDate        string `json:"end_date,omitempty"`
	WeekNo         int    `json:"week_no,omitempty"`
	AffectsClasses int    `json:"affects_classes"`
	Description    string `json:"description,omitempty"`
	CreatedAt      string `json:"created_at"`
}

// CourseScheduleItem 课表项
type CourseScheduleItem struct {
	ID           int64  `json:"id"`
	UserID       int64  `json:"user_id"`
	CourseID     string `json:"course_id"`
	CourseName   string `json:"course_name"`
	SemesterCode string `json:"semester_code"`
	Weekday      int    `json:"weekday"`
	StartPeriod  int    `json:"start_period"`
	EndPeriod    int    `json:"end_period"`
	WeeksPattern string `json:"weeks_pattern"`
	Location     string `json:"location,omitempty"`
	Teacher      string `json:"teacher,omitempty"`
	Color        string `json:"color"`
	CreatedAt    string `json:"created_at"`
}

// StudyPlan 学习计划
type StudyPlan struct {
	ID           int64    `json:"id"`
	UserID       int64    `json:"user_id"`
	Title        string   `json:"title"`
	PlanType     string   `json:"plan_type"`
	SemesterCode string   `json:"semester_code,omitempty"`
	StartDate    string   `json:"start_date"`
	EndDate      string   `json:"end_date"`
	Goals        []string `json:"goals"`
	Progress     float64  `json:"progress"`
	AIGenerated  bool     `json:"ai_generated"`
	Status       string   `json:"status"`
	LinkedPlanID *int64   `json:"linked_plan_id,omitempty"`
	CreatedAt    string   `json:"created_at"`
	UpdatedAt    string   `json:"updated_at"`
	// 任务统计（列表场景）
	TaskTotal   int              `json:"task_total,omitempty"`
	TaskDone    int              `json:"task_done,omitempty"`
	TaskPending int              `json:"task_pending,omitempty"`
	Tasks       []*StudyPlanTask `json:"tasks,omitempty"`
}

// StudyPlanTask 学习计划任务
type StudyPlanTask struct {
	ID                int64  `json:"id"`
	PlanID            int64  `json:"plan_id"`
	CourseID          string `json:"course_id,omitempty"`
	CourseName        string `json:"course_name,omitempty"`
	Title             string `json:"title"`
	Description       string `json:"description,omitempty"`
	ScheduledDate     string `json:"scheduled_date,omitempty"`
	ScheduledDuration int    `json:"scheduled_duration"`
	ActualDuration    int    `json:"actual_duration"`
	Status            string `json:"status"`
	Evidence          string `json:"evidence,omitempty"`
	Reflection        string `json:"reflection,omitempty"`
	SortOrder         int    `json:"sort_order"`
	CreatedAt         string `json:"created_at"`
}

// PlanOverviewItem 计划概览项（用于多Tab首页）
type PlanOverviewItem struct {
	PlanType  string  `json:"plan_type"`
	PlanCount int     `json:"plan_count"`
	Progress  float64 `json:"progress"` // 平均进度
	TaskTotal int     `json:"task_total"`
	TaskDone  int     `json:"task_done"`
}
