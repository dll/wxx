// Package model 学业学习模块（P4-d：从 handler 下沉为共享 DTO）。
package model

// Course 课程列表项
type Course struct {
	ID          int64   `json:"id"`
	CourseID    string  `json:"course_id"`
	CourseCode  string  `json:"course_code"`
	CourseName  string  `json:"course_name"`
	Credit      float64 `json:"credit"`
	Hours       int     `json:"hours"`
	Category    string  `json:"category"`
	Department  string  `json:"department"`
	Teacher     string  `json:"teacher"`
	Description string  `json:"description"`
	Semester    string  `json:"semester"`
	CreatedAt   string  `json:"created_at"`
}

// CourseDetail 课程详情
type CourseDetail struct {
	ID            int64   `json:"id"`
	CourseID      string  `json:"course_id"`
	CourseCode    string  `json:"course_code"`
	CourseName    string  `json:"course_name"`
	Credit        float64 `json:"credit"`
	Hours         int     `json:"hours"`
	Category      string  `json:"category"`
	Department    string  `json:"department"`
	Teacher       string  `json:"teacher"`
	Description   string  `json:"description"`
	Syllabus      string  `json:"syllabus"`
	Prerequisites string  `json:"prerequisites"`
	Textbook      string  `json:"textbook"`
	References    string  `json:"references"`
	Semester      string  `json:"semester"`
	Status        string  `json:"status"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

// GradeItem 成绩项
type GradeItem struct {
	ID            int64   `json:"id"`
	CourseID      string  `json:"course_id"`
	CourseName    string  `json:"course_name"`
	Semester      string  `json:"semester"`
	GradeType     string  `json:"grade_type"`
	Score         float64 `json:"score"`
	GPA           float64 `json:"gpa"`
	Rank          int     `json:"rank"`
	GradeLevel    string  `json:"grade_level"`
	Passed        int     `json:"passed"`
	CreditsEarned float64 `json:"credits_earned"`
	CreatedAt     string  `json:"created_at"`
}

// LearningResource 学习资源
type LearningResource struct {
	ID            int64  `json:"id"`
	ResourceID    string `json:"resource_id"`
	CourseID      string `json:"course_id"`
	CourseName    string `json:"course_name"`
	Title         string `json:"title"`
	ResourceType  string `json:"resource_type"`
	Chapter       string `json:"chapter"`
	FileURL       string `json:"file_url"`
	Author        string `json:"author"`
	DownloadCount int    `json:"download_count"`
	ViewCount     int    `json:"view_count"`
	Tags          string `json:"tags"`
	CreatedAt     string `json:"created_at"`
}

// ExamSchedule 考试安排
type ExamSchedule struct {
	ID         int64  `json:"id"`
	ExamID     string `json:"exam_id"`
	CourseID   string `json:"course_id"`
	CourseName string `json:"course_name"`
	ExamType   string `json:"exam_type"`
	Date       string `json:"date"`
	TimeStart  string `json:"time_start"`
	TimeEnd    string `json:"time_end"`
	Location   string `json:"location"`
	Seat       string `json:"seat"`
	Semester   string `json:"semester"`
	CreatedAt  string `json:"created_at"`
}

// GradeSummary 成绩统计
type GradeSummary struct {
	TotalGPA        float64 `json:"total_gpa"`
	TotalCredits    float64 `json:"total_credits"`
	EarnedCredits   float64 `json:"earned_credits"`
	TotalCourses    int     `json:"total_courses"`
	PassedCourses   int     `json:"passed_courses"`
	FailedCourses   int     `json:"failed_courses"`
	AverageScore    float64 `json:"average_score"`
	CurrentSemester string  `json:"current_semester"`
	SemesterGPA     float64 `json:"semester_gpa"`
	SemesterCredits float64 `json:"semester_credits"`
}
