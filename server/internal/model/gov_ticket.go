package model

// GovTicket 督办工单（D5-3「洞察→工单」治理回环）
// 对应迁移 090_gov_tickets 表。
// 由学院/学校书记从治理洞察（如育人 KPI 的 not_available/稀疏指标）一键生成，
// 分派给辅导员/教辅/党群责任人，状态流转：pending->processing->completed/closed。
// data_source 沿用洞察端语义（real/not_available），记录工单来源是否已有真实数据。
type GovTicket struct {
	ID            int64   `json:"id" db:"id"`
	TicketNo      string  `json:"ticket_no" db:"ticket_no"`
	Title         string  `json:"title" db:"title"`
	Category      string  `json:"category" db:"category"`       // insight=治理洞察 / supplement=补料督办（D5-1 联动）
	SourceType    string  `json:"source_type" db:"source_type"` // kpi / insight
	SourceKey     string  `json:"source_key" db:"source_key"`   // 来源主键（如 KPI key）
	SourceDesc    string  `json:"source_desc" db:"source_desc"` // 来源描述
	DataSource    string  `json:"data_source" db:"data_source"` // real / not_available / synthetic
	Priority      string  `json:"priority" db:"priority"`       // low / normal / high
	Status        string  `json:"status" db:"status"`           // pending / processing / completed / closed
	College       string  `json:"college" db:"college"`         // 空=全校，非空=本院
	AssigneeRole  string  `json:"assignee_role" db:"assignee_role"`
	AssigneeID    int64   `json:"assignee_id" db:"assignee_id"`
	AssigneeName  string  `json:"assignee_name" db:"assignee_name"`
	Deadline      string  `json:"deadline" db:"deadline"`
	Remark        string  `json:"remark" db:"remark"`
	CreatedBy     int64   `json:"created_by" db:"created_by"`
	CreatedByRole string  `json:"created_by_role" db:"created_by_role"`
	ClosedBy      int64   `json:"closed_by" db:"closed_by"`
	CreatedAt     string  `json:"created_at" db:"created_at"`
	UpdatedAt     string  `json:"updated_at" db:"updated_at"`
	ClosedAt      *string `json:"closed_at" db:"closed_at"`
}

// GovTicketLog 督办工单操作记录，对应 gov_ticket_logs 表
type GovTicketLog struct {
	ID           int64  `json:"id" db:"id"`
	TicketID     int64  `json:"ticket_id" db:"ticket_id"`
	Action       string `json:"action" db:"action"`
	OperatorID   int64  `json:"operator_id" db:"operator_id"`
	OperatorName string `json:"operator_name" db:"operator_name"`
	Detail       string `json:"detail" db:"detail"`
	CreatedAt    string `json:"created_at" db:"created_at"`
}
