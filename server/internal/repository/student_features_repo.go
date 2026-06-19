package repository

import (
	"database/sql"
	"fmt"
	"strings"
)

// StudentFeaturesRepo 学生功能数据访问层（竞赛+规划+入党+社团）
type StudentFeaturesRepo struct {
	db *sql.DB
}

// NewStudentFeaturesRepo 创建学生功能仓库
func NewStudentFeaturesRepo(db *sql.DB) *StudentFeaturesRepo {
	return &StudentFeaturesRepo{db: db}
}

// ══════════════════════════════════════════════════════════════
// 学科竞赛
// ══════════════════════════════════════════════════════════════

// ListCompetitions 获取竞赛列表
func (r *StudentFeaturesRepo) ListCompetitions(level, category, status string, page, pageSize int) ([]map[string]interface{}, int, error) {
	where := []string{"1=1"}
	args := []interface{}{}
	if level != "" {
		where = append(where, "level = ?")
		args = append(args, level)
	}
	if category != "" {
		where = append(where, "category = ?")
		args = append(args, category)
	}
	if status != "" {
		where = append(where, "status = ?")
		args = append(args, status)
	}
	whereStr := strings.Join(where, " AND ")

	var total int
	if err := r.db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM competitions WHERE %s", whereStr), args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计竞赛数量失败: %w", err)
	}

	offset := (page - 1) * pageSize
	queryArgs := append(args, pageSize, offset)
	rows, err := r.db.Query(fmt.Sprintf("SELECT id, name, level, category, organizer, description, requirements, features, registration_start, registration_end, competition_date, result_date, website, resource_links, max_team_size, is_team_competition, status, created_at FROM competitions WHERE %s ORDER BY competition_date DESC LIMIT ? OFFSET ?", whereStr), queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []map[string]interface{}
	for rows.Next() {
		var id, maxTeamSize, isTeam int
		var name, lev, cat, org, desc, req, feats, regStart, regEnd, compDate, resDate, website, resLinks, status, created string
		if err := rows.Scan(&id, &name, &lev, &cat, &org, &desc, &req, &feats, &regStart, &regEnd, &compDate, &resDate, &website, &resLinks, &maxTeamSize, &isTeam, &status, &created); err != nil {
			return nil, 0, fmt.Errorf("扫描竞赛记录失败: %w", err)
		}
		items = append(items, map[string]interface{}{
			"id": id, "name": name, "level": lev, "category": cat, "organizer": org,
			"description": desc, "requirements": req, "features": feats,
			"registration_start": regStart, "registration_end": regEnd,
			"competition_date": compDate, "result_date": resDate, "website": website,
			"resource_links": resLinks, "max_team_size": maxTeamSize,
			"is_team_competition": isTeam, "status": status, "created_at": created,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("遍历竞赛记录失败: %w", err)
	}
	return items, total, nil
}

// GetCompetition 获取竞赛详情
func (r *StudentFeaturesRepo) GetCompetition(id int64) (map[string]interface{}, error) {
	var maxTeamSize, isTeam int
	var name, lev, cat, org, desc, req, feats, regStart, regEnd, compDate, resDate, website, resLinks, status, created string
	err := r.db.QueryRow("SELECT name, level, category, organizer, description, requirements, features, registration_start, registration_end, competition_date, result_date, website, resource_links, max_team_size, is_team_competition, status, created_at FROM competitions WHERE id = ?", id).
		Scan(&name, &lev, &cat, &org, &desc, &req, &feats, &regStart, &regEnd, &compDate, &resDate, &website, &resLinks, &maxTeamSize, &isTeam, &status, &created)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"id": id, "name": name, "level": lev, "category": cat, "organizer": org,
		"description": desc, "requirements": req, "features": feats,
		"registration_start": regStart, "registration_end": regEnd,
		"competition_date": compDate, "result_date": resDate, "website": website,
		"resource_links": resLinks, "max_team_size": maxTeamSize,
		"is_team_competition": isTeam, "status": status, "created_at": created,
	}, nil
}

// RegisterCompetition 报名竞赛
func (r *StudentFeaturesRepo) RegisterCompetition(competitionID, userID int64, studentID, studentName, college, major, className, teamName, teamMembers, advisorName string) (int64, error) {
	result, err := r.db.Exec(`INSERT INTO competition_registrations (competition_id, user_id, student_id, student_name, college, major, class_name, team_name, team_members, advisor_name, status) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'registered')`,
		competitionID, userID, studentID, studentName, college, major, className, teamName, teamMembers, advisorName)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// GetMyCompetitionRegistrations 获取我的竞赛报名
func (r *StudentFeaturesRepo) GetMyCompetitionRegistrations(userID int64) ([]map[string]interface{}, error) {
	rows, err := r.db.Query(`SELECT cr.id, cr.competition_id, c.name, c.level, cr.status, cr.work_title, cr.award_level, cr.award_date, cr.created_at
		FROM competition_registrations cr LEFT JOIN competitions c ON cr.competition_id = c.id
		WHERE cr.user_id = ? ORDER BY cr.created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []map[string]interface{}
	for rows.Next() {
		var id, compID int
		var compName, lev, status, workTitle, awardLevel, awardDate, created string
		if err := rows.Scan(&id, &compID, &compName, &lev, &status, &workTitle, &awardLevel, &awardDate, &created); err != nil {
			return nil, fmt.Errorf("扫描竞赛报名记录失败: %w", err)
		}
		items = append(items, map[string]interface{}{
			"id": id, "competition_id": compID, "competition_name": compName, "level": lev,
			"status": status, "work_title": workTitle, "award_level": awardLevel, "award_date": awardDate, "created_at": created,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历竞赛报名记录失败: %w", err)
	}
	return items, nil
}

// SubmitWork 提交作品
func (r *StudentFeaturesRepo) SubmitWork(regID int64, workTitle, workDesc, workFileURL string) error {
	_, err := r.db.Exec("UPDATE competition_registrations SET work_title = ?, work_description = ?, work_file_url = ?, status = 'submitted', updated_at = datetime('now') WHERE id = ?", workTitle, workDesc, workFileURL, regID)
	return err
}

// GetCompetitionStats 竞赛统计
func (r *StudentFeaturesRepo) GetCompetitionStats() (map[string]interface{}, error) {
	stats := map[string]interface{}{}

	var total int
	if err := r.db.QueryRow("SELECT COUNT(*) FROM competitions").Scan(&total); err != nil {
		return nil, fmt.Errorf("统计竞赛总数失败: %w", err)
	}
	stats["total_competitions"] = total

	var regCount int
	if err := r.db.QueryRow("SELECT COUNT(*) FROM competition_registrations").Scan(&regCount); err != nil {
		return nil, fmt.Errorf("统计报名总数失败: %w", err)
	}
	stats["total_registrations"] = regCount

	var awardCount int
	if err := r.db.QueryRow("SELECT COUNT(*) FROM competition_registrations WHERE award_level != '' AND award_level IS NOT NULL").Scan(&awardCount); err != nil {
		return nil, fmt.Errorf("统计获奖总数失败: %w", err)
	}
	stats["total_awards"] = awardCount

	levelRows, err := r.db.Query("SELECT level, COUNT(*) FROM competitions GROUP BY level")
	if err != nil {
		return nil, fmt.Errorf("查询竞赛等级分布失败: %w", err)
	}
	defer levelRows.Close()
	dist := map[string]int{}
	for levelRows.Next() {
		var l string
		var c int
		if err := levelRows.Scan(&l, &c); err != nil {
			return nil, fmt.Errorf("扫描竞赛等级分布失败: %w", err)
		}
		dist[l] = c
	}
	if err := levelRows.Err(); err != nil {
		return nil, fmt.Errorf("遍历竞赛等级分布失败: %w", err)
	}
	stats["level_distribution"] = dist
	return stats, nil
}

// ══════════════════════════════════════════════════════════════
// 大学规划
// ══════════════════════════════════════════════════════════════

// ListPlanTemplates 获取规划模板列表
func (r *StudentFeaturesRepo) ListPlanTemplates(category string) ([]map[string]interface{}, error) {
	where := "is_active = 1"
	args := []interface{}{}
	if category != "" {
		where += " AND category = ?"
		args = append(args, category)
	}
	rows, err := r.db.Query(fmt.Sprintf("SELECT id, name, category, description, target_audience, duration, goals, milestones, success_cases, ai_prompt FROM plan_templates WHERE %s ORDER BY name", where), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []map[string]interface{}
	for rows.Next() {
		var id int
		var name, cat, desc, audience, dur, goals, mstones, cases, prompt string
		if err := rows.Scan(&id, &name, &cat, &desc, &audience, &dur, &goals, &mstones, &cases, &prompt); err != nil {
			return nil, fmt.Errorf("扫描规划模板失败: %w", err)
		}
		items = append(items, map[string]interface{}{
			"id": id, "name": name, "category": cat, "description": desc,
			"target_audience": audience, "duration": dur, "goals": goals,
			"milestones": mstones, "success_cases": cases, "ai_prompt": prompt,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历规划模板失败: %w", err)
	}
	return items, nil
}

// ListMyPlans 获取我的规划列表
func (r *StudentFeaturesRepo) ListMyPlans(userID int64) ([]map[string]interface{}, error) {
	rows, err := r.db.Query(`SELECT sp.id, sp.title, sp.category, sp.progress, sp.status, sp.reviewer_comment, sp.created_at, pt.name as template_name
		FROM student_plans sp LEFT JOIN plan_templates pt ON sp.template_id = pt.id
		WHERE sp.user_id = ? ORDER BY sp.created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []map[string]interface{}
	for rows.Next() {
		var id int
		var title, cat, status, comment, created, tmplName string
		var progress float64
		if err := rows.Scan(&id, &title, &cat, &progress, &status, &comment, &created, &tmplName); err != nil {
			return nil, fmt.Errorf("扫描规划记录失败: %w", err)
		}
		items = append(items, map[string]interface{}{
			"id": id, "title": title, "category": cat, "progress": progress,
			"status": status, "reviewer_comment": comment, "created_at": created, "template_name": tmplName,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历规划记录失败: %w", err)
	}
	return items, nil
}

// CreatePlan 创建规划
func (r *StudentFeaturesRepo) CreatePlan(userID int64, templateID int, title, category string, academicYear, semester int, goals string) (int64, error) {
	result, err := r.db.Exec(`INSERT INTO student_plans (user_id, template_id, title, category, academic_year, semester, goals, status) VALUES (?, ?, ?, ?, ?, ?, ?, 'draft')`,
		userID, templateID, title, category, academicYear, semester, goals)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// UpdatePlanStatus 更新规划状态
func (r *StudentFeaturesRepo) UpdatePlanStatus(planID int64, status, comment string) error {
	_, err := r.db.Exec("UPDATE student_plans SET status = ?, reviewer_comment = ?, reviewed_at = datetime('now'), updated_at = datetime('now') WHERE id = ?", status, comment, planID)
	return err
}

// UpdatePlanProgress 更新规划进度
func (r *StudentFeaturesRepo) UpdatePlanProgress(planID int64, progress float64) error {
	_, err := r.db.Exec("UPDATE student_plans SET progress = ?, updated_at = datetime('now') WHERE id = ?", progress, planID)
	return err
}

// ══════════════════════════════════════════════════════════════
// 入党教育
// ══════════════════════════════════════════════════════════════

// ListPartyStages 获取入党阶段列表
func (r *StudentFeaturesRepo) ListPartyStages() ([]map[string]interface{}, error) {
	rows, err := r.db.Query("SELECT id, code, name, description, required_docs, sort_order FROM party_stages ORDER BY sort_order")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []map[string]interface{}
	for rows.Next() {
		var id, order int
		var code, name, desc, docs string
		if err := rows.Scan(&id, &code, &name, &desc, &docs, &order); err != nil {
			return nil, fmt.Errorf("扫描入党阶段失败: %w", err)
		}
		items = append(items, map[string]interface{}{
			"id": id, "code": code, "name": name, "description": desc, "required_docs": docs, "sort_order": order,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历入党阶段失败: %w", err)
	}
	return items, nil
}

// GetMyPartyProgress 获取我的入党进度
func (r *StudentFeaturesRepo) GetMyPartyProgress(userID int64) (map[string]interface{}, error) {
	var id, uid int
	var sid, sname, college, stage, applyDate, actDate, devDate, probStart, convDate, status, notes, created, updated string
	err := r.db.QueryRow(`SELECT id, user_id, student_id, student_name, college, current_stage, apply_date, activator_date, development_date, probation_start, conversion_date, status, notes, created_at, updated_at FROM party_progress WHERE user_id = ?`, userID).
		Scan(&id, &uid, &sid, &sname, &college, &stage, &applyDate, &actDate, &devDate, &probStart, &convDate, &status, &notes, &created, &updated)
	if err != nil {
		return nil, err
	}
	// 获取阶段名称
	var stageName string
	if err := r.db.QueryRow("SELECT name FROM party_stages WHERE code = ?", stage).Scan(&stageName); err != nil {
		stageName = stage
	}
	return map[string]interface{}{
		"id": id, "user_id": uid, "student_id": sid, "student_name": sname, "college": college,
		"current_stage": stage, "current_stage_name": stageName,
		"apply_date": applyDate, "activator_date": actDate, "development_date": devDate,
		"probation_start": probStart, "conversion_date": convDate,
		"status": status, "notes": notes, "created_at": created, "updated_at": updated,
	}, nil
}

// UpdatePartyProgress 更新入党进度
func (r *StudentFeaturesRepo) UpdatePartyProgress(userID int64, stage, notes string) error {
	var exists int
	if err := r.db.QueryRow("SELECT COUNT(*) FROM party_progress WHERE user_id = ?", userID).Scan(&exists); err != nil {
		return fmt.Errorf("查询入党进度失败: %w", err)
	}
	if exists == 0 {
		_, err := r.db.Exec(`INSERT INTO party_progress (user_id, current_stage, status, notes) VALUES (?, ?, ?, ?)`,
			userID, stage, stage, notes)
		return err
	}
	_, err := r.db.Exec("UPDATE party_progress SET current_stage = ?, status = ?, notes = ?, updated_at = datetime('now') WHERE user_id = ?", stage, stage, notes, userID)
	return err
}

// ListMyStudyRecords 获取我的学习记录
func (r *StudentFeaturesRepo) ListMyStudyRecords(userID int64) ([]map[string]interface{}, error) {
	rows, err := r.db.Query(`SELECT id, study_type, title, content, duration, study_date, certificate, status, created_at FROM party_study_records WHERE user_id = ? ORDER BY study_date DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []map[string]interface{}
	for rows.Next() {
		var id, dur int
		var stype, title, content, sdate, cert, status, created string
		if err := rows.Scan(&id, &stype, &title, &content, &dur, &sdate, &cert, &status, &created); err != nil {
			return nil, fmt.Errorf("扫描学习记录失败: %w", err)
		}
		items = append(items, map[string]interface{}{
			"id": id, "study_type": stype, "title": title, "content": content,
			"duration": dur, "study_date": sdate, "certificate": cert, "status": status, "created_at": created,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历学习记录失败: %w", err)
	}
	return items, nil
}

// AddStudyRecord 添加学习记录
func (r *StudentFeaturesRepo) AddStudyRecord(userID int64, studyType, title, content string, duration int, studyDate, certificate string) (int64, error) {
	result, err := r.db.Exec(`INSERT INTO party_study_records (user_id, study_type, title, content, duration, study_date, certificate) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		userID, studyType, title, content, duration, studyDate, certificate)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// GetPartyStats 入党统计
func (r *StudentFeaturesRepo) GetPartyStats() (map[string]interface{}, error) {
	stats := map[string]interface{}{}

	var total int
	if err := r.db.QueryRow("SELECT COUNT(*) FROM party_progress").Scan(&total); err != nil {
		return nil, fmt.Errorf("统计入党人数失败: %w", err)
	}
	stats["total_applicants"] = total

	var studyCount int
	if err := r.db.QueryRow("SELECT COUNT(*) FROM party_study_records").Scan(&studyCount); err != nil {
		return nil, fmt.Errorf("统计学习记录数失败: %w", err)
	}
	stats["total_study_records"] = studyCount

	var totalDuration int
	if err := r.db.QueryRow("SELECT COALESCE(SUM(duration), 0) FROM party_study_records").Scan(&totalDuration); err != nil {
		return nil, fmt.Errorf("统计学习时长失败: %w", err)
	}
	stats["total_study_hours"] = totalDuration / 60

	stageRows, err := r.db.Query("SELECT current_stage, COUNT(*) FROM party_progress GROUP BY current_stage")
	if err != nil {
		return nil, fmt.Errorf("查询入党阶段分布失败: %w", err)
	}
	defer stageRows.Close()
	dist := map[string]int{}
	for stageRows.Next() {
		var s string
		var c int
		if err := stageRows.Scan(&s, &c); err != nil {
			return nil, fmt.Errorf("扫描入党阶段分布失败: %w", err)
		}
		dist[s] = c
	}
	if err := stageRows.Err(); err != nil {
		return nil, fmt.Errorf("遍历入党阶段分布失败: %w", err)
	}
	stats["stage_distribution"] = dist
	return stats, nil
}

// ══════════════════════════════════════════════════════════════
// 社团生活
// ══════════════════════════════════════════════════════════════

// ListClubs 获取社团列表
func (r *StudentFeaturesRepo) ListClubs(category string, page, pageSize int) ([]map[string]interface{}, int, error) {
	where := "status = 'active'"
	args := []interface{}{}
	if category != "" {
		where += " AND category = ?"
		args = append(args, category)
	}
	var total int
	if err := r.db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM clubs WHERE %s", where), args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计社团数量失败: %w", err)
	}

	offset := (page - 1) * pageSize
	queryArgs := append(args, pageSize, offset)
	rows, err := r.db.Query(fmt.Sprintf("SELECT id, name, category, description, president, contact_info, member_count, max_members, status, created_at FROM clubs WHERE %s ORDER BY member_count DESC LIMIT ? OFFSET ?", where), queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var items []map[string]interface{}
	for rows.Next() {
		var id, mc, mm int
		var name, cat, desc, pres, contact, status, created string
		if err := rows.Scan(&id, &name, &cat, &desc, &pres, &contact, &mc, &mm, &status, &created); err != nil {
			return nil, 0, fmt.Errorf("扫描社团记录失败: %w", err)
		}
		items = append(items, map[string]interface{}{
			"id": id, "name": name, "category": cat, "description": desc,
			"president": pres, "contact_info": contact,
			"member_count": mc, "max_members": mm, "status": status, "created_at": created,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("遍历社团记录失败: %w", err)
	}
	return items, total, nil
}

// GetClub 获取社团详情
func (r *StudentFeaturesRepo) GetClub(id int64) (map[string]interface{}, error) {
	var mc, mm int
	var name, cat, desc, pres, contact, status, created string
	err := r.db.QueryRow("SELECT name, category, description, president, contact_info, member_count, max_members, status, created_at FROM clubs WHERE id = ?", id).
		Scan(&name, &cat, &desc, &pres, &contact, &mc, &mm, &status, &created)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"id": id, "name": name, "category": cat, "description": desc,
		"president": pres, "contact_info": contact,
		"member_count": mc, "max_members": mm, "status": status, "created_at": created,
	}, nil
}

// JoinClub 加入社团
func (r *StudentFeaturesRepo) JoinClub(clubID, userID int64, studentID, studentName, role string) (int64, error) {
	result, err := r.db.Exec(`INSERT INTO club_members (club_id, user_id, student_id, student_name, role, join_date, status) VALUES (?, ?, ?, ?, ?, date('now'), 'active')`,
		clubID, userID, studentID, studentName, role)
	if err != nil {
		return 0, err
	}
	if _, err := r.db.Exec("UPDATE clubs SET member_count = member_count + 1 WHERE id = ?", clubID); err != nil {
		return 0, fmt.Errorf("更新社团人数失败: %w", err)
	}
	return result.LastInsertId()
}

// GetMyClubs 获取我加入的社团
func (r *StudentFeaturesRepo) GetMyClubs(userID int64) ([]map[string]interface{}, error) {
	rows, err := r.db.Query(`SELECT cm.id, cm.club_id, c.name, cm.role, cm.join_date, cm.status
		FROM club_members cm LEFT JOIN clubs c ON cm.club_id = c.id
		WHERE cm.user_id = ? ORDER BY cm.join_date DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []map[string]interface{}
	for rows.Next() {
		var id, clubID int
		var name, role, joinDate, status string
		if err := rows.Scan(&id, &clubID, &name, &role, &joinDate, &status); err != nil {
			return nil, fmt.Errorf("扫描我的社团记录失败: %w", err)
		}
		items = append(items, map[string]interface{}{
			"id": id, "club_id": clubID, "club_name": name, "role": role, "join_date": joinDate, "status": status,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历我的社团记录失败: %w", err)
	}
	return items, nil
}

// ListClubActivities 获取社团活动列表
func (r *StudentFeaturesRepo) ListClubActivities(clubID int64, status string, page, pageSize int) ([]map[string]interface{}, int, error) {
	where := "1=1"
	args := []interface{}{}
	if clubID > 0 {
		where += " AND ca.club_id = ?"
		args = append(args, clubID)
	}
	if status != "" {
		where += " AND ca.status = ?"
		args = append(args, status)
	}
	var total int
	if err := r.db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM club_activities ca WHERE %s", where), args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计社团活动数量失败: %w", err)
	}

	offset := (page - 1) * pageSize
	queryArgs := append(args, pageSize, offset)
	rows, err := r.db.Query(fmt.Sprintf(`SELECT ca.id, ca.club_id, c.name, ca.title, ca.description, ca.activity_type, ca.start_time, ca.end_time, ca.location, ca.max_participants, ca.current_participants, ca.status, ca.created_at
		FROM club_activities ca LEFT JOIN clubs c ON ca.club_id = c.id WHERE %s ORDER BY ca.start_time DESC LIMIT ? OFFSET ?`, where), queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var items []map[string]interface{}
	for rows.Next() {
		var id, clubID, mp, cp int
		var clubName, title, desc, atype, stime, etime, loc, status, created string
		if err := rows.Scan(&id, &clubID, &clubName, &title, &desc, &atype, &stime, &etime, &loc, &mp, &cp, &status, &created); err != nil {
			return nil, 0, fmt.Errorf("扫描社团活动记录失败: %w", err)
		}
		items = append(items, map[string]interface{}{
			"id": id, "club_id": clubID, "club_name": clubName, "title": title,
			"description": desc, "activity_type": atype, "start_time": stime, "end_time": etime,
			"location": loc, "max_participants": mp, "current_participants": cp,
			"status": status, "created_at": created,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("遍历社团活动记录失败: %w", err)
	}
	return items, total, nil
}

// RegisterClubActivity 报名社团活动
func (r *StudentFeaturesRepo) RegisterClubActivity(activityID, userID int64, studentName string) (int64, error) {
	result, err := r.db.Exec(`INSERT INTO club_activity_registrations (activity_id, user_id, student_name, status) VALUES (?, ?, ?, 'registered')`,
		activityID, userID, studentName)
	if err != nil {
		return 0, err
	}
	if _, err := r.db.Exec("UPDATE club_activities SET current_participants = current_participants + 1 WHERE id = ?", activityID); err != nil {
		return 0, fmt.Errorf("更新活动参与人数失败: %w", err)
	}
	return result.LastInsertId()
}
