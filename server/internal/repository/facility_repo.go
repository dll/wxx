package repository

import (
	"database/sql"
	"time"
)

// FacilityRecord 后勤服务记录（真实登记数据）
// 对应迁移 086_facility_records 表。
type FacilityRecord struct {
	ID           int64     `json:"id"`
	Role         string    `json:"role"`        // 岗位类型: lab/clean/hotwater/dorm/envir/library
	Title        string    `json:"title"`       // 事项简述
	Location     string    `json:"location"`    // 地点
	Detail       string    `json:"detail"`      // 详情/数量/备注
	OperatorID   int64     `json:"operator_id"` // 登记人
	OperatorName string    `json:"operator_name"`
	StudentID    int64     `json:"student_id"` // 关联学生(0=无)
	StudentName  string    `json:"student_name"`
	OccurredAt   string    `json:"occurred_at"` // 服务发生时间(ISO)
	CreatedAt    time.Time `json:"created_at"`
	DataSource   string    `json:"data_source"` // 固定 real
}

// FacilityRoleMeta 岗位类型元信息（供前端下拉/看板展示）
var FacilityRoleMeta = map[string]string{
	"lab":      "实验室开门/关门",
	"clean":    "教室保洁卫生",
	"hotwater": "热水供应",
	"dorm":     "宿舍晚归查岗",
	"envir":    "校园环卫学习环境",
	"library":  "图书馆借阅管理",
}

// FacilityRepo 后勤服务记录数据访问层
type FacilityRepo struct {
	db *sql.DB
}

func NewFacilityRepo(db *sql.DB) *FacilityRepo {
	return &FacilityRepo{db: db}
}

// Create 登记一条后勤服务记录（真实数据）
func (r *FacilityRepo) Create(rec *FacilityRecord) (int64, error) {
	res, err := r.db.Exec(
		`INSERT INTO facility_records (role, title, location, detail, operator_id, operator_name, student_id, student_name, occurred_at, data_source)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rec.Role, rec.Title, rec.Location, rec.Detail,
		rec.OperatorID, rec.OperatorName, rec.StudentID, rec.StudentName,
		rec.OccurredAt, "real",
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// List 查询后勤服务记录，支持按 岗位/操作人/学生/时间 过滤。
// operatorID>0 时限定本人；否则需有权限的人查询全部。
func (r *FacilityRepo) List(role, operatorName string, studentID int64, from, to string, limit int) ([]FacilityRecord, error) {
	q := `SELECT id, role, title, location, detail, operator_id, operator_name, student_id, student_name, occurred_at, created_at, data_source
	      FROM facility_records WHERE 1=1`
	args := []interface{}{}
	if role != "" {
		q += ` AND role = ?`
		args = append(args, role)
	}
	if operatorName != "" {
		q += ` AND operator_name LIKE ?`
		args = append(args, "%"+operatorName+"%")
	}
	if studentID > 0 {
		q += ` AND student_id = ?`
		args = append(args, studentID)
	}
	if from != "" {
		q += ` AND occurred_at >= ?`
		args = append(args, from)
	}
	if to != "" {
		q += ` AND occurred_at <= ?`
		args = append(args, to)
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q += ` ORDER BY occurred_at DESC LIMIT ?`
	args = append(args, limit)

	rows, err := r.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []FacilityRecord
	for rows.Next() {
		var rv FacilityRecord
		var created string
		if err := rows.Scan(&rv.ID, &rv.Role, &rv.Title, &rv.Location, &rv.Detail,
			&rv.OperatorID, &rv.OperatorName, &rv.StudentID, &rv.StudentName,
			&rv.OccurredAt, &created, &rv.DataSource); err != nil {
			return nil, err
		}
		rv.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", created)
		out = append(out, rv)
	}
	return out, rows.Err()
}

// Dashboard 后勤台看板汇总：按时段统计各岗位服务次数 + 服务学生去重数 + 总服务量
func (r *FacilityRepo) Dashboard(operatorID int64, from, to string) (map[string]int, int, int, error) {
	base := ` FROM facility_records WHERE 1=1`
	args := []interface{}{}
	if operatorID > 0 {
		base += ` AND operator_id = ?`
		args = append(args, operatorID)
	}
	if from != "" {
		base += ` AND occurred_at >= ?`
		args = append(args, from)
	}
	if to != "" {
		base += ` AND occurred_at <= ?`
		args = append(args, to)
	}

	// 各岗位服务次数
	rows, err := r.db.Query(`SELECT role, COUNT(*)`+base+` GROUP BY role`, args...)
	if err != nil {
		return nil, 0, 0, err
	}
	defer rows.Close()
	byRole := map[string]int{}
	for rows.Next() {
		var role string
		var cnt int
		if err := rows.Scan(&role, &cnt); err != nil {
			return nil, 0, 0, err
		}
		byRole[role] = cnt
	}
	if err := rows.Err(); err != nil {
		return nil, 0, 0, err
	}

	// 总服务量
	var total int
	if err := r.db.QueryRow(`SELECT COUNT(*)`+base, args...).Scan(&total); err != nil {
		return nil, 0, 0, err
	}
	// 关联学生去重数（有学生关联的记录）
	var stuCnt int
	err = r.db.QueryRow(`SELECT COUNT(DISTINCT student_id)`+base+` AND student_id > 0`, args...).Scan(&stuCnt)
	if err != nil {
		stuCnt = 0
	}
	return byRole, total, stuCnt, nil
}
