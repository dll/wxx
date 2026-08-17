package repository

import (
	"database/sql"
	"fmt"
	"time"

	dbutil "github.com/dll/wxx/server/internal/db"
	"github.com/dll/wxx/server/internal/model"
)

// GovTicketRepo 督办工单数据访问层（D5-3「洞察→工单」治理回环）
// 对应迁移 090_gov_tickets 表；单表 + 轻量操作日志，复用 feedback/process 的流转心智。
type GovTicketRepo struct {
	db    *sql.DB
	mysql bool
}

// NewGovTicketRepo 创建督办工单 repo
func NewGovTicketRepo(db *sql.DB) *GovTicketRepo {
	return &GovTicketRepo{db: db, mysql: dbutil.IsMySQL(db)}
}

// GovTicketCreateReq 创建督办工单请求（service 层入参，跨 handler→service→repo 边界）
// CreatedByName 仅用于写操作日志，非持久化字段。
type GovTicketCreateReq struct {
	Title         string `json:"title"`
	Category      string `json:"category"`      // insight / supplement
	SourceType    string `json:"source_type"`   // insight / kpi
	SourceKey     string `json:"source_key"`    // 来源主键（如 KPI key）
	SourceDesc    string `json:"source_desc"`   // 来源描述
	DataSource    string `json:"data_source"`   // real / not_available
	Priority      string `json:"priority"`      // low / normal / high
	Status        string `json:"status"`        // pending/processing/completed/closed
	College       string `json:"college"`       // 空=全校，非空=本院
	AssigneeRole  string `json:"assignee_role"` // counselor/teacher/assistant/party
	AssigneeID    int64  `json:"assignee_id"`
	AssigneeName  string `json:"assignee_name"`
	Deadline      string `json:"deadline"`
	Remark        string `json:"remark"`
	CreatedBy     int64  `json:"created_by"`
	CreatedByRole string `json:"created_by_role"`
	CreatedByName string `json:"created_by_name"`
}

const govTicketCols = `id, ticket_no, title, category, source_type, source_key, source_desc,
 data_source, priority, status, college, assignee_role, assignee_id, assignee_name,
 deadline, remark, created_by, created_by_role, closed_by, created_at, updated_at, closed_at`

func scanGovTicket(row interface{ Scan(...interface{}) error }, t *model.GovTicket) error {
	return row.Scan(&t.ID, &t.TicketNo, &t.Title, &t.Category, &t.SourceType, &t.SourceKey,
		&t.SourceDesc, &t.DataSource, &t.Priority, &t.Status, &t.College, &t.AssigneeRole,
		&t.AssigneeID, &t.AssigneeName, &t.Deadline, &t.Remark, &t.CreatedBy, &t.CreatedByRole,
		&t.ClosedBy, &t.CreatedAt, &t.UpdatedAt, &t.ClosedAt)
}

func scanGovTicketRow(row *sql.Row, t *model.GovTicket) error {
	return row.Scan(&t.ID, &t.TicketNo, &t.Title, &t.Category, &t.SourceType, &t.SourceKey,
		&t.SourceDesc, &t.DataSource, &t.Priority, &t.Status, &t.College, &t.AssigneeRole,
		&t.AssigneeID, &t.AssigneeName, &t.Deadline, &t.Remark, &t.CreatedBy, &t.CreatedByRole,
		&t.ClosedBy, &t.CreatedAt, &t.UpdatedAt, &t.ClosedAt)
}

// Create 创建督办工单，返回新工单 id
func (r *GovTicketRepo) Create(t *model.GovTicket) (int64, error) {
	if t.TicketNo == "" {
		t.TicketNo = fmt.Sprintf("GT-%s-%d", time.Now().Format("20060102"), time.Now().UnixMilli()%100000)
	}
	if t.Category == "" {
		t.Category = "insight"
	}
	if t.SourceType == "" {
		t.SourceType = "insight"
	}
	if t.Priority == "" {
		t.Priority = "normal"
	}
	if t.Status == "" {
		t.Status = "pending"
	}
	if t.DataSource == "" {
		t.DataSource = "not_available"
	}
	result, err := r.db.Exec(
		`INSERT INTO gov_tickets (ticket_no, title, category, source_type, source_key, source_desc,
		 data_source, priority, status, college, assignee_role, assignee_id, assignee_name,
		 deadline, remark, created_by, created_by_role)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		t.TicketNo, t.Title, t.Category, t.SourceType, t.SourceKey, t.SourceDesc,
		t.DataSource, t.Priority, t.Status, t.College, t.AssigneeRole, t.AssigneeID, t.AssigneeName,
		t.Deadline, t.Remark, t.CreatedBy, t.CreatedByRole,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// GetByID 按 ID 查询督办工单
func (r *GovTicketRepo) GetByID(id int64) (*model.GovTicket, error) {
	t := &model.GovTicket{}
	err := scanGovTicketRow(r.db.QueryRow(`SELECT `+govTicketCols+` FROM gov_tickets WHERE id=?`, id), t)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return t, nil
}

// List 分页查询督办工单。
// assigneeID>0 时限定本人分派（责任人视角）；status 空=全部。
func (r *GovTicketRepo) List(status, college, category string, assigneeID int64, offset, limit int) ([]*model.GovTicket, int, error) {
	queryBase := `FROM gov_tickets WHERE 1=1`
	var args []interface{}
	if status != "" {
		queryBase += ` AND status = ?`
		args = append(args, status)
	}
	if college != "" {
		queryBase += ` AND college = ?`
		args = append(args, college)
	}
	if category != "" {
		queryBase += ` AND category = ?`
		args = append(args, category)
	}
	if assigneeID > 0 {
		queryBase += ` AND assignee_id = ?`
		args = append(args, assigneeID)
	}

	var count int
	if err := r.db.QueryRow(`SELECT COUNT(*) `+queryBase, args...).Scan(&count); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(`SELECT `+govTicketCols+` `+queryBase+` ORDER BY id DESC LIMIT ? OFFSET ?`,
		append(args, limit, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []*model.GovTicket
	for rows.Next() {
		t := &model.GovTicket{}
		if err := scanGovTicket(rows, t); err != nil {
			return nil, 0, err
		}
		items = append(items, t)
	}
	return items, count, rows.Err()
}

// CountByStatus 按状态统计（书记督办总览）
func (r *GovTicketRepo) CountByStatus(college string) (map[string]int, error) {
	where := ` WHERE 1=1`
	var args []interface{}
	if college != "" {
		where += ` AND college = ?`
		args = append(args, college)
	}
	rows, err := r.db.Query(`SELECT status, COUNT(*) FROM gov_tickets`+where+` GROUP BY status`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[string]int{"pending": 0, "processing": 0, "completed": 0, "closed": 0}
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err == nil {
			result[status] = count
		}
	}
	return result, rows.Err()
}

// Assign 分派/改派责任人（书记）
func (r *GovTicketRepo) Assign(id, assigneeID int64, assigneeRole, assigneeName, deadline string, byUser int64, byName string) error {
	_, err := r.db.Exec(
		`UPDATE gov_tickets SET assignee_id=?, assignee_role=?, assignee_name=?, deadline=?,
		 updated_at=CURRENT_TIMESTAMP WHERE id=?`,
		assigneeID, assigneeRole, assigneeName, deadline, id,
	)
	if err != nil {
		return err
	}
	return r.AddLog(id, "assigned", byUser, byName, fmt.Sprintf("分派给 %s(%s)", assigneeName, assigneeRole))
}

// UpdateStatus 状态流转：pending->processing->completed/closed。
// 完成后强制待办任务推进；关闭记录 closed_by/closed_at。
func (r *GovTicketRepo) UpdateStatus(id, opID int64, opName, status, detail string) error {
	var err error
	if status == "completed" || status == "closed" {
		_, err = r.db.Exec(
			`UPDATE gov_tickets SET status=?, closed_by=?, closed_at=CURRENT_TIMESTAMP,
			 updated_at=CURRENT_TIMESTAMP WHERE id=?`,
			status, opID, id,
		)
	} else {
		_, err = r.db.Exec(
			`UPDATE gov_tickets SET status=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
			status, id,
		)
	}
	if err != nil {
		return err
	}
	return r.AddLog(id, status, opID, opName, detail)
}

// AddLog 追加工单操作记录
func (r *GovTicketRepo) AddLog(ticketID int64, action string, opID int64, opName, detail string) error {
	_, err := r.db.Exec(
		`INSERT INTO gov_ticket_logs (ticket_id, action, operator_id, operator_name, detail) VALUES (?,?,?,?,?)`,
		ticketID, action, opID, opName, detail,
	)
	return err
}

// ListLogs 获取工单操作记录
func (r *GovTicketRepo) ListLogs(ticketID int64) ([]*model.GovTicketLog, error) {
	rows, err := r.db.Query(
		`SELECT id, ticket_id, action, operator_id, operator_name, detail, created_at
		 FROM gov_ticket_logs WHERE ticket_id=? ORDER BY id ASC`, ticketID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*model.GovTicketLog
	for rows.Next() {
		l := &model.GovTicketLog{}
		if err := rows.Scan(&l.ID, &l.TicketID, &l.Action, &l.OperatorID, &l.OperatorName, &l.Detail, &l.CreatedAt); err == nil {
			items = append(items, l)
		}
	}
	return items, rows.Err()
}
