package repository

import (
	"database/sql"
	"fmt"
)

// SecondClassStudent 班级第二课堂看板中单个学生的参与统计（真实聚合）
type SecondClassStudent struct {
	UserID        int64  `json:"user_id"`
	StudentID     string `json:"student_id"`
	Name          string `json:"name"`
	College       string `json:"college"`
	Major         string `json:"major"`
	ClassName     string `json:"class_name"`
	ActivityCount int    `json:"activity_count"` // 报名活动数
	AttendCount   int    `json:"attend_count"`   // 实际到场数
	PointTotal    int    `json:"point_total"`    // 积分合计
	PointSource   string `json:"point_source"`   // 积分来源说明（如 reason 去重）
}

// SecondClassBoard 辅导员第二课堂看板聚合结果
type SecondClassBoard struct {
	College       string               `json:"college"`
	Students      []SecondClassStudent `json:"students"`
	StudentTotal  int                  `json:"student_total"`
	ActivityTotal int                  `json:"activity_total"` // 名下学生报名活动去重数
	AttendTotal   int                  `json:"attend_total"`   // 名下学生到场总人次
	PointTotal    int                  `json:"point_total"`    // 名下学生积分合计
	DataSource    string               `json:"data_source"`    // real / not_available
	Note          string               `json:"note"`
}

// SecondClassRepo 第二课堂数据访问层（辅导员侧·班级看板）
// 数据源：users（名下学生）+ health_activity_signups（报名/到场）+ health_activities（活动）
//   - student_points（积分）—— 全部真实表聚合，不造数。无记录时诚实返回 not_available。
type SecondClassRepo struct {
	db *sql.DB
}

func NewSecondClassRepo(db *sql.DB) *SecondClassRepo {
	return &SecondClassRepo{db: db}
}

// ClassSecondClassBoard 按辅导员归属范围（owner_scope/owner_id）聚合并下学生的第二课堂参与情况。
// ownerScope/ownerID 来自登录辅导员 userCtx，范围锁定防越权（与 UserRepo.List 同构）。
func (r *SecondClassRepo) ClassSecondClassBoard(ownerScope, ownerID string) (*SecondClassBoard, error) {
	// ① 拉名下学生（与学生名单同范围，避免越权）
	where := ` WHERE role = 'student'`
	args := []interface{}{}
	if ownerScope != "" {
		where += ` AND owner_scope = ?`
		args = append(args, ownerScope)
	}
	if ownerID != "" {
		where += ` AND owner_id = ?`
		args = append(args, ownerID)
	}
	rows, err := r.db.Query(
		`SELECT id, username, COALESCE(display_name,''), COALESCE(college,''), COALESCE(major,''), COALESCE(class_name,'')
		 FROM users`+where+` ORDER BY id ASC LIMIT 500`, args...)
	if err != nil {
		return nil, fmt.Errorf("查询名下学生: %w", err)
	}
	type stu struct {
		id, pointTotal                            int64
		username, name, college, major, className string
	}
	var list []stu
	for rows.Next() {
		var s stu
		if err := rows.Scan(&s.id, &s.username, &s.name, &s.college, &s.major, &s.className); err != nil {
			rows.Close()
			return nil, err
		}
		s.pointTotal = 0
		list = append(list, s)
	}
	rows.Close()

	if len(list) == 0 {
		// 名下无学生（数据归属可能未对齐）——诚实返回空，不造名单
		return &SecondClassBoard{
			DataSource: "not_available",
			Note:       "名下暂无学生（学生归属范围与账号未对齐时显示为空）",
		}, nil
	}

	// ② 聚合每个学生的报名/到场（health_activity_signups）
	signupCount := map[int64]int{}             // user_id -> 报名数
	attendCount := map[int64]int{}             // user_id -> 到场数
	activitySet := map[int64]map[string]bool{} // user_id -> activity_id 去重
	idArgs := make([]interface{}, 0, len(list))
	for _, s := range list {
		idArgs = append(idArgs, s.id)
	}
	ph := buildPlaceholders(len(list))
	rows, err = r.db.Query(
		`SELECT user_id, activity_id, status, COALESCE(attended,0) FROM health_activity_signups
		 WHERE user_id IN (`+ph+`)`, idArgs...)
	if err == nil {
		for rows.Next() {
			var uid int64
			var actID, status string
			var attended int64
			if err := rows.Scan(&uid, &actID, &status, &attended); err != nil {
				rows.Close()
				return nil, err
			}
			signupCount[uid]++
			if activitySet[uid] == nil {
				activitySet[uid] = map[string]bool{}
			}
			if actID != "" {
				activitySet[uid][actID] = true
			}
			if attended > 0 {
				attendCount[uid]++
			}
		}
		rows.Close()
	}

	// ③ 聚合积分（student_points，source 注明）
	pointSource := map[int64]string{}
	rows, err = r.db.Query(
		`SELECT user_id, COALESCE(SUM(points),0), COALESCE(MAX(reason),'') FROM student_points
		 WHERE user_id IN (`+ph+`) GROUP BY user_id`, idArgs...)
	if err == nil {
		for rows.Next() {
			var uid, total int64
			var reason string
			if err := rows.Scan(&uid, &total, &reason); err != nil {
				rows.Close()
				return nil, err
			}
			for i := range list {
				if list[i].id == uid {
					list[i].pointTotal = total
					pointSource[uid] = reason
					break
				}
			}
		}
		rows.Close()
	}

	// ④ 组装结果
	students := make([]SecondClassStudent, 0, len(list))
	actTotal := 0
	attendTotal := 0
	pointTotal := 0
	for _, s := range list {
		actCnt := len(activitySet[s.id])
		attCnt := attendCount[s.id]
		pt := s.pointTotal
		actTotal += actCnt
		attendTotal += attCnt
		pointTotal += int(pt)
		students = append(students, SecondClassStudent{
			UserID:        s.id,
			StudentID:     s.username,
			Name:          s.name,
			College:       s.college,
			Major:         s.major,
			ClassName:     s.className,
			ActivityCount: actCnt,
			AttendCount:   attCnt,
			PointTotal:    int(pt),
			PointSource:   pointSource[s.id],
		})
	}

	return &SecondClassBoard{
		College:       ownerID,
		Students:      students,
		StudentTotal:  len(students),
		ActivityTotal: actTotal,
		AttendTotal:   attendTotal,
		PointTotal:    pointTotal,
		DataSource:    "real",
	}, nil
}

// buildPlaceholders 生成 (?,?,...) 占位符串，供 IN 子句使用
func buildPlaceholders(n int) string {
	if n <= 0 {
		return ""
	}
	s := ""
	for i := 0; i < n; i++ {
		if i > 0 {
			s += ","
		}
		s += "?"
	}
	return s
}
