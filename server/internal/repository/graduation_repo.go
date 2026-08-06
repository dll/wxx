package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/dll/wxx/server/internal/model"
)

// GraduationRepo 毕设选题数据访问层
type GraduationRepo struct {
	db *sql.DB
}

// NewGraduationRepo 创建毕设选题仓库
func NewGraduationRepo(db *sql.DB) *GraduationRepo {
	return &GraduationRepo{db: db}
}

// ── 导师相关 ──

// ListAdvisors 获取导师列表
func (r *GraduationRepo) ListAdvisors(college string, page, pageSize int) ([]*model.Advisor, int, error) {
	where := []string{"is_active = 1"}
	args := []interface{}{}

	if college != "" {
		where = append(where, "college = ?")
		args = append(args, college)
	}

	whereStr := strings.Join(where, " AND ")

	var total int
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM advisors WHERE %s", whereStr)
	if err := r.db.QueryRow(countSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计导师数量失败: %w", err)
	}

	offset := (page - 1) * pageSize
	querySQL := fmt.Sprintf("SELECT id, name, advisor_id, title, college, department, research_areas, max_students, is_active, created_at, updated_at FROM advisors WHERE %s ORDER BY name LIMIT ? OFFSET ?", whereStr)
	args = append(args, pageSize, offset)

	rows, err := r.db.Query(querySQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询导师列表失败: %w", err)
	}
	defer rows.Close()

	var advisors []*model.Advisor
	for rows.Next() {
		a := &model.Advisor{}
		if err := rows.Scan(&a.ID, &a.Name, &a.AdvisorID, &a.Title, &a.College, &a.Department, &a.ResearchAreas, &a.MaxStudents, &a.IsActive, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("扫描导师记录失败: %w", err)
		}
		advisors = append(advisors, a)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("遍历导师记录失败: %w", err)
	}
	return advisors, total, nil
}

// GetAdvisor 获取导师详情
func (r *GraduationRepo) GetAdvisor(id int64) (*model.Advisor, error) {
	a := &model.Advisor{}
	err := r.db.QueryRow("SELECT id, name, advisor_id, title, college, department, research_areas, max_students, is_active, created_at, updated_at FROM advisors WHERE id = ?", id).
		Scan(&a.ID, &a.Name, &a.AdvisorID, &a.Title, &a.College, &a.Department, &a.ResearchAreas, &a.MaxStudents, &a.IsActive, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("查询导师详情失败: %w", err)
	}
	return a, nil
}

// GetAdvisorCurrentCount 获取导师当前指导学生数
func (r *GraduationRepo) GetAdvisorCurrentCount(advisorID int64) (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM student_topic_selections WHERE advisor_id = ? AND status IN ('pending', 'confirmed')", advisorID).Scan(&count)
	return count, err
}

// ── 选题相关 ──

// ListTopics 获取选题列表
func (r *GraduationRepo) ListTopics(college, major, difficulty, status string, batch, page, pageSize int) ([]*model.ThesisTopic, int, error) {
	where := []string{"1=1"}
	args := []interface{}{}

	if college != "" {
		where = append(where, "college = ?")
		args = append(args, college)
	}
	if major != "" {
		where = append(where, "major = ?")
		args = append(args, major)
	}
	if difficulty != "" {
		where = append(where, "difficulty = ?")
		args = append(args, difficulty)
	}
	if status != "" {
		where = append(where, "status = ?")
		args = append(args, status)
	}
	if batch > 0 {
		where = append(where, "batch = ?")
		args = append(args, batch)
	}

	whereStr := strings.Join(where, " AND ")

	var total int
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM thesis_topics WHERE %s", whereStr)
	if err := r.db.QueryRow(countSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计选题数量失败: %w", err)
	}

	offset := (page - 1) * pageSize
	querySQL := fmt.Sprintf(`
		SELECT t.id, t.title, t.advisor_id, a.name as advisor_name, t.college, t.major, t.topic_type, t.nature, 
		       t.result_form, t.difficulty, t.description, t.requirements, t.keywords, 
		       t.max_students, t.selected_count, t.batch, t.status, t.created_at, t.updated_at
		FROM thesis_topics t 
		LEFT JOIN advisors a ON t.advisor_id = a.id 
		WHERE %s ORDER BY t.created_at DESC LIMIT ? OFFSET ?`, whereStr)
	args = append(args, pageSize, offset)

	rows, err := r.db.Query(querySQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询选题列表失败: %w", err)
	}
	defer rows.Close()

	var topics []*model.ThesisTopic
	for rows.Next() {
		t := &model.ThesisTopic{}
		if err := rows.Scan(&t.ID, &t.Title, &t.AdvisorID, &t.AdvisorName, &t.College, &t.Major, &t.TopicType, &t.Nature,
			&t.ResultForm, &t.Difficulty, &t.Description, &t.Requirements, &t.Keywords,
			&t.MaxStudents, &t.SelectedCount, &t.Batch, &t.Status, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("扫描选题记录失败: %w", err)
		}
		topics = append(topics, t)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("遍历选题记录失败: %w", err)
	}
	return topics, total, nil
}

// GetTopic 获取选题详情
func (r *GraduationRepo) GetTopic(id int64) (*model.ThesisTopic, error) {
	t := &model.ThesisTopic{}
	err := r.db.QueryRow(`
		SELECT t.id, t.title, t.advisor_id, a.name as advisor_name, t.college, t.major, t.topic_type, t.nature, 
		       t.result_form, t.difficulty, t.description, t.requirements, t.keywords, 
		       t.max_students, t.selected_count, t.batch, t.status, t.created_at, t.updated_at
		FROM thesis_topics t 
		LEFT JOIN advisors a ON t.advisor_id = a.id 
		WHERE t.id = ?`, id).
		Scan(&t.ID, &t.Title, &t.AdvisorID, &t.AdvisorName, &t.College, &t.Major, &t.TopicType, &t.Nature,
			&t.ResultForm, &t.Difficulty, &t.Description, &t.Requirements, &t.Keywords,
			&t.MaxStudents, &t.SelectedCount, &t.Batch, &t.Status, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("查询选题详情失败: %w", err)
	}
	return t, nil
}

// CreateTopic 创建选题
func (r *GraduationRepo) CreateTopic(t *model.ThesisTopic) (int64, error) {
	result, err := r.db.Exec(`
		INSERT INTO thesis_topics (title, advisor_id, college, major, topic_type, nature, result_form, difficulty, description, requirements, keywords, max_students, batch, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.Title, t.AdvisorID, t.College, t.Major, t.TopicType, t.Nature, t.ResultForm, t.Difficulty,
		t.Description, t.Requirements, t.Keywords, t.MaxStudents, t.Batch, t.Status)
	if err != nil {
		return 0, fmt.Errorf("创建选题失败: %w", err)
	}
	return result.LastInsertId()
}

// UpdateTopic 更新选题（字段白名单防注入）
func (r *GraduationRepo) UpdateTopic(id int64, fields map[string]interface{}) error {
	if len(fields) == 0 {
		return nil
	}
	allowedFields := map[string]bool{
		"title": true, "advisor_id": true, "college": true, "major": true,
		"topic_type": true, "nature": true, "result_form": true, "difficulty": true,
		"description": true, "requirements": true, "keywords": true,
		"max_students": true, "status": true,
	}
	setParts := []string{}
	args := []interface{}{}
	for k, v := range fields {
		if !allowedFields[k] {
			continue
		}
		setParts = append(setParts, k+" = ?")
		args = append(args, v)
	}
	if len(setParts) == 0 {
		return nil
	}
	args = append(args, id)
	query := fmt.Sprintf("UPDATE thesis_topics SET %s, updated_at = datetime('now') WHERE id = ?", strings.Join(setParts, ", "))
	_, err := r.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("更新选题失败: %w", err)
	}
	return nil
}

// DeleteTopic 删除毕设选题（先清理关联选题记录）
func (r *GraduationRepo) DeleteTopic(id int64) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM student_topic_selections WHERE topic_id = ?`, id); err != nil {
		return fmt.Errorf("清理选题关联记录失败: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM thesis_topics WHERE id = ?`, id); err != nil {
		return fmt.Errorf("删除选题失败: %w", err)
	}
	return tx.Commit()
}

// ── 学生选题相关 ──

// ListSelections 获取学生选题列表
func (r *GraduationRepo) ListSelections(topicID int64, batch, page, pageSize int) ([]*model.StudentTopicSelection, int, error) {
	where := []string{"1=1"}
	args := []interface{}{}

	if topicID > 0 {
		where = append(where, "topic_id = ?")
		args = append(args, topicID)
	}
	if batch > 0 {
		where = append(where, "batch = ?")
		args = append(args, batch)
	}

	whereStr := strings.Join(where, " AND ")

	var total int
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM student_topic_selections WHERE %s", whereStr)
	if err := r.db.QueryRow(countSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计选题记录数量失败: %w", err)
	}

	offset := (page - 1) * pageSize
	querySQL := fmt.Sprintf(`
		SELECT s.id, s.user_id, s.student_id, s.student_name, s.college, s.major, s.class_name, s.batch,
		       s.topic_id, t.title as topic_name, s.advisor_id, a.name as advisor_name, 
		       s.status, s.preference_order, s.reason, s.confirmed_at, s.created_at, s.updated_at
		FROM student_topic_selections s
		LEFT JOIN thesis_topics t ON s.topic_id = t.id
		LEFT JOIN advisors a ON s.advisor_id = a.id
		WHERE %s ORDER BY s.preference_order LIMIT ? OFFSET ?`, whereStr)
	args = append(args, pageSize, offset)

	rows, err := r.db.Query(querySQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询选题记录列表失败: %w", err)
	}
	defer rows.Close()

	var selections []*model.StudentTopicSelection
	for rows.Next() {
		s := &model.StudentTopicSelection{}
		if err := rows.Scan(&s.ID, &s.UserID, &s.StudentID, &s.StudentName, &s.College, &s.Major, &s.ClassName, &s.Batch,
			&s.TopicID, &s.TopicName, &s.AdvisorID, &s.AdvisorName,
			&s.Status, &s.PreferenceOrder, &s.Reason, &s.ConfirmedAt, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("扫描选题记录失败: %w", err)
		}
		selections = append(selections, s)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("遍历选题记录失败: %w", err)
	}
	return selections, total, nil
}

// GetUserSelection 获取用户的选题记录
func (r *GraduationRepo) GetUserSelection(userID int64) (*model.StudentTopicSelection, error) {
	s := &model.StudentTopicSelection{}
	err := r.db.QueryRow(`
		SELECT s.id, s.user_id, s.student_id, s.student_name, s.college, s.major, s.class_name, s.batch,
		       s.topic_id, t.title as topic_name, s.advisor_id, a.name as advisor_name, 
		       s.status, s.preference_order, s.reason, s.confirmed_at, s.created_at, s.updated_at
		FROM student_topic_selections s
		LEFT JOIN thesis_topics t ON s.topic_id = t.id
		LEFT JOIN advisors a ON s.advisor_id = a.id
		WHERE s.user_id = ?
		ORDER BY s.created_at DESC LIMIT 1`, userID).
		Scan(&s.ID, &s.UserID, &s.StudentID, &s.StudentName, &s.College, &s.Major, &s.ClassName, &s.Batch,
			&s.TopicID, &s.TopicName, &s.AdvisorID, &s.AdvisorName,
			&s.Status, &s.PreferenceOrder, &s.Reason, &s.ConfirmedAt, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("查询用户选题失败: %w", err)
	}
	return s, nil
}

// CreateSelection 创建选题记录
func (r *GraduationRepo) CreateSelection(s *model.StudentTopicSelection) (int64, error) {
	result, err := r.db.Exec(`
		INSERT INTO student_topic_selections (user_id, student_id, student_name, college, major, class_name, batch, topic_id, advisor_id, status, preference_order, reason)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.UserID, s.StudentID, s.StudentName, s.College, s.Major, s.ClassName, s.Batch,
		s.TopicID, s.AdvisorID, s.Status, s.PreferenceOrder, s.Reason)
	if err != nil {
		return 0, fmt.Errorf("创建选题记录失败: %w", err)
	}

	// 更新选题已选人数
	if s.TopicID > 0 {
		if _, err := r.db.Exec("UPDATE thesis_topics SET selected_count = selected_count + 1 WHERE id = ?", s.TopicID); err != nil {
			return 0, fmt.Errorf("更新选题人数失败: %w", err)
		}
	}

	return result.LastInsertId()
}

// UpdateSelectionStatus 更新选题状态
func (r *GraduationRepo) UpdateSelectionStatus(id int64, status string) error {
	now := time.Now().Format(time.RFC3339)
	_, err := r.db.Exec("UPDATE student_topic_selections SET status = ?, confirmed_at = ?, updated_at = datetime('now') WHERE id = ?", status, now, id)
	if err != nil {
		return fmt.Errorf("更新选题状态失败: %w", err)
	}
	return nil
}

// ── 里程碑相关 ──

// ListMilestones 获取里程碑列表
func (r *GraduationRepo) ListMilestones(batch int) ([]*model.GraduationMilestone, error) {
	rows, err := r.db.Query("SELECT id, batch, code, name, deadline, weight, description, sort_order, created_at FROM graduation_milestones WHERE batch = ? ORDER BY sort_order", batch)
	if err != nil {
		return nil, fmt.Errorf("查询里程碑列表失败: %w", err)
	}
	defer rows.Close()

	var milestones []*model.GraduationMilestone
	for rows.Next() {
		m := &model.GraduationMilestone{}
		if err := rows.Scan(&m.ID, &m.Batch, &m.Code, &m.Name, &m.Deadline, &m.Weight, &m.Description, &m.SortOrder, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("扫描里程碑记录失败: %w", err)
		}
		milestones = append(milestones, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历里程碑记录失败: %w", err)
	}
	return milestones, nil
}

// ── 统计相关 ──

// GetTopicStats 获取选题统计
func (r *GraduationRepo) GetTopicStats(batch int) (map[string]interface{}, error) {
	stats := map[string]interface{}{}

	// 总选题数
	var totalTopics int
	if err := r.db.QueryRow("SELECT COUNT(*) FROM thesis_topics WHERE batch = ?", batch).Scan(&totalTopics); err != nil {
		return nil, fmt.Errorf("统计选题总数失败: %w", err)
	}
	stats["total_topics"] = totalTopics

	// 各难度选题数
	difficultyRows, err := r.db.Query("SELECT difficulty, COUNT(*) FROM thesis_topics WHERE batch = ? GROUP BY difficulty", batch)
	if err != nil {
		return nil, fmt.Errorf("查询难度分布失败: %w", err)
	}
	defer difficultyRows.Close()
	difficultyStats := map[string]int{}
	for difficultyRows.Next() {
		var d string
		var c int
		if err := difficultyRows.Scan(&d, &c); err != nil {
			return nil, fmt.Errorf("扫描难度分布失败: %w", err)
		}
		difficultyStats[d] = c
	}
	if err := difficultyRows.Err(); err != nil {
		return nil, fmt.Errorf("遍历难度分布失败: %w", err)
	}
	stats["difficulty_distribution"] = difficultyStats

	// 各状态选题数
	statusRows, err := r.db.Query("SELECT status, COUNT(*) FROM thesis_topics WHERE batch = ? GROUP BY status", batch)
	if err != nil {
		return nil, fmt.Errorf("查询状态分布失败: %w", err)
	}
	defer statusRows.Close()
	statusStats := map[string]int{}
	for statusRows.Next() {
		var s string
		var c int
		if err := statusRows.Scan(&s, &c); err != nil {
			return nil, fmt.Errorf("扫描状态分布失败: %w", err)
		}
		statusStats[s] = c
	}
	if err := statusRows.Err(); err != nil {
		return nil, fmt.Errorf("遍历状态分布失败: %w", err)
	}
	stats["status_distribution"] = statusStats

	// 选题人数
	var totalSelections int
	if err := r.db.QueryRow("SELECT COUNT(*) FROM student_topic_selections WHERE batch = ? AND status IN ('pending', 'confirmed')", batch).Scan(&totalSelections); err != nil {
		return nil, fmt.Errorf("统计选题人数失败: %w", err)
	}
	stats["total_selections"] = totalSelections

	return stats, nil
}
