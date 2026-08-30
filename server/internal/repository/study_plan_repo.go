// Package repository 学习计划与校历仓库（P4-d：从 study_plan 系列三个 handler 下沉的裸 SQL）。
package repository

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/dll/wxx/server/internal/model"
)

// StudyPlanRepo 学习计划数据访问层。
type StudyPlanRepo struct {
	db *sql.DB
}

// ErrCalendarNotFound 学期校历不存在。
var ErrCalendarNotFound = errors.New("学期校历不存在")

// ErrPlanNotFound 学习计划不存在（或不属于该用户）。
var ErrPlanNotFound = errors.New("学习计划不存在")

// NewStudyPlanRepo 创建学习计划仓库。
func NewStudyPlanRepo(db *sql.DB) *StudyPlanRepo {
	return &StudyPlanRepo{db: db}
}

// ── 校历 ──

const calendarColumns = "id, academic_year, semester, semester_code, semester_name, start_date, end_date, " +
	"register_date, total_weeks, week_start_day, status, created_at, updated_at"

// scanCalendar 扫描一行校历。
func scanCalendar(row interface{ Scan(...interface{}) error }) (*model.AcademicCalendar, error) {
	c := &model.AcademicCalendar{}
	err := row.Scan(&c.ID, &c.AcademicYear, &c.Semester, &c.SemesterCode,
		&c.SemesterName, &c.StartDate, &c.EndDate,
		&c.RegisterDate, &c.TotalWeeks, &c.WeekStartDay,
		&c.Status, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// ResolveCurrentCalendar 解析当前学期：在学期内 → 即将开始 → 最近结束；返回校历与教学周。
func (r *StudyPlanRepo) ResolveCurrentCalendar() (*model.AcademicCalendar, int, error) {
	today := time.Now().Format("2006-01-02")

	// 1. 当前在某个学期内
	calendar, err := scanCalendar(r.db.QueryRow(
		"SELECT "+calendarColumns+" FROM academic_calendars WHERE start_date <= ? AND end_date >= ? ORDER BY id DESC LIMIT 1",
		today, today,
	))
	if err == nil {
		return calendar, calcCurrentWeek(calendar.StartDate, today), nil
	}
	if err != sql.ErrNoRows {
		return nil, 0, err
	}

	// 2. 不在任何学期：返回最近的即将开始学期
	calendar, err = scanCalendar(r.db.QueryRow(
		"SELECT "+calendarColumns+" FROM academic_calendars WHERE start_date > ? ORDER BY start_date ASC LIMIT 1",
		today,
	))
	if err == nil {
		return calendar, 0, nil
	}
	if err != sql.ErrNoRows {
		return nil, 0, err
	}

	// 3. 都没有：返回最近的已完成学期
	calendar, err = scanCalendar(r.db.QueryRow(
		"SELECT "+calendarColumns+" FROM academic_calendars WHERE end_date < ? ORDER BY end_date DESC LIMIT 1",
		today,
	))
	if err == sql.ErrNoRows {
		return nil, 0, nil // 数据库无任何校历数据
	}
	if err != nil {
		return nil, 0, err
	}
	return calendar, 0, nil
}

// GetCalendarBySemester 按学期代码取校历（无记录返回 ErrCalendarNotFound）。
func (r *StudyPlanRepo) GetCalendarBySemester(semesterCode string) (*model.AcademicCalendar, error) {
	calendar, err := scanCalendar(r.db.QueryRow(
		"SELECT "+calendarColumns+" FROM academic_calendars WHERE semester_code = ?",
		semesterCode,
	))
	if err == sql.ErrNoRows {
		return nil, ErrCalendarNotFound
	}
	if err != nil {
		return nil, err
	}
	return calendar, nil
}

const eventColumns = "id, semester_code, event_name, event_type, start_date, end_date, week_no, affects_classes, description, created_at"

// scanCalendarEvents 扫描行集为事件列表。
func scanCalendarEvents(rows *sql.Rows) []*model.CalendarEvent {
	var list []*model.CalendarEvent
	for rows.Next() {
		e := &model.CalendarEvent{}
		var endDate, description sql.NullString
		var weekNo sql.NullInt64
		if err := rows.Scan(&e.ID, &e.SemesterCode, &e.EventName, &e.EventType,
			&e.StartDate, &endDate, &weekNo, &e.AffectsClasses, &description, &e.CreatedAt); err != nil {
			log.Printf("[WARN] 校历事件行扫描失败: %v", err)
			continue
		}
		if endDate.Valid {
			e.EndDate = endDate.String
		}
		if weekNo.Valid {
			e.WeekNo = int(weekNo.Int64)
		}
		if description.Valid {
			e.Description = description.String
		}
		list = append(list, e)
	}
	return list
}

// ListRecentEvents 查询指定学期近期事件（当前日期前后窗口）。
func (r *StudyPlanRepo) ListRecentEvents(semesterCode, from, to string) ([]*model.CalendarEvent, error) {
	rows, err := r.db.Query(
		"SELECT "+eventColumns+" FROM academic_calendar_events WHERE semester_code = ? AND start_date <= ? AND end_date >= ? "+
			"ORDER BY start_date ASC, id ASC",
		semesterCode, to, from,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCalendarEvents(rows), nil
}

// ListEventsBySemester 查询指定学期全部事件。
func (r *StudyPlanRepo) ListEventsBySemester(semesterCode string) ([]*model.CalendarEvent, error) {
	rows, err := r.db.Query(
		"SELECT "+eventColumns+" FROM academic_calendar_events WHERE semester_code = ? ORDER BY start_date ASC, id ASC",
		semesterCode,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCalendarEvents(rows), nil
}

// ── 课表 ──

// ListTimetable 用户指定学期课表。
func (r *StudyPlanRepo) ListTimetable(userID int64, semesterCode string) ([]*model.CourseScheduleItem, error) {
	rows, err := r.db.Query(
		"SELECT id, user_id, course_id, course_name, semester_code, weekday, start_period, end_period, "+
			"weeks_pattern, location, teacher, color, created_at "+
			"FROM course_schedules WHERE user_id = ? AND semester_code = ? ORDER BY weekday ASC, start_period ASC, id ASC",
		userID, semesterCode,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	all := make([]*model.CourseScheduleItem, 0)
	for rows.Next() {
		item := &model.CourseScheduleItem{}
		if err := rows.Scan(&item.ID, &item.UserID, &item.CourseID, &item.CourseName,
			&item.SemesterCode, &item.Weekday, &item.StartPeriod, &item.EndPeriod,
			&item.WeeksPattern, &item.Location, &item.Teacher, &item.Color, &item.CreatedAt); err != nil {
			log.Printf("[WARN] 课表行扫描失败: %v", err)
			continue
		}
		all = append(all, item)
	}
	return all, rows.Err()
}

// ── 学习计划 ──

// scanStudyPlan 扫描一行学习计划。
func scanStudyPlan(row interface{ Scan(...interface{}) error }) (*model.StudyPlan, error) {
	plan := &model.StudyPlan{}
	var semesterCode sql.NullString
	var linkedID sql.NullInt64
	var goalsJSON string
	var aiGen int
	if err := row.Scan(&plan.ID, &plan.UserID, &plan.Title, &plan.PlanType, &semesterCode,
		&plan.StartDate, &plan.EndDate, &goalsJSON, &plan.Progress, &aiGen, &plan.Status,
		&linkedID, &plan.CreatedAt, &plan.UpdatedAt); err != nil {
		return nil, err
	}
	if semesterCode.Valid {
		plan.SemesterCode = semesterCode.String
	}
	if linkedID.Valid {
		v := linkedID.Int64
		plan.LinkedPlanID = &v
	}
	plan.AIGenerated = aiGen == 1
	_ = json.Unmarshal([]byte(goalsJSON), &plan.Goals)
	if plan.Goals == nil {
		plan.Goals = []string{}
	}
	return plan, nil
}

// GetPlanByID 按 ID+user_id 查询计划（校验归属；无记录返回 ErrPlanNotFound）。
func (r *StudyPlanRepo) GetPlanByID(id, userID int64) (*model.StudyPlan, error) {
	plan, err := scanStudyPlan(r.db.QueryRow(
		"SELECT id, user_id, title, plan_type, semester_code, start_date, end_date, goals_json, "+
			"progress, ai_generated, status, linked_plan_id, created_at, updated_at "+
			"FROM study_plans WHERE id = ? AND user_id = ?",
		id, userID,
	))
	if err == sql.ErrNoRows {
		return nil, ErrPlanNotFound
	}
	if err != nil {
		return nil, err
	}
	return plan, nil
}

// ListPlansByUser 用户计划列表（可选类型过滤）。
func (r *StudyPlanRepo) ListPlansByUser(userID int64, planType string) ([]*model.StudyPlan, error) {
	where := []string{"user_id = ?"}
	args := []interface{}{userID}
	if planType != "" {
		where = append(where, "plan_type = ?")
		args = append(args, planType)
	}

	rows, err := r.db.Query(
		"SELECT id, user_id, title, plan_type, semester_code, start_date, end_date, goals_json, "+
			"progress, ai_generated, status, linked_plan_id, created_at, updated_at "+
			"FROM study_plans "+buildWhereClause(where)+" ORDER BY created_at DESC, id DESC",
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*model.StudyPlan
	for rows.Next() {
		plan, err := scanStudyPlan(rows)
		if err != nil {
			log.Printf("[WARN] 学习计划行扫描失败: %v", err)
			continue
		}
		list = append(list, plan)
	}
	return list, rows.Err()
}

// ListTasksByPlan 查询计划下全部任务。
func (r *StudyPlanRepo) ListTasksByPlan(planID int64) ([]*model.StudyPlanTask, error) {
	rows, err := r.db.Query(
		"SELECT id, plan_id, course_id, course_name, title, description, scheduled_date, "+
			"scheduled_duration, actual_duration, status, evidence, reflection, sort_order, created_at "+
			"FROM study_plan_tasks WHERE plan_id = ? ORDER BY sort_order ASC, id ASC",
		planID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*model.StudyPlanTask
	for rows.Next() {
		t := &model.StudyPlanTask{}
		var courseID, courseName, description, scheduledDate, evidence, reflection sql.NullString
		if err := rows.Scan(&t.ID, &t.PlanID, &courseID, &courseName, &t.Title, &description,
			&scheduledDate, &t.ScheduledDuration, &t.ActualDuration, &t.Status,
			&evidence, &reflection, &t.SortOrder, &t.CreatedAt); err != nil {
			log.Printf("[WARN] 计划任务行扫描失败: %v", err)
			continue
		}
		if courseID.Valid {
			t.CourseID = courseID.String
		}
		if courseName.Valid {
			t.CourseName = courseName.String
		}
		if description.Valid {
			t.Description = description.String
		}
		if scheduledDate.Valid {
			t.ScheduledDate = scheduledDate.String
		}
		if evidence.Valid {
			t.Evidence = evidence.String
		}
		if reflection.Valid {
			t.Reflection = reflection.String
		}
		list = append(list, t)
	}
	return list, rows.Err()
}

// FillPlanTaskStats 填充计划的任务完成统计。
func (r *StudyPlanRepo) FillPlanTaskStats(plan *model.StudyPlan) {
	if plan == nil {
		return
	}
	if err := r.db.QueryRow(
		"SELECT COUNT(*), SUM(CASE WHEN status = 'done' THEN 1 ELSE 0 END), "+
			"SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) "+
			"FROM study_plan_tasks WHERE plan_id = ?",
		plan.ID,
	).Scan(&plan.TaskTotal, &plan.TaskDone, &plan.TaskPending); err != nil {
		log.Printf("[WARN] 计划任务统计失败 plan_id=%d: %v", plan.ID, err)
	}
}

// RecalcPlanProgress 按任务完成率重算计划进度。
func (r *StudyPlanRepo) RecalcPlanProgress(planID int64) {
	var total, done int
	if err := r.db.QueryRow(
		"SELECT COUNT(*), SUM(CASE WHEN status = 'done' THEN 1 ELSE 0 END) FROM study_plan_tasks WHERE plan_id = ?",
		planID,
	).Scan(&total, &done); err != nil {
		log.Printf("[WARN] 任务完成率统计失败 plan_id=%d: %v", planID, err)
		return
	}
	var progress float64
	if total > 0 {
		progress = float64(done) / float64(total) * 100
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	if _, err := r.db.Exec("UPDATE study_plans SET progress = ?, updated_at = ? WHERE id = ?", progress, now, planID); err != nil {
		log.Printf("[WARN] 计划进度回写失败 plan_id=%d: %v", planID, err)
	}
}

// CreatePlan 创建计划并返回 ID。
func (r *StudyPlanRepo) CreatePlan(userID int64, title, planType, semesterCode, startDate, endDate, goalsJSON string) (int64, error) {
	now := time.Now().Format("2006-01-02 15:04:05")
	res, err := r.db.Exec(
		"INSERT INTO study_plans (user_id, title, plan_type, semester_code, start_date, end_date, goals_json, progress, ai_generated, status, created_at, updated_at) "+
			"VALUES (?, ?, ?, ?, ?, ?, ?, 0, 0, 'active', ?, ?)",
		userID, title, planType, semesterCode, startDate, endDate, goalsJSON, now, now,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// CreateTask 在计划下创建任务并返回 ID。
func (r *StudyPlanRepo) CreateTask(planID int64, courseID, courseName, title, description, scheduledDate string, scheduledDuration, sortOrder int) (int64, error) {
	now := time.Now().Format("2006-01-02 15:04:05")
	res, err := r.db.Exec(
		"INSERT INTO study_plan_tasks (plan_id, course_id, course_name, title, description, scheduled_date, scheduled_duration, actual_duration, status, sort_order, created_at) "+
			"VALUES (?, ?, ?, ?, ?, ?, ?, 0, 'pending', ?, ?)",
		planID, courseID, courseName, title, description, scheduledDate,
		scheduledDuration, sortOrder, now,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// PlanUpdateFields 计划部分更新字段（零值/nil 表示不改）。
type PlanUpdateFields struct {
	Title        string
	PlanType     string
	SemesterCode string
	StartDate    string
	EndDate      string
	GoalsJSON    string // 非空时覆盖 goals_json
	Status       string
	Progress     *float64
}

// UpdatePlan 计划部分更新（校验归属）。
func (r *StudyPlanRepo) UpdatePlan(id, userID int64, f PlanUpdateFields) error {
	now := time.Now().Format("2006-01-02 15:04:05")
	var progress sql.NullFloat64
	if f.Progress != nil {
		progress = sql.NullFloat64{Float64: *f.Progress, Valid: true}
	}
	_, err := r.db.Exec(
		"UPDATE study_plans SET title = COALESCE(NULLIF(?, ''), title), "+
			"plan_type = COALESCE(NULLIF(?, ''), plan_type), "+
			"semester_code = CASE WHEN ? <> '' THEN ? ELSE semester_code END, "+
			"start_date = COALESCE(NULLIF(?, ''), start_date), "+
			"end_date = COALESCE(NULLIF(?, ''), end_date), "+
			"goals_json = CASE WHEN ? <> '' THEN ? ELSE goals_json END, "+
			"status = COALESCE(NULLIF(?, ''), status), "+
			"progress = CASE WHEN ? IS NOT NULL THEN ? ELSE progress END, "+
			"updated_at = ? WHERE id = ? AND user_id = ?",
		f.Title, f.PlanType,
		f.SemesterCode, f.SemesterCode,
		f.StartDate, f.EndDate,
		f.GoalsJSON, f.GoalsJSON,
		f.Status,
		progress, progress,
		now, id, userID,
	)
	return err
}

// DeletePlan 删除计划（外键级联删任务），返回受影响行数。
func (r *StudyPlanRepo) DeletePlan(id, userID int64) (int64, error) {
	res, err := r.db.Exec("DELETE FROM study_plans WHERE id = ? AND user_id = ?", id, userID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// TaskUpdateFields 任务部分更新字段（零值/nil 表示不改）。
type TaskUpdateFields struct {
	Title             string
	Description       string
	ScheduledDate     string
	ScheduledDuration *int
	ActualDuration    *int
	Status            string
	Evidence          string
	Reflection        string
	SortOrder         *int
}

// UpdateTask 任务部分更新，返回受影响行数。
func (r *StudyPlanRepo) UpdateTask(taskID, planID int64, f TaskUpdateFields) (int64, error) {
	var scheduledDuration, actualDuration, sortOrder sql.NullInt64
	if f.ScheduledDuration != nil {
		scheduledDuration = sql.NullInt64{Int64: int64(*f.ScheduledDuration), Valid: true}
	}
	if f.ActualDuration != nil {
		actualDuration = sql.NullInt64{Int64: int64(*f.ActualDuration), Valid: true}
	}
	if f.SortOrder != nil {
		sortOrder = sql.NullInt64{Int64: int64(*f.SortOrder), Valid: true}
	}
	res, err := r.db.Exec(
		"UPDATE study_plan_tasks SET "+
			"title = COALESCE(NULLIF(?, ''), title), "+
			"description = COALESCE(NULLIF(?, ''), description), "+
			"scheduled_date = COALESCE(NULLIF(?, ''), scheduled_date), "+
			"scheduled_duration = CASE WHEN ? IS NOT NULL THEN ? ELSE scheduled_duration END, "+
			"actual_duration = CASE WHEN ? IS NOT NULL THEN ? ELSE actual_duration END, "+
			"status = COALESCE(NULLIF(?, ''), status), "+
			"evidence = COALESCE(NULLIF(?, ''), evidence), "+
			"reflection = COALESCE(NULLIF(?, ''), reflection), "+
			"sort_order = CASE WHEN ? IS NOT NULL THEN ? ELSE sort_order END "+
			"WHERE id = ? AND plan_id = ?",
		f.Title, f.Description, f.ScheduledDate,
		scheduledDuration, scheduledDuration,
		actualDuration, actualDuration,
		f.Status,
		f.Evidence, f.Reflection,
		sortOrder, sortOrder,
		taskID, planID,
	)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// PlansOverview 各时间维度计划概览。
func (r *StudyPlanRepo) PlansOverview(userID int64, planTypes []string) ([]*model.PlanOverviewItem, error) {
	overview := make([]*model.PlanOverviewItem, 0, len(planTypes))
	for _, pt := range planTypes {
		item := &model.PlanOverviewItem{PlanType: pt}
		// 计划数与平均进度
		if err := r.db.QueryRow(
			"SELECT COUNT(*), COALESCE(AVG(progress), 0) FROM study_plans WHERE user_id = ? AND plan_type = ?",
			userID, pt,
		).Scan(&item.PlanCount, &item.Progress); err != nil {
			log.Printf("[WARN] 计划概览统计失败 type=%s: %v", pt, err)
		}

		// 任务统计：通过 JOIN 关联到该用户该类型的所有计划
		var done int
		if err := r.db.QueryRow(
			"SELECT COUNT(*), COALESCE(SUM(CASE WHEN t.status = 'done' THEN 1 ELSE 0 END), 0) "+
				"FROM study_plan_tasks t JOIN study_plans p ON t.plan_id = p.id "+
				"WHERE p.user_id = ? AND p.plan_type = ?",
			userID, pt,
		).Scan(&item.TaskTotal, &done); err != nil {
			log.Printf("[WARN] 计划概览任务统计失败 type=%s: %v", pt, err)
		}
		item.TaskDone = done

		if item.TaskDone > 0 && item.TaskTotal > 0 {
			// 任务完成进度优先于平均进度
			item.Progress = float64(item.TaskDone) / float64(item.TaskTotal) * 100
		}
		overview = append(overview, item)
	}
	return overview, nil
}

// calcCurrentWeek 计算当前教学周：(today - start_date) / 7 + 1
func calcCurrentWeek(startDate, today string) int {
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return 0
	}
	now, err := time.Parse("2006-01-02", today)
	if err != nil {
		return 0
	}
	if now.Before(start) {
		return 0
	}
	days := int(now.Sub(start).Hours() / 24)
	return days/7 + 1
}
