package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// TeacherCourse 教师授课关系申报（R3，迁移 094）
// 状态机：pending/approved/rejected（对称 graduation_outcome）。
// approved 唯一来源为教辅真实审核操作（ReviewTeacherCourse），绝不脚本批量置位。
type TeacherCourse struct {
	ID           int64  `json:"id"`
	TeacherID    int64  `json:"teacher_id"`
	TeacherName  string `json:"teacher_name"`  // 冗余展示（join users），非权威
	CourseID     string `json:"course_id"`
	CourseName   string `json:"course_name"` // 冗余展示名，非权威
	Semester     string `json:"semester"`
	Status       string `json:"status"` // pending/approved/rejected
	CreatedBy    int64  `json:"created_by"`
	ReviewedBy   int64  `json:"reviewed_by"`
	ReviewedName string `json:"reviewed_name"`
	ReviewNote   string `json:"review_note"`
	ReviewedAt   string `json:"reviewed_at"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

// TeacherCourse 状态常量（对齐 graduation_outcome）
const (
	CourseStatusPending  = "pending"
	CourseStatusApproved = "approved"
	CourseStatusRejected = "rejected"
)

// TeacherCourseRepo 教师授课关系申报访问层（R3）
type TeacherCourseRepo struct {
	db *sql.DB
}

func NewTeacherCourseRepo(db *sql.DB) *TeacherCourseRepo {
	return &TeacherCourseRepo{db: db}
}

// normalizeCourseID 写入前规范化 course_id：trim + 统一大小写，避免 CS101/cs101 双记录。
func normalizeCourseID(s string) string {
	return strings.TrimSpace(strings.ToUpper(s))
}

// normalizeSemester 写入前规范化 semester：trim（保留语义不变）。
func normalizeSemester(s string) string {
	return strings.TrimSpace(s)
}

// SubmitTeacherCourse 申报授课关系（幂等，见状态机语义）：
//   - 同 (teacher_id, course_id, semester) 已有 pending → 报错「待审核中，请勿重复申报」；
//   - 已有 approved → 返回「已通过」，不重复插入；
//   - 已有 rejected → 允许重新申报，置回 pending（走重审）；
//   - 无 → 新增 pending。
//   - 仅 teacher 可申报；申报人=本人（created_by=teacher_id），杜绝代他人申报。
func (r *TeacherCourseRepo) SubmitTeacherCourse(tc *TeacherCourse) (int64, string, error) {
	if tc == nil || tc.TeacherID <= 0 {
		return 0, "", fmt.Errorf("教师不能为空")
	}
	if tc.CourseID == "" || tc.Semester == "" {
		return 0, "", fmt.Errorf("课程和学期不能为空")
	}
	tc.CourseID = normalizeCourseID(tc.CourseID)
	tc.Semester = normalizeSemester(tc.Semester)

	// 先查后插（事务），符合方言兼容（SQLite/MySQL 均可用，不引入 ON CONFLICT）。
	tx, err := r.db.Begin()
	if err != nil {
		return 0, "", err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	var existingID int64
	var existingStatus string
	err = tx.QueryRow(
		`SELECT id, status FROM teacher_courses WHERE teacher_id=? AND course_id=? AND semester=?`,
		tc.TeacherID, tc.CourseID, tc.Semester).Scan(&existingID, &existingStatus)
	switch {
	case err == sql.ErrNoRows:
		// 无记录：新增 pending
		res, err2 := tx.Exec(
			`INSERT INTO teacher_courses (teacher_id, course_id, course_name, semester, status, created_by, reviewed_by, review_note)
			 VALUES (?, ?, ?, ?, 'pending', ?, 0, '')`,
			tc.TeacherID, tc.CourseID, tc.CourseName, tc.Semester, tc.CreatedBy,
		)
		if err2 != nil {
			err = err2
			return 0, "", err
		}
		id, _ := res.LastInsertId()
		if err2 := tx.Commit(); err2 != nil {
			err = err2
			return 0, "", err
		}
		return id, CourseStatusPending, nil
	case err != nil:
		return 0, "", err
	}

	// 已有记录：按状态机处理
	switch existingStatus {
	case CourseStatusPending:
		err = fmt.Errorf("该课程授课申报待审核中，请勿重复申报")
		return 0, "", err
	case CourseStatusApproved:
		err = tx.Commit()
		if err != nil {
			return 0, "", err
		}
		return existingID, CourseStatusApproved, nil
	case CourseStatusRejected:
		// 驳回后可重新申报：置回 pending 并清空审核留痕
		if _, err2 := tx.Exec(
			`UPDATE teacher_courses SET status='pending', course_name=?, reviewed_by=0, reviewed_name='', review_note='', reviewed_at=NULL, updated_at=datetime('now','localtime') WHERE id=?`,
			tc.CourseName, existingID); err2 != nil {
			err = err2
			return 0, "", err
		}
		if err2 := tx.Commit(); err2 != nil {
			err = err2
			return 0, "", err
		}
		return existingID, CourseStatusPending, nil
	default:
		err = fmt.Errorf("未知申报状态: %s", existingStatus)
		return 0, "", err
	}
}

// ListTeacherCourses 查询申报（按教师/状态过滤）。
// teacherID>0 → 某教师本人申报；status!="" → 按状态过滤。
func (r *TeacherCourseRepo) ListTeacherCourses(teacherID int64, status string, limit int) ([]TeacherCourse, error) {
	q := `SELECT tc.id, tc.teacher_id, COALESCE(u.display_name, u.username, ''),
	             tc.course_id, tc.course_name, tc.semester, tc.status,
	             tc.created_by, tc.reviewed_by, tc.reviewed_name, tc.review_note, tc.reviewed_at,
	             tc.created_at, tc.updated_at
	      FROM teacher_courses tc LEFT JOIN users u ON u.id = tc.teacher_id WHERE 1=1`
	args := []interface{}{}
	if teacherID > 0 {
		q += ` AND tc.teacher_id = ?`
		args = append(args, teacherID)
	}
	if status != "" {
		q += ` AND tc.status = ?`
		args = append(args, status)
	}
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	q += ` ORDER BY tc.id DESC LIMIT ?`
	args = append(args, limit)

	rows, err := r.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []TeacherCourse
	for rows.Next() {
		var tc TeacherCourse
		if err := rows.Scan(&tc.ID, &tc.TeacherID, &tc.TeacherName, &tc.CourseID, &tc.CourseName,
			&tc.Semester, &tc.Status, &tc.CreatedBy, &tc.ReviewedBy, &tc.ReviewedName,
			&tc.ReviewNote, &tc.ReviewedAt, &tc.CreatedAt, &tc.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, tc)
	}
	return list, rows.Err()
}

// ListPendingTeacherCourses 待审核申报（教辅审核列表）
func (r *TeacherCourseRepo) ListPendingTeacherCourses(limit int) ([]TeacherCourse, error) {
	return r.ListTeacherCourses(0, CourseStatusPending, limit)
}

// ReviewTeacherCourse 审核一条申报（pending→approved/rejected）。
// 仅 approved/rejected 有效；落 reviewed_by/reviewed_name/reviewed_at/review_note（审核即真实操作留痕）。
// 拒绝重复审核：仅允许审核 pending 状态记录。
func (r *TeacherCourseRepo) ReviewTeacherCourse(id, reviewerID int64, reviewerName, status, note string) error {
	if status != CourseStatusApproved && status != CourseStatusRejected {
		return fmt.Errorf("无效审核状态: %s", status)
	}
	// 仅允许审核 pending 状态（防重复审核/审核已定案记录）
	var cur string
	err := r.db.QueryRow(`SELECT status FROM teacher_courses WHERE id=?`, id).Scan(&cur)
	if err == sql.ErrNoRows {
		return fmt.Errorf("申报记录不存在")
	}
	if err != nil {
		return err
	}
	if cur != CourseStatusPending {
		return fmt.Errorf("仅待审核(pending)申报可审核，当前状态=%s", cur)
	}
	_, err = r.db.Exec(
		`UPDATE teacher_courses SET status=?, reviewed_by=?, reviewed_name=?, review_note=?, reviewed_at=?, updated_at=datetime('now','localtime') WHERE id=?`,
		status, reviewerID, reviewerName, note, time.Now().Format(time.RFC3339), id,
	)
	return err
}

// CountPendingTeacherCourses 待审核申报条数（教辅入口角标）
func (r *TeacherCourseRepo) CountPendingTeacherCourses() (int, error) {
	var n int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM teacher_courses WHERE status='pending'`).Scan(&n)
	return n, err
}

// CourseExists 申报入参校验（R3 补强 M）：course_id 是否存在于 courses 主数据表。
// courses.course_id 为 TEXT UNIQUE 稳定键，是 teacher_courses.course_id 的权威口径（对外键语义）。
// 校验只读 courses 目录，不触碰申报/审核状态机；courses 目录为空时任何申报都会被拒绝，
// 表明「courses 主数据未就绪」，申报入口应提示教师先核对课程ID（诚实口径，不误放虚构课程）。
func (r *TeacherCourseRepo) CourseExists(courseID string) (bool, error) {
	var n int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM courses WHERE course_id = ?`, normalizeCourseID(courseID)).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// GetTeacherCourseStatus 成绩强校验判据（P1）：返回是否存在 + 状态。
// 返回 (exists, status, err)：无申报 exists=false,status="".
func (r *TeacherCourseRepo) GetTeacherCourseStatus(teacherID int64, courseID, semester string) (bool, string, error) {
	var status string
	err := r.db.QueryRow(
		`SELECT status FROM teacher_courses WHERE teacher_id=? AND course_id=? AND semester=?`,
		teacherID, normalizeCourseID(courseID), normalizeSemester(semester)).Scan(&status)
	if err == sql.ErrNoRows {
		return false, "", nil
	}
	if err != nil {
		return false, "", err
	}
	return true, status, nil
}
