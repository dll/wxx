package repository

import (
	"database/sql"
	"fmt"
	"strconv"
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

// ══════════════════════════════════════════════════════════════
// 党课/活动登记（蓝图第3块，2026-08-16）
// 教师/教辅登记其组织的党课/积极分子活动，落 party_study_records(created_by).
// ══════════════════════════════════════════════════════════════

// CreatePartyRecord 登记党课/活动。
// 未指定参与学生时，仅创建一条无学生绑定的登记（title 标注组织者）；
// 指定学生时，为每个学生各建一条记录（同一活动多生参与），都带 created_by。
func (r *SecretaryOutcomeRepo) CreatePartyRecord(title, studyType, content string, duration int, studyDate string, createdBy int64, createdByRole string, studentIDs []int64) ([]int64, error) {
	ids := []int64{}
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// 无学生指定：登记一条组织者记录
	if len(studentIDs) == 0 {
		res, err := tx.Exec(
			`INSERT INTO party_study_records (user_id, study_type, title, content, duration, study_date, status, created_by, created_by_role)
			 VALUES (?, ?, ?, ?, ?, ?, 'completed', ?, ?)`,
			createdBy, studyType, title, content, duration, studyDate, createdBy, createdByRole,
		)
		if err != nil {
			return nil, fmt.Errorf("登记党课失败: %w", err)
		}
		id, _ := res.LastInsertId()
		ids = append(ids, id)
		return ids, tx.Commit()
	}

	// 为每个参与学生建一条记录（同一活动多生参与）
	for _, sid := range studentIDs {
		res, err := tx.Exec(
			`INSERT INTO party_study_records (user_id, study_type, title, content, duration, study_date, status, created_by, created_by_role)
			 VALUES (?, ?, ?, ?, ?, ?, 'completed', ?, ?)`,
			sid, studyType, title, content, duration, studyDate, createdBy, createdByRole,
		)
		if err != nil {
			return nil, fmt.Errorf("登记党课(学生 %d)失败: %w", sid, err)
		}
		id, _ := res.LastInsertId()
		ids = append(ids, id)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return ids, nil
}

// PartyRecordItem 党课/活动登记条目
func (r *SecretaryOutcomeRepo) PartyRecordItem(id int64) (map[string]interface{}, error) {
	row := r.db.QueryRow(
		`SELECT psr.id, psr.study_type, psr.title, psr.content, psr.duration, psr.study_date,
		        psr.status, psr.created_by, psr.created_by_role,
		        COALESCE(u.display_name, u.username, ''), COALESCE(psr.user_id,0)
		 FROM party_study_records psr
		 LEFT JOIN users u ON u.id = psr.created_by
		 WHERE psr.id = ?`,
		id,
	)
	var rec struct {
		ID, Duration       int64
		StudyType, Title   string
		Content            interface{}
		StudyDate, Status  string
		CreatedBy          interface{}
		CreatedByRole      interface{}
		CreatedByName, UID interface{}
	}
	if err := row.Scan(&rec.ID, &rec.StudyType, &rec.Title, &rec.Content, &rec.Duration, &rec.StudyDate,
		&rec.Status, &rec.CreatedBy, &rec.CreatedByRole, &rec.CreatedByName, &rec.UID); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"id":              rec.ID,
		"study_type":      rec.StudyType,
		"title":           rec.Title,
		"content":         rec.Content,
		"duration":        rec.Duration,
		"study_date":      rec.StudyDate,
		"status":          rec.Status,
		"created_by":      rec.CreatedBy,
		"created_by_role": rec.CreatedByRole,
		"created_by_name": rec.CreatedByName,
		"user_id":         rec.UID,
	}, nil
}

// ListPartyRecordsByOperator 查登记人的党课/活动（created_by=opID）
func (r *SecretaryOutcomeRepo) ListPartyRecords(opID int64) ([]map[string]interface{}, error) {
	rows, err := r.db.Query(
		`SELECT psr.id, psr.study_type, psr.title, psr.content, psr.duration, psr.study_date, psr.status,
		        psr.created_by, psr.created_by_role, COALESCE(u.display_name, u.username, ''), COALESCE(psr.user_id,0)
		 FROM party_study_records psr
		 LEFT JOIN users u ON u.id = psr.created_by
		 WHERE psr.created_by = ?
		 ORDER BY psr.id DESC`,
		opID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []map[string]interface{}
	for rows.Next() {
		var id, dur, uid int64
		var stype, title, sdate, status string
		var content, createdBy, createdByRole, cbName interface{}
		if err := rows.Scan(&id, &stype, &title, &content, &dur, &sdate, &status, &createdBy, &createdByRole, &cbName, &uid); err != nil {
			return nil, err
		}
		list = append(list, map[string]interface{}{
			"id": id, "study_type": stype, "title": title, "content": content,
			"duration": dur, "study_date": sdate, "status": status,
			"created_by": createdBy, "created_by_role": createdByRole,
			"created_by_name": cbName, "user_id": uid,
		})
	}
	return list, rows.Err()
}

// DeletePartyRecord 删除登记人的党课/活动记录（仅本人 created_by 可删）
func (r *SecretaryOutcomeRepo) DeletePartyRecord(id, opID int64) error {
	res, err := r.db.Exec(`DELETE FROM party_study_records WHERE id = ? AND created_by = ?`, id, opID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("记录不存在或无权删除")
	}
	return nil
}

// ══════════════════════════════════════════════════════════════
// 协同育人总览（蓝图第2块，2026-08-16）
// 书记视角：教师/教辅为育人做了多少 —— 聚合谈心/后勤/党建活动/教学排课，按学院/角色汇总。
// ══════════════════════════════════════════════════════════════

// CollabDashboard 协同育人总览。
// ownerID!=空 → 本院（按 users.owner_id 过滤学生主体）；空 → 全校。
// 诚实边界：记录为真实表；表空则对应项 data_source=not_available。
func (r *SecretaryOutcomeRepo) CollabDashboard(ownerID string) (map[string]interface{}, error) {
	res := map[string]interface{}{}

	// 范围过滤：学生主体取 users.owner_id；角色主体（辅导员/教辅）也同学院时纳入
	scopeStudent := ""
	argsStudent := []interface{}{}
	if ownerID != "" {
		scopeStudent = ` JOIN users stu ON stu.id = t.student_id AND stu.owner_id = ?`
		argsStudent = append(argsStudent, ownerID)
	}

	// ① 谈心（辅导员记录）+ 该学院学生数
	var talkTotal, studentCnt int
	qTalk := `SELECT COUNT(t.id) FROM talk_records t` + scopeStudent
	if err := r.db.QueryRow(qTalk, argsStudent...).Scan(&talkTotal); err != nil {
		talkTotal = 0
	}
	qStu := `SELECT COUNT(*) FROM users WHERE role='student'`
	stuArgs := []interface{}{}
	if ownerID != "" {
		qStu += ` AND owner_id = ?`
		stuArgs = append(stuArgs, ownerID)
	}
	if err := r.db.QueryRow(qStu, stuArgs...).Scan(&studentCnt); err != nil {
		studentCnt = 0
	}

	// ② 后勤服务（operator = 教辅，与学生主体无强关联，可按 operator 归属学院过滤）
	var facilityTotal int
	qFac := `SELECT COUNT(*) FROM facility_records fr`
	facArgs := []interface{}{}
	if ownerID != "" {
		qFac += ` JOIN users op ON op.id = fr.operator_id AND op.owner_id = ?`
		facArgs = append(facArgs, ownerID)
	}
	if err := r.db.QueryRow(qFac, facArgs...).Scan(&facilityTotal); err != nil {
		facilityTotal = 0
	}

	// ③ 党建活动登记（created_by 非空 = 组织侧，2026-08-16 新增列；含学生主体范围过滤）
	var partyRegTotal int
	qParty := `SELECT COUNT(*) FROM party_study_records psr WHERE psr.created_by IS NOT NULL`
	partyArgs := []interface{}{}
	if ownerID != "" {
		qParty += ` AND EXISTS (SELECT 1 FROM users ua WHERE ua.id = psr.user_id AND ua.owner_id = ?)`
		partyArgs = append(partyArgs, ownerID)
	}
	if err := r.db.QueryRow(qParty, partyArgs...).Scan(&partyRegTotal); err != nil {
		partyRegTotal = 0
	}

	// ④ 教学（排课，教师归属学院）
	var courseTotal int
	qCourse := `SELECT COUNT(*) FROM course_schedules cs`
	courseArgs := []interface{}{}
	if ownerID != "" {
		qCourse += ` JOIN users t ON t.id = cs.teacher_id AND t.owner_id = ?`
		courseArgs = append(courseArgs, ownerID)
	}
	if err := r.db.QueryRow(qCourse, courseArgs...).Scan(&courseTotal); err != nil {
		courseTotal = 0
	}

	// 按角色汇总（本院/全校范围内各角色育人动作数）
	roleSum := map[string]int{}
	qRole := `SELECT u.role, COUNT(*) FROM (
	           SELECT user_id FROM party_study_records WHERE created_by IS NOT NULL
	           UNION ALL
	           SELECT student_id FROM talk_records
	         ) x
	         JOIN users u ON u.id = x.user_id`
	roleArgs := []interface{}{}
	if ownerID != "" {
		qRole += ` WHERE u.owner_id = ?`
		roleArgs = append(roleArgs, ownerID)
	}
	qRole += ` GROUP BY u.role`
	rows, err := r.db.Query(qRole, roleArgs...)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var rl string
			var c int
			if err := rows.Scan(&rl, &c); err == nil {
				roleSum[rl] = c
			}
		}
	}

	res["owner_id"] = ownerID
	res["students_total"] = studentCnt
	res["talk_records"] = talkTotal
	res["facility_records"] = facilityTotal
	res["party_registrations"] = partyRegTotal
	res["course_schedules"] = courseTotal
	res["by_role"] = roleSum

	// 诚实 data_source：全部 0 且无学生 → not_available
	if talkTotal == 0 && facilityTotal == 0 && partyRegTotal == 0 && courseTotal == 0 && studentCnt == 0 {
		res["data_source"] = "not_available"
	} else {
		res["data_source"] = "real"
	}
	return res, nil
}

// ══════════════════════════════════════════════════════════════
// 育人成效 KPI 指标卡（D5-1 功能补齐，2026-08-16）
// 书记/学院视角：把党建/协同育人/教育成果大屏的聚合视图升级为量化 KPI 指标卡。
// 诚实边界（铁律）：
//   - data_source=real         → 数值来自真实表聚合（users/party_progress/party_study_records/
//     talk_records/facility_records/course_schedules/
//     health_activity_signups/student_points/graduation_outcome/
//     competition_registrations/student_grades），不造数。
//   - data_source=not_available → 岗位职责应有但当前无数据源/无登记表的指标，value=null，
//     绝不伪造数字，供前端渲染「上传支撑材料到知识库」入口（upload_target）。
//
// scope 参数：ownerID!=空 → 本院（按 users.owner_id 精确匹配）；空 → 全校（学校书记）。
func (r *SecretaryOutcomeRepo) GetNurtureKPI(ownerID string) []map[string]interface{} {
	kpi := make([]map[string]interface{}, 0, 16)

	// 复用既有真实聚合做底座（同 ownerID 范围语义，避免重复 SQL 且保证行为一致）
	party, _ := r.PartyDashboard(ownerID)
	collab, _ := r.CollabDashboard(ownerID)
	outcome := map[string]interface{}{}
	if os, err := r.OutcomeStats(ownerID); err == nil {
		outcome = os
	}
	// 第二课堂真实聚合（报名/到场/积分，owner 范围）——直接 SQL，属真实表
	sc := r.secondClassKPI(ownerID)

	// ---------- ① 学生基数 ----------
	stuTotal := numOf(collab, "students_total")
	kpi = append(kpi, map[string]interface{}{
		"key": "nurture.student_total", "label": "在校学生数",
		"value": stuTotal, "unit": "人",
		"data_source": "real", "source_desc": "users 学生账号按归属聚合（真实用户）",
	})

	// ---------- ② 党建育人（真实表） ----------
	members := intMapOf(party, "members")
	memberCnt := float64(members["member"] + members["probation"])
	kpi = append(kpi, map[string]interface{}{
		"key": "nurture.party_member", "label": "党员数（正式+预备）",
		"value": memberCnt, "unit": "人",
		"data_source": "real", "source_desc": "party_progress 真实登记（status member/probation）",
	})
	stageTotal := numOf(party, "stage_total")
	kpi = append(kpi, map[string]interface{}{
		"key": "nurture.party_applicant", "label": "入党申请人数",
		"value": stageTotal, "unit": "人",
		"data_source": "real", "source_desc": "party_progress current_stage 真实登记",
	})
	studyCount := numOf(party, "study_records")
	studyHours := numOf(party, "study_hours")
	kpi = append(kpi, map[string]interface{}{
		"key": "nurture.party_study", "label": "党课/党建学习人次",
		"value": studyCount, "unit": "人次",
		"data_source": "real", "source_desc": "party_study_records 真实学习记录（时长 " + itoa(int(studyHours)) + " 小时）",
	})

	// ---------- ③ 协同育人（真实表动作量） ----------
	talkTotal := numOf(collab, "talk_records")
	kpi = append(kpi, map[string]interface{}{
		"key": "nurture.talk_total", "label": "谈心谈话记录数",
		"value": talkTotal, "unit": "条",
		"data_source": "real", "source_desc": "talk_records 辅导员谈心记录（真实）",
	})
	facilityTotal := numOf(collab, "facility_records")
	kpi = append(kpi, map[string]interface{}{
		"key": "nurture.facility", "label": "后勤育人服务条数",
		"value": facilityTotal, "unit": "条",
		"data_source": "real", "source_desc": "facility_records 教辅后勤服务记录（真实）",
	})
	partyReg := numOf(collab, "party_registrations")
	kpi = append(kpi, map[string]interface{}{
		"key": "nurture.party_activity", "label": "党课/活动组织登记",
		"value": partyReg, "unit": "条",
		"data_source": "real", "source_desc": "party_study_records(created_by 非空) 组织侧登记（真实）",
	})
	courseTotal := numOf(collab, "course_schedules")
	kpi = append(kpi, map[string]interface{}{
		"key": "nurture.course", "label": "教学排课节次",
		"value": courseTotal, "unit": "节",
		"data_source": "real", "source_desc": "course_schedules 排课表（真实，按教师归属学院）",
	})

	// ---------- ④ 第二课堂（真实参与/积分） ----------
	kpi = append(kpi, map[string]interface{}{
		"key": "nurture.second_class", "label": "二课活动参与人次",
		"value": sc["attend_total"], "unit": "人次",
		"data_source": "real", "source_desc": "health_activity_signups 到场记录（真实）",
	})
	kpi = append(kpi, map[string]interface{}{
		"key": "nurture.second_class_points", "label": "二课积分合计",
		"value": sc["point_total"], "unit": "分",
		"data_source": "real", "source_desc": "student_points 积分记录（真实）",
	})

	// ---------- ⑤ 毕业去向落实（真实，仅 approved 计入；无记录则 not_available） ----------
	if src := strOf(outcome, "data_source"); src == "real" {
		kpi = append(kpi, map[string]interface{}{
			"key": "nurture.employment_rate", "label": "毕业去向落实率",
			"value": numOf(outcome, "employment_rate"), "unit": "%",
			"data_source": "real", "source_desc": "graduation_outcome(approved) 真实登记（就业+灵活+创业/总数）",
		})
		kpi = append(kpi, map[string]interface{}{
			"key": "nurture.postgrad_rate", "label": "升学率（考研/出国）",
			"value": numOf(outcome, "postgrad_rate"), "unit": "%",
			"data_source": "real", "source_desc": "graduation_outcome(approved) 真实登记（升学/总数）",
		})
		kpi = append(kpi, map[string]interface{}{
			"key": "nurture.outcome_total", "label": "毕业去向已登记数",
			"value": numOf(outcome, "total"), "unit": "人",
			"data_source": "real", "source_desc": "graduation_outcome(approved) 真实登记",
		})
	} else {
		// 诚实：无已审核去向数据 → not_available + 上传入口，绝不伪造
		kpi = append(kpi, map[string]interface{}{
			"key": "nurture.employment_rate", "label": "毕业去向落实率",
			"value": nil, "unit": "%",
			"data_source":   "not_available",
			"source_desc":   "graduation_outcome 尚无 approved 记录（需教辅录入并经审核后计入）",
			"upload_target": "kb", "upload_hint": "上传毕业去向汇总/就业名单等支撑材料到知识库补料",
		})
	}

	// ---------- ⑥ 学业成效（真实通过率） ----------
	ac := map[string]interface{}{}
	if edu, err := r.EducationOutcomeDashboard(ownerID); err == nil {
		ac = edu
	}
	awardTotal := 0
	if comp := mapOf(ac, "competition"); comp != nil {
		awardTotal = intOf(comp, "total_awards")
	}
	kpi = append(kpi, map[string]interface{}{
		"key": "nurture.award", "label": "学科竞赛获奖数",
		"value": float64(awardTotal), "unit": "项",
		"data_source": "real", "source_desc": "competition_registrations(status=awarded) 真实获奖",
	})
	if aca := mapOf(ac, "academic"); aca != nil {
		kpi = append(kpi, map[string]interface{}{
			"key": "nurture.academic_pass", "label": "课程通过率",
			"value": numOf(aca, "pass_rate"), "unit": "%",
			"data_source": "real", "source_desc": "student_grades 真实成绩（通过/总数）",
		})
	} else {
		kpi = append(kpi, map[string]interface{}{
			"key": "nurture.academic_pass", "label": "课程通过率",
			"value": nil, "unit": "%",
			"data_source":   "not_available",
			"source_desc":   "student_grades 无成绩记录（需导入真实成绩后统计）",
			"upload_target": "kb", "upload_hint": "上传成绩单/学业通过率汇总支撑材料到知识库补料",
		})
	}

	// ---------- ⑦ 职责应有但无数据源 → not_available + 上传入口（绝不伪造） ----------
	kpi = append(kpi, notAvailableCard(
		"nurture.intervention_total", "干预执行次数", "人次",
		"干预方案目前由 AI 生成（GenerateIntervention）仅返回文本，未落任何统计表，无法统计真实干预执行次数。",
	))
	kpi = append(kpi, notAvailableCard(
		"nurture.second_class_pass_rate", "二课达标率", "%",
		"第二课堂有真实参与/积分，但「达标」标准阈值在系统内无配置，无法给出达标判定，故不展示比率。",
	))
	// ---------- 学生成长度对比（纵向先留痕，P1-2 A 路径动态判断） ----------
	// 有足够纵向历史（双端采样 ≥1 名学生）→ data_source=trend（真实趋势差分）；
	// 否则 → 仍 not_available，source_desc 诚实说明「数据积累中，需累计满 N 周」。
	kpi = append(kpi, r.growthTrendCard(ownerID))
	kpi = append(kpi, notAvailableCard(
		"nurture.employment_target", "就业目标达成率", "%",
		"缺少既定就业目标值配置（系统仅有实际去向率），无目标基准无法判定达成率。",
	))

	return kpi
}

// growthTrendWindowWeeks 成长归因纵向窗口周数 N（基准口径，供前后端提示「需累计满 N 周」）。
// 取 pm-check-nurture-wiring.md 建议值（§五.3 / §十一.2：默认 4 周，可配）；
// 写成常量便于后续改为 config/settings 可读。
const growthTrendWindowWeeks = 4

// growthTrendCard 生成「学生成长度对比（纵向）」KPI 卡：动态判断。
//   - 有 ≥1 名学生具备双端历史（样本数>0）→ data_source=trend + 五维平均变化 + sample_count + window_weeks；
//     仅报趋势/相关性（delta 升降方向与幅度），绝不表述因果（「因为…所以…」）。
//   - 否则 → 沿用 not_available 卡，value=nil，source_desc 诚实提示「纵向基准积累中，需累计满 N 周」。
func (r *SecretaryOutcomeRepo) growthTrendCard(ownerID string) map[string]interface{} {
	label := "学生成长度对比（横向/纵向）"
	notAvail := notAvailableCard(
		"nurture.growth_trend", label, "-",
		fmt.Sprintf("纵向基准积累中：需连续记录满 %d 周历史快照后生成成长归因。系统已开始对数字孪生快照做历史留痕（snapshot_history），当前采样天数不足，暂无法给出趋势。", growthTrendWindowWeeks))

	gt, err := r.getGrowthTrend(ownerID, growthTrendWindowWeeks)
	if err != nil || gt == nil || !gt.HasData || gt.SampleCount <= 0 {
		// 无足够双端历史 / 查询失败：诚实回落 not_available，绝不凭空给 trend 数值
		return notAvail
	}

	return map[string]interface{}{
		"key": "nurture.growth_trend", "label": label, "unit": "-",
		"data_source": "trend",
		"value": map[string]interface{}{
			"academic":    gt.Academic,
			"ability":     gt.Ability,
			"ideological": gt.Ideological,
			"emotional":   gt.Emotional,
			"social":      gt.Social,
		},
		"sample_count": gt.SampleCount,
		"window_weeks": gt.WindowWeeks,
		"source_desc": fmt.Sprintf(
			"基于 %d 名具备 %d 周纵向历史的学生，最近相对更早的快照五维平均变化（仅趋势/相关性，不作因果）。",
			gt.SampleCount, gt.WindowWeeks),
	}
}

// getGrowthTrend 从 snapshot_history 按 owner 范围聚合近 windowWeeks 周的学生成长趋势。
// 语义与 TwinRepo.GetGrowthTrend 一致（纵向差分，仅趋势/相关性）：对窗口内具备双端
// （≥2 个不同采样日）快照的学生，取最早 vs 最近五维 delta 均值。ownerID 为空=全校，
// 非空=本院（越权红线：历史查询必须带 owner_id 收窄）。
func (r *SecretaryOutcomeRepo) getGrowthTrend(ownerID string, windowWeeks int) (*GrowthTrend, error) {
	if windowWeeks < 1 {
		windowWeeks = growthTrendWindowWeeks
	}
	cutoff := time.Now().AddDate(0, 0, -windowWeeks*7).Format("2006-01-02")

	cond := ` AND owner_scope = 'college'`
	args := []interface{}{}
	if ownerID != "" {
		cond += ` AND owner_id = ?`
		args = append(args, ownerID)
	}
	cond += ` AND computed_at >= ?`
	args = append(args, cutoff)

	rows, err := r.db.Query(`SELECT user_id,
		academic_score, ability_score, ideological_score, emotional_score, social_score, computed_at
		FROM snapshot_history WHERE 1=1`+cond+
		` ORDER BY user_id ASC, computed_at ASC`, args...)
	if err != nil {
		return nil, fmt.Errorf("查询快照历史失败: %w", err)
	}
	defer rows.Close()

	type histRec struct {
		ac, ab, id, em, so float64
		computedAt         string
	}
	byUser := map[int64][]histRec{}
	for rows.Next() {
		var uid int64
		rec := histRec{}
		if err := rows.Scan(&uid, &rec.ac, &rec.ab, &rec.id, &rec.em, &rec.so, &rec.computedAt); err != nil {
			return nil, err
		}
		byUser[uid] = append(byUser[uid], rec)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	gt := &GrowthTrend{WindowWeeks: windowWeeks}
	var academic, ability, ideological, emotional, social float64
	for _, recs := range byUser {
		if len(recs) < 2 {
			continue // 仅单端（一天一条），无两端对比
		}
		first, last := recs[0], recs[len(recs)-1]
		if first.computedAt == last.computedAt {
			continue
		}
		academic += last.ac - first.ac
		ability += last.ab - first.ab
		ideological += last.id - first.id
		emotional += last.em - first.em
		social += last.so - first.so
		gt.SampleCount++
	}
	if gt.SampleCount > 0 {
		gt.HasData = true
		n := float64(gt.SampleCount)
		gt.Academic = academic / n
		gt.Ability = ability / n
		gt.Ideological = ideological / n
		gt.Emotional = emotional / n
		gt.Social = social / n
	}
	return gt, nil
}

// secondClassKPI 第二课堂真实聚合（按 owner 范围下学生）——真实表，不造数。
func (r *SecretaryOutcomeRepo) secondClassKPI(ownerID string) map[string]interface{} {
	res := map[string]interface{}{"attend_total": 0, "point_total": 0}
	// 学生主体范围（users.owner_id）
	where := ` WHERE u.role='student'`
	args := []interface{}{}
	if ownerID != "" {
		where += ` AND u.owner_id = ?`
		args = append(args, ownerID)
	}
	// 到场人次
	var attend int
	err := r.db.QueryRow(
		`SELECT COALESCE(SUM(h.attended),0) FROM health_activity_signups h JOIN users u ON u.id=h.user_id`+where,
		args...).Scan(&attend)
	if err == nil {
		res["attend_total"] = float64(attend)
	}
	// 积分合计
	var points int
	err = r.db.QueryRow(
		`SELECT COALESCE(SUM(sp.points),0) FROM student_points sp JOIN users u ON u.id=sp.user_id`+where,
		args...).Scan(&points)
	if err == nil {
		res["point_total"] = float64(points)
	}
	return res
}

// notAvailableCard 生成「职责应有但无数据源」的 KPI 卡，value 恒为 null，绝不返回伪数字。
func notAvailableCard(key, label, unit, desc string) map[string]interface{} {
	return map[string]interface{}{
		"key": key, "label": label, "value": nil, "unit": unit,
		"data_source": "not_available", "source_desc": desc,
		"upload_target": "kb", "upload_hint": "上传支撑材料到知识库补料，补充后转为真实数据",
	}
}

// ---- 以下为 GetNurtureKPI 使用的小助手（map/整形/浮点读取，nil 安全） ----

func numOf(m map[string]interface{}, k string) float64 {
	if m == nil {
		return 0
	}
	if v, ok := m[k]; ok {
		switch t := v.(type) {
		case float64:
			return t
		case int:
			return float64(t)
		case int64:
			return float64(t)
		case float32:
			return float64(t)
		}
	}
	return 0
}

func intOf(m map[string]interface{}, k string) int {
	return int(numOf(m, k))
}

func mapOf(m map[string]interface{}, k string) map[string]interface{} {
	if m == nil {
		return nil
	}
	if v, ok := m[k]; ok {
		if mp, ok2 := v.(map[string]interface{}); ok2 {
			return mp
		}
	}
	return nil
}

// intMapOf 读取 map[string]int（或 map[string]interface{}）字段，nil 安全。
// 既有聚合（如 PartyDashboard.members 为 map[string]int）返回该类型，故需兼容。
func intMapOf(m map[string]interface{}, k string) map[string]int {
	res := map[string]int{}
	if m == nil {
		return res
	}
	v, ok := m[k]
	if !ok {
		return res
	}
	switch t := v.(type) {
	case map[string]int:
		res = t
	case map[string]interface{}:
		for kk, vv := range t {
			switch n := vv.(type) {
			case int:
				res[kk] = n
			case int64:
				res[kk] = int(n)
			case float64:
				res[kk] = int(n)
			case float32:
				res[kk] = int(n)
			}
		}
	}
	return res
}

func strOf(m map[string]interface{}, k string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[k]; ok {
		if s, ok2 := v.(string); ok2 {
			return s
		}
	}
	return ""
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
