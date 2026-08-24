package repository

import (
	"database/sql"
	"fmt"
	"strings"
)

// Homework 教师作业信息发布（P2 轻量版，迁移 095）
// 蔚小芯侧重教育非教学：本表仅存「信息发布 + 归属」，无作业内容/附件/提交表。
// 归属强约束：course_id 必须对应该教师 approved 授课关系（teacher_courses，service 层校验）。
// status 状态机：active/published/archived（轻量信息态，非审核流）。
type Homework struct {
	ID          int64  `json:"id"`
	TeacherID   int64  `json:"teacher_id"`
	TeacherName string `json:"teacher_name"` // 冗余展示（join users），非权威
	CourseID    string `json:"course_id"`
	CourseName  string `json:"course_name"` // 冗余展示名，非权威
	Semester    string `json:"semester"`
	Title       string `json:"title"`
	Description string `json:"description"`
	PublishAt   string `json:"publish_at"`
	DueAt       string `json:"due_at"`
	Status      string `json:"status"` // active/published/archived
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// Homework 状态常量（轻量信息态）
const (
	HomeworkStatusActive    = "active"
	HomeworkStatusPublished = "published"
	HomeworkStatusArchived  = "archived"
)

// HomeworkRepo 教师作业信息发布访问层（P2）
type HomeworkRepo struct {
	db *sql.DB
}

func NewHomeworkRepo(db *sql.DB) *HomeworkRepo {
	return &HomeworkRepo{db: db}
}

// Publish 发布作业信息（幂等：UNIQUE(teacher_id,course_id,semester,title) 同组合仅一条）。
// 幂等语义：同组合已存在 → 返回已存在 id + existed=true（不覆盖用户内容，避免误改他人字段）。
// 归属强校验（teacher_id/course_id/semester 是否已 approved）由 service 层在写库前完成。
func (r *HomeworkRepo) Publish(h *Homework) (int64, bool, error) {
	if h == nil || h.TeacherID <= 0 {
		return 0, false, fmt.Errorf("教师不能为空")
	}
	if h.CourseID == "" || h.Semester == "" || h.Title == "" {
		return 0, false, fmt.Errorf("课程、学期、标题不能为空")
	}
	h.CourseID = normalizeCourseID(h.CourseID)
	h.Semester = normalizeSemester(h.Semester)

	tx, err := r.db.Begin()
	if err != nil {
		return 0, false, err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// 先查后插（事务，方言兼容，不引入 ON CONFLICT）
	var existingID int64
	err = tx.QueryRow(
		`SELECT id FROM homework WHERE teacher_id=? AND course_id=? AND semester=? AND title=?`,
		h.TeacherID, h.CourseID, h.Semester, strings.TrimSpace(h.Title)).Scan(&existingID)
	switch {
	case err == sql.ErrNoRows:
		status := h.Status
		if status == "" {
			status = HomeworkStatusActive
		}
		res, err2 := tx.Exec(
			`INSERT INTO homework (teacher_id, course_id, course_name, semester, title, description, publish_at, due_at, status)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			h.TeacherID, h.CourseID, h.CourseName, h.Semester, strings.TrimSpace(h.Title),
			h.Description, h.PublishAt, h.DueAt, status,
		)
		if err2 != nil {
			err = err2
			return 0, false, err
		}
		id, _ := res.LastInsertId()
		if err2 := tx.Commit(); err2 != nil {
			err = err2
			return 0, false, err
		}
		return id, false, nil
	case err != nil:
		return 0, false, err
	}
	// 已存在：不覆盖内容，返回 existed=true（诚实幂等）
	if err := tx.Commit(); err != nil {
		return 0, false, err
	}
	return existingID, true, nil
}

// GetOwnHomework 查询本人一条作业（读取边界=teacher_id，编辑/下架前归属校验用）。
// 已归档(archived)也返回，便于本人复核历史；返回 (hw,nil) 或 (nil,nil)=不存在。
func (r *HomeworkRepo) GetOwnHomework(id, teacherID int64) (*Homework, error) {
	q := `SELECT h.id, h.teacher_id, COALESCE(u.display_name, u.username, ''),
	             h.course_id, h.course_name, h.semester, h.title, h.description,
	             h.publish_at, h.due_at, h.status, h.created_at, h.updated_at
	      FROM homework h LEFT JOIN users u ON u.id = h.teacher_id
	      WHERE h.id=? AND h.teacher_id=?`
	row := r.db.QueryRow(q, id, teacherID)
	var h Homework
	var publishAt, dueAt sql.NullString
	err := row.Scan(&h.ID, &h.TeacherID, &h.TeacherName, &h.CourseID, &h.CourseName,
		&h.Semester, &h.Title, &h.Description, &publishAt, &dueAt,
		&h.Status, &h.CreatedAt, &h.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	h.PublishAt = publishAt.String
	h.DueAt = dueAt.String
	return &h, nil
}

// Update 编辑作业信息（仅本人已由 GetOwnHomework 保证；course_id/semester 归属不因编辑改变）。
// 允许更新 title/description/publish_at/due_at/status。
func (r *HomeworkRepo) Update(id int64, h *Homework) error {
	if h == nil || id <= 0 {
		return fmt.Errorf("参数错误")
	}
	status := h.Status
	if status == "" {
		status = HomeworkStatusActive
	}
	_, err := r.db.Exec(
		`UPDATE homework SET course_name=?, title=?, description=?, publish_at=?, due_at=?, status=?, updated_at=datetime('now','localtime') WHERE id=?`,
		h.CourseName, strings.TrimSpace(h.Title), h.Description, h.PublishAt, h.DueAt, status, id,
	)
	return err
}

// Archive 下架作业（软删：置 archived，审计可溯，不物理删除）。
// 仅本人作业可下架（GetOwnHomework 已保证归属）。
func (r *HomeworkRepo) Archive(id int64) error {
	_, err := r.db.Exec(
		`UPDATE homework SET status=?, updated_at=datetime('now','localtime') WHERE id=?`,
		HomeworkStatusArchived, id,
	)
	return err
}

// ListByTeacher 教师本人的作业清单（按 teacher_id，最新在前）。
func (r *HomeworkRepo) ListByTeacher(teacherID int64) ([]Homework, error) {
	q := `SELECT h.id, h.teacher_id, COALESCE(u.display_name, u.username, ''),
	             h.course_id, h.course_name, h.semester, h.title, h.description,
	             h.publish_at, h.due_at, h.status, h.created_at, h.updated_at
	      FROM homework h LEFT JOIN users u ON u.id = h.teacher_id
	      WHERE h.teacher_id=?
	      ORDER BY h.id DESC`
	rows, err := r.db.Query(q, teacherID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Homework
	for rows.Next() {
		var h Homework
		var publishAt, dueAt sql.NullString
		if err := rows.Scan(&h.ID, &h.TeacherID, &h.TeacherName, &h.CourseID, &h.CourseName,
			&h.Semester, &h.Title, &h.Description, &publishAt, &dueAt,
			&h.Status, &h.CreatedAt, &h.UpdatedAt); err != nil {
			return nil, err
		}
		h.PublishAt = publishAt.String
		h.DueAt = dueAt.String
		list = append(list, h)
	}
	return list, rows.Err()
}

// CourseGradeStats 课程成绩只读聚合结果（基于 student_grades 真实数据）。
// 按课程维度（course_id+semester+grade_type='final'）聚合：人数/均分/及格率/四档分布。
// 0 行返回 total=0 + 空分布（不补造，诚实空）。仅 approved 授课课程可查（service 层白名单校验）。
type CourseGradeStats struct {
	CourseID     string         `json:"course_id"`
	CourseName   string         `json:"course_name"`
	Semester     string         `json:"semester"`
	Total        int            `json:"total"`         // 录入人数
	AvgScore     float64        `json:"avg_score"`     // 均分（0 行=0）
	PassRate     float64        `json:"pass_rate"`     // 及格率 0-1（0 行=0）
	PassedCount  int            `json:"passed_count"`  // 及格人数
	Levels       map[string]int `json:"levels"`        // 优秀/良好/及格/不及格 四档人数
	NotAvailable bool           `json:"not_available"` // 0 行诚实标记
}

// GradeStatsByCourse 按课程聚合成绩统计（只读，无任何写入/聚合幻数）。
// 含 student join（取学生名）、passed 计数、gradeLevelOf 四档分布。
// 仅统计 grade_type='final' 期末成绩，对齐成绩导入幂等口径。
func (r *HomeworkRepo) GradeStatsByCourse(courseID, semester string) (*CourseGradeStats, error) {
	courseID = normalizeCourseID(courseID)

	// 聚合基础：count / 均分 / 及格数
	// 含 student join：仅统计 role='student' 的真实学生成绩行（对称成绩导入口径，防校非学生数据污染统计）。
	var total int
	var avgScore float64
	var passedCount int
	err := r.db.QueryRow(
		`SELECT COUNT(*),
		        COALESCE(AVG(g.score),0),
		        COALESCE(SUM(CASE WHEN g.passed=1 THEN 1 ELSE 0 END),0)
		 FROM student_grades g
		 LEFT JOIN users u ON CAST(g.user_id AS INTEGER)=u.id
		 WHERE g.course_id=? AND g.semester=? AND g.grade_type='final' AND u.role='student'`,
		courseID, semester).Scan(&total, &avgScore, &passedCount)
	if err != nil {
		return nil, err
	}

	stats := &CourseGradeStats{
		CourseID:    courseID,
		Semester:    semester,
		Total:       total,
		AvgScore:    avgScore,
		PassedCount: passedCount,
		Levels: map[string]int{
			"优秀":  0,
			"良好":  0,
			"及格":  0,
			"不及格": 0,
		},
	}
	if total == 0 {
		stats.AvgScore = 0
		stats.PassRate = 0
		stats.NotAvailable = true
		return stats, nil
	}
	stats.PassRate = float64(passedCount) / float64(total)

	// 四档分布：每条真实分数对称复用 gradeLevelOf 逻辑（只读，SQLite 端聚合行级）
	rows, err := r.db.Query(
		`SELECT g.score, COALESCE(u.display_name, '') AS name, g.course_name
		 FROM student_grades g LEFT JOIN users u ON CAST(g.user_id AS INTEGER)=u.id
		 WHERE g.course_id=? AND g.semester=? AND g.grade_type='final'`,
		courseID, semester)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var score float64
		var name, courseName string
		if err := rows.Scan(&score, &name, &courseName); err != nil {
			return nil, err
		}
		if stats.CourseName == "" && courseName != "" {
			stats.CourseName = courseName
		}
		stats.Levels[gradeLevelOf(score)]++
	}
	return stats, rows.Err()
}
