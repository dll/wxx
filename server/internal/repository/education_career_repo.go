// Package repository 就业指导仓库（P4-d：从 education_career_handler 下沉的 11 处裸 SQL）。
package repository

import (
	"database/sql"
	"errors"
	"log"

	"github.com/dll/wxx/server/internal/model"
)

// ErrCareerNotFound 就业域资源不存在（政策/职位）。
var ErrCareerNotFound = errors.New("就业资源不存在")

// CareerRepo 就业指导数据访问层。
type CareerRepo struct {
	db *sql.DB
}

// NewCareerRepo 创建就业指导仓库。
func NewCareerRepo(db *sql.DB) *CareerRepo {
	return &CareerRepo{db: db}
}

// ── 就业政策 ──

// ListCareerPolicies 分页政策列表。
func (r *CareerRepo) ListCareerPolicies(category, level string, page, pageSize int) ([]*model.CareerPolicy, int, error) {
	where := []string{"status = 'active'"}
	args := []interface{}{}
	if category != "" {
		where = append(where, "category = ?")
		args = append(args, category)
	}
	if level != "" {
		where = append(where, "level = ?")
		args = append(args, level)
	}
	whereSQL := buildWhereClause(where)

	var total int
	if err := r.db.QueryRow("SELECT COUNT(*) FROM career_policies "+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(
		"SELECT id, policy_id, title, category, level, source, summary, publish_date, tags, view_count, created_at "+
			"FROM career_policies "+whereSQL+" ORDER BY publish_date DESC, id DESC LIMIT ? OFFSET ?",
		append(args, pageSize, pageOffset(page, pageSize))...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	list := make([]*model.CareerPolicy, 0)
	for rows.Next() {
		item := &model.CareerPolicy{}
		if err := rows.Scan(&item.ID, &item.PolicyID, &item.Title, &item.Category, &item.Level,
			&item.Source, &item.Summary, &item.PublishDate, &item.Tags, &item.ViewCount, &item.CreatedAt); err != nil {
			continue
		}
		list = append(list, item)
	}
	return list, total, rows.Err()
}

// GetCareerPolicyDetail 活动状态政策详情（无记录返回 ErrCareerNotFound）。
func (r *CareerRepo) GetCareerPolicyDetail(policyID string) (*model.CareerPolicyDetail, error) {
	detail := &model.CareerPolicyDetail{}
	err := r.db.QueryRow(
		"SELECT id, policy_id, title, category, level, source, content, summary, publish_date, "+
			"effective_date, expiry_date, tags, status, view_count, created_at, updated_at "+
			"FROM career_policies WHERE policy_id = ? AND status = 'active'", policyID,
	).Scan(&detail.ID, &detail.PolicyID, &detail.Title, &detail.Category, &detail.Level,
		&detail.Source, &detail.Content, &detail.Summary, &detail.PublishDate,
		&detail.EffectiveDate, &detail.ExpiryDate, &detail.Tags, &detail.Status,
		&detail.ViewCount, &detail.CreatedAt, &detail.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrCareerNotFound
	}
	if err != nil {
		return nil, err
	}
	return detail, nil
}

// IncrementPolicyView 政策阅读数自增（best-effort）。
func (r *CareerRepo) IncrementPolicyView(id int64) {
	if _, err := r.db.Exec("UPDATE career_policies SET view_count = view_count + 1 WHERE id = ?", id); err != nil {
		log.Printf("[WARN] 政策阅读数自增失败 id=%d: %v", id, err)
	}
}

// ── 招聘信息 ──

// ListJobPostings 分页职位列表。
func (r *CareerRepo) ListJobPostings(positionType, industry, location string, page, pageSize int) ([]*model.JobPosting, int, error) {
	where := []string{"status = 'active'"}
	args := []interface{}{}
	if positionType != "" {
		where = append(where, "position_type = ?")
		args = append(args, positionType)
	}
	if industry != "" {
		where = append(where, "industry = ?")
		args = append(args, industry)
	}
	if location != "" {
		where = append(where, "location LIKE ?")
		args = append(args, "%"+location+"%")
	}
	whereSQL := buildWhereClause(where)

	var total int
	if err := r.db.QueryRow("SELECT COUNT(*) FROM job_postings "+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(
		"SELECT id, job_id, company_name, company_logo, position_name, position_type, industry, "+
			"salary_min, salary_max, salary_unit, location, education, deadline, view_count, apply_count, created_at "+
			"FROM job_postings "+whereSQL+" ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?",
		append(args, pageSize, pageOffset(page, pageSize))...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	list := make([]*model.JobPosting, 0)
	for rows.Next() {
		item := &model.JobPosting{}
		if err := rows.Scan(&item.ID, &item.JobID, &item.CompanyName, &item.CompanyLogo,
			&item.PositionName, &item.PositionType, &item.Industry, &item.SalaryMin,
			&item.SalaryMax, &item.SalaryUnit, &item.Location, &item.Education,
			&item.Deadline, &item.ViewCount, &item.ApplyCount, &item.CreatedAt); err != nil {
			continue
		}
		list = append(list, item)
	}
	return list, total, rows.Err()
}

// GetJobPostingDetail 活动状态职位详情（无记录返回 ErrCareerNotFound）。
func (r *CareerRepo) GetJobPostingDetail(jobID string) (*model.JobPostingDetail, error) {
	detail := &model.JobPostingDetail{}
	err := r.db.QueryRow(
		"SELECT id, job_id, company_name, company_logo, company_intro, position_name, position_type, "+
			"industry, salary_min, salary_max, salary_unit, location, education, major_requirement, "+
			"description, requirement, benefits, application_url, deadline, source, status, "+
			"view_count, apply_count, created_at, updated_at "+
			"FROM job_postings WHERE job_id = ? AND status = 'active'", jobID,
	).Scan(&detail.ID, &detail.JobID, &detail.CompanyName, &detail.CompanyLogo, &detail.CompanyIntro,
		&detail.PositionName, &detail.PositionType, &detail.Industry, &detail.SalaryMin,
		&detail.SalaryMax, &detail.SalaryUnit, &detail.Location, &detail.Education,
		&detail.MajorRequirement, &detail.Description, &detail.Requirement, &detail.Benefits,
		&detail.ApplicationURL, &detail.Deadline, &detail.Source, &detail.Status,
		&detail.ViewCount, &detail.ApplyCount, &detail.CreatedAt, &detail.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrCareerNotFound
	}
	if err != nil {
		return nil, err
	}
	return detail, nil
}

// IncrementJobView 职位浏览数自增（best-effort）。
func (r *CareerRepo) IncrementJobView(id int64) {
	if _, err := r.db.Exec("UPDATE job_postings SET view_count = view_count + 1 WHERE id = ?", id); err != nil {
		log.Printf("[WARN] 职位浏览数自增失败 id=%d: %v", id, err)
	}
}

// ── 宣讲会 ──

// ListInfoSessions 宣讲会列表（单日或日期范围）。
func (r *CareerRepo) ListInfoSessions(date, startDate, endDate string) ([]*model.InfoSession, error) {
	where := []string{"status = 'active'"}
	args := []interface{}{}
	if date != "" {
		where = append(where, "date = ?")
		args = append(args, date)
	} else {
		if startDate != "" {
			where = append(where, "date >= ?")
			args = append(args, startDate)
		}
		if endDate != "" {
			where = append(where, "date <= ?")
			args = append(args, endDate)
		}
	}

	rows, err := r.db.Query(
		"SELECT id, session_id, company_name, company_logo, title, date, time_start, time_end, "+
			"location, campus, description, registration_url, capacity, registered_count, created_at "+
			"FROM info_sessions "+buildWhereClause(where)+" ORDER BY date ASC, time_start ASC, id ASC",
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*model.InfoSession, 0)
	for rows.Next() {
		item := &model.InfoSession{}
		if err := rows.Scan(&item.ID, &item.SessionID, &item.CompanyName, &item.CompanyLogo,
			&item.Title, &item.Date, &item.TimeStart, &item.TimeEnd, &item.Location,
			&item.Campus, &item.Description, &item.RegistrationURL, &item.Capacity,
			&item.RegisteredCount, &item.CreatedAt); err != nil {
			continue
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

// ── 面试题库 ──

// ListInterviewQuestions 分页面试题。
func (r *CareerRepo) ListInterviewQuestions(category, industry string, page, pageSize int) ([]*model.InterviewQuestion, int, error) {
	where := []string{"status = 'active'"}
	args := []interface{}{}
	if category != "" {
		where = append(where, "category = ?")
		args = append(args, category)
	}
	if industry != "" {
		where = append(where, "industry = ?")
		args = append(args, industry)
	}
	whereSQL := buildWhereClause(where)

	var total int
	if err := r.db.QueryRow("SELECT COUNT(*) FROM interview_questions "+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(
		"SELECT id, question_id, category, industry, position, question, answer_hint, keywords, difficulty, created_at "+
			"FROM interview_questions "+whereSQL+" ORDER BY difficulty ASC, id ASC LIMIT ? OFFSET ?",
		append(args, pageSize, pageOffset(page, pageSize))...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	list := make([]*model.InterviewQuestion, 0)
	for rows.Next() {
		item := &model.InterviewQuestion{}
		if err := rows.Scan(&item.ID, &item.QuestionID, &item.Category, &item.Industry,
			&item.Position, &item.Question, &item.AnswerHint, &item.Keywords,
			&item.Difficulty, &item.CreatedAt); err != nil {
			continue
		}
		list = append(list, item)
	}
	return list, total, rows.Err()
}
