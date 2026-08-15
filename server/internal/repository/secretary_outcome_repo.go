package repository

import (
	"database/sql"
	"fmt"
	"time"
)

// GraduationOutcome 毕业生去向（真实登记，含审核流）
// 对应迁移 087_graduation_outcome 表。
type GraduationOutcome struct {
	ID            int64  `json:"id"`
	StudentID     int64  `json:"student_id"`
	StudentName   string `json:"student_name"`
	College       string `json:"college"`
	Major         string `json:"major"`
	GraduateYear  int    `json:"graduate_year"`
	OutcomeType   string `json:"outcome_type"` // employment/postgrad/study_abroad/flexible/entrepreneurship/unemployed
	EmployerName  string `json:"employer_name"`
	Position      string `json:"position"`
	Remark        string `json:"remark"`
	Status        string `json:"status"` // pending/approved/rejected
	SubmittedBy   int64  `json:"submitted_by"`
	SubmittedRole string `json:"submitted_role"`
	ReviewedBy    int64  `json:"reviewed_by"`
	ReviewedName  string `json:"reviewed_name"`
	ReviewNote    string `json:"review_note"`
	ReviewedAt    string `json:"reviewed_at"`
	DataSource    string `json:"data_source"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

// OutcomeTypeMeta 去向类型元信息
var OutcomeTypeMeta = map[string]string{
	"employment":       "就业",
	"postgrad":         "国内升读研",
	"study_abroad":     "出国/境外升学",
	"flexible":         "灵活就业",
	"entrepreneurship": "自主创业",
	"unemployed":       "暂未就业",
}

// SecretaryOutcomeRepo 书记教育成果数据访问层
// 数据源：竞赛获奖(competition_registrations)、入党(party_progress)、学业(student_grades)、
// 毕业(graduation_progress/thesis_topics)、离校(process_records)、谈心(talk_records)、
// 后勤(facility_records)、去向(graduation_outcome) —— 全部真实表聚合，不造数。
type SecretaryOutcomeRepo struct {
	db *sql.DB
}

func NewSecretaryOutcomeRepo(db *sql.DB) *SecretaryOutcomeRepo {
	return &SecretaryOutcomeRepo{db: db}
}

// ============ 毕业去向登记（CRUD + 审核）============

// CreateOutcome 新增一条毕业去向（学生自报 status=pending，教辅录入可取 approved 或 pending）
func (r *SecretaryOutcomeRepo) CreateOutcome(o *GraduationOutcome) (int64, error) {
	res, err := r.db.Exec(
		`INSERT INTO graduation_outcome
		 (student_id, student_name, college, major, graduate_year, outcome_type,
		  employer_name, position, remark, status, submitted_by, submitted_role, data_source)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		o.StudentID, o.StudentName, o.College, o.Major, o.GraduateYear, o.OutcomeType,
		o.EmployerName, o.Position, o.Remark, o.Status, o.SubmittedBy, o.SubmittedRole, "real",
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// ListOutcomes 查询毕业去向（支持状态/学院/届别/学生过滤）
func (r *SecretaryOutcomeRepo) ListOutcomes(status, college string, year int, studentID int64, limit int) ([]GraduationOutcome, error) {
	q := `SELECT id, student_id, student_name, college, major, graduate_year, outcome_type,
	             employer_name, position, remark, status, submitted_by, submitted_role,
	             reviewed_by, reviewed_name, review_note, reviewed_at, data_source, created_at, updated_at
	      FROM graduation_outcome WHERE 1=1`
	args := []interface{}{}
	if status != "" {
		q += ` AND status = ?`
		args = append(args, status)
	}
	if college != "" {
		q += ` AND college = ?`
		args = append(args, college)
	}
	if year > 0 {
		q += ` AND graduate_year = ?`
		args = append(args, year)
	}
	if studentID > 0 {
		q += ` AND student_id = ?`
		args = append(args, studentID)
	}
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	q += ` ORDER BY id DESC LIMIT ?`
	args = append(args, limit)

	rows, err := r.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []GraduationOutcome
	for rows.Next() {
		var o GraduationOutcome
		if err := rows.Scan(&o.ID, &o.StudentID, &o.StudentName, &o.College, &o.Major, &o.GraduateYear,
			&o.OutcomeType, &o.EmployerName, &o.Position, &o.Remark, &o.Status,
			&o.SubmittedBy, &o.SubmittedRole, &o.ReviewedBy, &o.ReviewedName, &o.ReviewNote,
			&o.ReviewedAt, &o.DataSource, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, o)
	}
	return list, rows.Err()
}

// ReviewOutcome 审核一条毕业去向（教辅：批准后计入统计）
func (r *SecretaryOutcomeRepo) ReviewOutcome(id, reviewerID int64, reviewerName, status, note string) error {
	if status != "approved" && status != "rejected" {
		return fmt.Errorf("无效审核状态: %s", status)
	}
	_, err := r.db.Exec(
		`UPDATE graduation_outcome SET status = ?, reviewed_by = ?, reviewed_name = ?,
		        review_note = ?, reviewed_at = ?, updated_at = datetime('now','localtime')
		  WHERE id = ?`,
		status, reviewerID, reviewerName, note, time.Now().Format(time.RFC3339), id,
	)
	return err
}

// CountPendingOutcomes 待审核条数（供教辅入口角标）
func (r *SecretaryOutcomeRepo) CountPendingOutcomes() (int, error) {
	var n int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM graduation_outcome WHERE status='pending'`).Scan(&n)
	return n, err
}

// ============ 书记教育成果大屏聚合（真实 SQL）============

// EducationOutcomeDashboard 聚合书记大屏数据。
// college!="" 时限定某学院（学院书记看本院）；否则全校（学校书记）。
func (r *SecretaryOutcomeRepo) EducationOutcomeDashboard(college string) (map[string]interface{}, error) {
	res := map[string]interface{}{}
	and := ""   // 附加在已有 WHERE 子句后的查询（竞赛/去向）：AND college=?
	where := "" // 附加在无 WHERE 子句后的查询（party/学业）：WHERE college=?
	args := []interface{}{}
	if college != "" {
		and = ` AND college = ?`
		where = ` WHERE college = ?`
		args = append(args, college)
	}

	// ① 竞赛获奖：总数 / 按等级 / 按级别 / 指导教师榜
	var awardTotal int
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM competition_registrations WHERE status='awarded'`+and, args...).Scan(&awardTotal); err != nil {
		return nil, fmt.Errorf("竞赛获奖统计: %w", err)
	}
	awardByLevel := map[string]int{}
	rows, err := r.db.Query(`SELECT COALESCE(award_level,'') lvl, COUNT(*) FROM competition_registrations WHERE status='awarded'`+and+` GROUP BY lvl`, args...)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var lvl string
		var c int
		if err := rows.Scan(&lvl, &c); err != nil {
			rows.Close()
			return nil, err
		}
		awardByLevel[lvl] = c
	}
	rows.Close()

	awardByNat := map[string]int{}
	rows, err = r.db.Query(`SELECT COALESCE(cm.level,'') lvl, COUNT(*) FROM competition_registrations cr JOIN competitions cm ON cm.id=cr.competition_id WHERE cr.status='awarded'`+and+` GROUP BY lvl`, args...)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var lvl string
		var c int
		if err := rows.Scan(&lvl, &c); err != nil {
			rows.Close()
			return nil, err
		}
		awardByNat[lvl] = c
	}
	rows.Close()

	// 指导教师榜（必须体现：教师除了授课还指导学生竞赛）
	advisor := []map[string]interface{}{}
	qAdvisor := `SELECT COALESCE(cr.advisor_name,''), cr.college, COUNT(*) c FROM competition_registrations cr
	             WHERE cr.status='awarded' AND cr.advisor_name <> ''`
	qAdvisorArgs := []interface{}{}
	if college != "" {
		qAdvisor += ` AND cr.college = ?`
		qAdvisorArgs = append(qAdvisorArgs, college)
	}
	qAdvisor += ` GROUP BY cr.advisor_name, cr.college ORDER BY c DESC LIMIT 20`
	rows, err = r.db.Query(qAdvisor, qAdvisorArgs...)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var name, col string
		var c int
		if err := rows.Scan(&name, &col, &c); err != nil {
			rows.Close()
			return nil, err
		}
		advisor = append(advisor, map[string]interface{}{"name": name, "college": col, "awards": c})
	}
	rows.Close()

	// ② 入党：各阶段人数 / 党员(正式)数 / 入党介绍人榜
	partyStage := map[string]int{}
	rows, err = r.db.Query(`SELECT current_stage, COUNT(*) FROM party_progress`+where+` GROUP BY current_stage`, args...)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var s string
		var c int
		if err := rows.Scan(&s, &c); err != nil {
			rows.Close()
			return nil, err
		}
		partyStage[s] = c
	}
	rows.Close()

	partyMembers := map[string]int{}
	qMembers := `SELECT status, COUNT(*) FROM party_progress WHERE status IN ('member','probation')` + and
	rows, err = r.db.Query(qMembers+` GROUP BY status`, args...)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var s string
		var c int
		if err := rows.Scan(&s, &c); err != nil {
			rows.Close()
			return nil, err
		}
		partyMembers[s] = c
	}
	rows.Close()

	// ③ 学业：通过率 / 人数（college 过滤需 join student_profile_snapshot.college）
	var gradeCnt, passCnt int
	qGrade := `SELECT COUNT(sg.id), COALESCE(SUM(sg.passed),0) FROM student_grades sg`
	qGradeArgs := []interface{}{}
	if college != "" {
		qGrade += ` JOIN student_profile_snapshot sps ON sps.user_id = CAST(sg.user_id AS INTEGER) WHERE sps.college = ?`
		qGradeArgs = append(qGradeArgs, college)
	}
	if err := r.db.QueryRow(qGrade, qGradeArgs...).Scan(&gradeCnt, &passCnt); err != nil {
		// 兼容：快照表可能缺失该生，回退全校统计而非报错
		if err2 := r.db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(passed),0) FROM student_grades`).Scan(&gradeCnt, &passCnt); err2 != nil {
			return nil, fmt.Errorf("学业统计: %w", err)
		}
	}

	// ④ 谈心：人次（全校/全院记录总量）
	var talkTotal int
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM talk_records`).Scan(&talkTotal); err != nil {
		talkTotal = 0
	}

	// ⑤ 后勤服务：总条数
	var facilityTotal int
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM facility_records`).Scan(&facilityTotal); err != nil {
		facilityTotal = 0
	}

	// ⑥ 毕业去向（仅 approved 计入统计；未接入则诚实 not_available）
	outcomeStats, err := r.OutcomeStats(college)
	if err != nil {
		outcomeStats = map[string]interface{}{"data_source": "not_available", "note": "毕业去向数据待接入"}
	}

	res["college"] = college
	res["competition"] = map[string]interface{}{
		"total_awards":   awardTotal,
		"by_level":       awardByLevel,
		"by_nationality": awardByNat,
		"advisor_rank":   advisor,
		"data_source":    "real",
	}
	res["party"] = map[string]interface{}{
		"stage_distribution": partyStage,
		"members":            partyMembers,
		"data_source":        "real",
	}
	res["academic"] = map[string]interface{}{
		"grade_count": gradeCnt,
		"pass_rate":   passRate(passCnt, gradeCnt),
		"data_source": "real",
	}
	res["counseling"] = map[string]interface{}{"talk_total": talkTotal, "data_source": "real"}
	res["facility"] = map[string]interface{}{"record_total": facilityTotal, "data_source": "real"}
	res["outcome"] = outcomeStats
	res["meta"] = map[string]interface{}{"outcome_types": OutcomeTypeMeta}
	return res, nil
}

// OutcomeStats 毕业去向统计（仅 approved），returns data_source real / not_available
func (r *SecretaryOutcomeRepo) OutcomeStats(college string) (map[string]interface{}, error) {
	and := ""
	args := []interface{}{}
	if college != "" {
		and = ` AND college = ?`
		args = append(args, college)
	}
	var total int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM graduation_outcome WHERE status='approved'`+and, args...).Scan(&total)
	if err != nil {
		return nil, err
	}
	if total == 0 {
		// 表可能空（未录入），诚实返回 not_available
		return map[string]interface{}{
			"total": 0, "rate_by_type": map[string]int{}, "data_source": "not_available",
			"note": "毕业去向数据待接入（已建表，需教辅录入并经审核）",
		}, nil
	}
	byType := map[string]int{}
	rows, err := r.db.Query(`SELECT outcome_type, COUNT(*) FROM graduation_outcome WHERE status='approved'`+and+` GROUP BY outcome_type`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var t string
		var c int
		if err := rows.Scan(&t, &c); err != nil {
			return nil, err
		}
		byType[t] = c
	}
	// 就业率 = (employment+flexible+entrepreneurship) / total；考研(升学)率 = (postgrad+study_abroad)/total
	employ := byType["employment"] + byType["flexible"] + byType["entrepreneurship"]
	postgrad := byType["postgrad"] + byType["study_abroad"]
	return map[string]interface{}{
		"total":           total,
		"rate_by_type":    byType,
		"employment_rate": percent(employ, total),
		"postgrad_rate":   percent(postgrad, total),
		"data_source":     "real",
	}, nil
}

func percent(a, b int) float64 {
	if b <= 0 {
		return 0
	}
	return float64(a) / float64(b) * 100
}

func passRate(pass, total int) float64 {
	if total <= 0 {
		return 0
	}
	return float64(pass) / float64(total) * 100
}

// ══════════════════════════════════════════════════════════════
// 党建育人聚合（书记接线）
// ══════════════════════════════════════════════════════════════
//
// ownerID != "" 时限定本院（按 users.owner_id 精确匹配，学院书记看本院）；
// ownerID == "" 时全校（学校书记）。
//
// 诚实边界：party_progress/party_study_records 目前由学生自报（意向登记），
// 非组织确认。real=有记录；self_reported=只有自报（当前阶段)；
// not_available=表里无任何记录（未接入真实党建数据）。
func (r *SecretaryOutcomeRepo) PartyDashboard(ownerID string) (map[string]interface{}, error) {
	res := map[string]interface{}{}
	// 本院范围过滤（通过 users.owner_id 唯一归属键，而非 party_progress.college 中文学名）
	scopeJoin := ""
	scopeArg := []interface{}{}
	if ownerID != "" {
		scopeJoin = ` JOIN users u ON u.id = pp.user_id WHERE u.owner_id = ?`
		scopeArg = append(scopeArg, ownerID)
	}

	// ① 入党漏斗：各当前阶段人数（applicant 申请 → activist 积极分子 → development 发展对象 → probation 预备 → member 党员）
	stageRows, err := r.db.Query(`SELECT pp.current_stage, COUNT(*) FROM party_progress pp`+scopeJoin+` GROUP BY pp.current_stage`, scopeArg...)
	if err != nil {
		return nil, fmt.Errorf("党建漏斗统计: %w", err)
	}
	defer stageRows.Close()
	stageDist := map[string]int{}
	stageTotal := 0
	for stageRows.Next() {
		var s string
		var c int
		if err := stageRows.Scan(&s, &c); err != nil {
			return nil, fmt.Errorf("党建漏斗扫描: %w", err)
		}
		stageDist[s] = c
		stageTotal += c
	}
	if err := stageRows.Err(); err != nil {
		return nil, fmt.Errorf("党建漏斗遍历: %w", err)
	}

	// ② 党员数（正式 member + 预备 probation）
	memberMap := map[string]int{}
	qMembers2 := `SELECT p.status, COUNT(*) FROM party_progress p JOIN users u ON u.id = p.user_id WHERE p.status IN ('member','probation')`
	mArgs := []interface{}{}
	if ownerID != "" {
		qMembers2 += ` AND u.owner_id = ?`
		mArgs = append(mArgs, ownerID)
	}
	qMembers2 += ` GROUP BY p.status`
	mRows, err := r.db.Query(qMembers2, mArgs...)
	if err != nil {
		return nil, fmt.Errorf("党员数统计: %w", err)
	}
	defer mRows.Close()
	for mRows.Next() {
		var s string
		var c int
		if err := mRows.Scan(&s, &c); err != nil {
			return nil, fmt.Errorf("党员数扫描: %w", err)
		}
		memberMap[s] = c
	}
	if err := mRows.Err(); err != nil {
		return nil, fmt.Errorf("党员数遍历: %w", err)
	}

	// ③ 学习记录：总人次 / 总时长(小时) / 按类型分布 / 党课(理论)人次
	studyScopeJoin := ""
	studyScopeArg := []interface{}{}
	if ownerID != "" {
		studyScopeJoin = ` JOIN users u ON u.id = psr.user_id WHERE u.owner_id = ?`
		studyScopeArg = append(studyScopeArg, ownerID)
	}
	var studyCount, studyDuration int
	if err := r.db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(psr.duration),0) FROM party_study_records psr`+studyScopeJoin, studyScopeArg...).Scan(&studyCount, &studyDuration); err != nil {
		return nil, fmt.Errorf("学习记录统计: %w", err)
	}
	typeRows, err := r.db.Query(`SELECT psr.study_type, COUNT(*), COALESCE(SUM(psr.duration),0) FROM party_study_records psr`+studyScopeJoin+` GROUP BY psr.study_type`, studyScopeArg...)
	if err != nil {
		return nil, fmt.Errorf("学习类型统计: %w", err)
	}
	defer typeRows.Close()
	studyByType := map[string]map[string]int{}
	for typeRows.Next() {
		var t string
		var c, d int
		if err := typeRows.Scan(&t, &c, &d); err != nil {
			return nil, fmt.Errorf("学习类型扫描: %w", err)
		}
		studyByType[t] = map[string]int{"count": c, "hours": d / 60}
	}
	if err := typeRows.Err(); err != nil {
		return nil, fmt.Errorf("学习类型遍历: %w", err)
	}

	res["stage_distribution"] = stageDist
	res["stage_total"] = stageTotal
	res["members"] = memberMap
	res["study_records"] = studyCount
	res["study_hours"] = studyDuration / 60
	res["study_by_type"] = studyByType
	res["owner_id"] = ownerID
	res["data_source"] = partyDataSource(stageTotal, studyCount)
	return res, nil
}

// partyDataSource 诚实标注党建数据来源
func partyDataSource(stageTotal, studyCount int) string {
	if stageTotal == 0 && studyCount == 0 {
		return "not_available" // 无任何记录：未接入真实党建数据
	}
	return "self_reported" // 现有记录为自报/意向登记，尚未经组织确认
}
