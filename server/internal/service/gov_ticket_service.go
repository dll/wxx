package service

import (
	"context"
	"fmt"
	"log"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
)

// 督办工单状态常量
const (
	TicketStatusPending    = "pending"    // 待办
	TicketStatusProcessing = "processing" // 处理中
	TicketStatusCompleted  = "completed"  // 完成
	TicketStatusClosed     = "closed"     // 关闭
)

var validTicketStatuses = map[string]bool{
	TicketStatusPending:    true,
	TicketStatusProcessing: true,
	TicketStatusCompleted:  true,
	TicketStatusClosed:     true,
}

// 工单类别
const (
	TicketCategoryInsight    = "insight"    // 治理洞察督办
	TicketCategorySupplement = "supplement" // 补料督办（D5-1 联动：催办上传材料到知识库）
)

var validTicketCategories = map[string]bool{
	TicketCategoryInsight:    true,
	TicketCategorySupplement: true,
}

var validPriorities = map[string]bool{"low": true, "normal": true, "high": true}

// GovTicketService 督办工单服务（D5-3「洞察→工单」治理回环）
// 复用 feedback/outcome 的流转与持久化心智，最小实现：单表 + 轻量操作日志。
// 诚实边界：工单从真实洞察/KPI 生成，不伪造；KPI 补料工单仅在
// data_source=not_available 且 upload_target=kb 的真实缺失指标上生成。
type GovTicketService struct {
	repo *repository.GovTicketRepo
	kpi  *repository.SecretaryOutcomeRepo // D5-1 联动：读取育人 KPI 卡获取真实缺失指标
}

// NewGovTicketService 创建督办工单服务
func NewGovTicketService(repo *repository.GovTicketRepo, kpi *repository.SecretaryOutcomeRepo) *GovTicketService {
	return &GovTicketService{repo: repo, kpi: kpi}
}

// addLog 追加工单操作日志（统一入口）。写日志失败仅告警不回滚主流程，
// 但必须留痕可排障：督办工单的状态流转依赖日志可追溯。
func (s *GovTicketService) addLog(ticketID int64, action string, opID int64, opName, detail string) {
	if err := s.repo.AddLog(ticketID, action, opID, opName, detail); err != nil {
		log.Printf("[WARN] 追加工单日志失败 ticket_id=%d action=%s: %v", ticketID, action, err)
	}
}

// CreateTicket 创建督办工单（治理洞察类）。
// college 空=全校（school_admin），非空=本院。
func (s *GovTicketService) CreateTicket(ctx context.Context, req *repository.GovTicketCreateReq) (int64, error) {
	if req == nil {
		return 0, fmt.Errorf("工单请求为空")
	}
	if req.Title == "" {
		return 0, fmt.Errorf("督办标题不能为空")
	}
	if req.Category == "" {
		req.Category = TicketCategoryInsight
	}
	if !validTicketCategories[req.Category] {
		return 0, fmt.Errorf("未知工单类别: %s", req.Category)
	}
	if req.SourceType == "" {
		req.SourceType = "insight"
	}
	if req.Priority == "" {
		req.Priority = "normal"
	}
	if !validPriorities[req.Priority] {
		return 0, fmt.Errorf("未知优先级: %s", req.Priority)
	}
	if req.Status == "" {
		req.Status = TicketStatusPending
	}
	if req.SourceDesc != "" && req.DataSource == "" {
		// 缺省沿用洞察端诚实语义：来源描述作为洞察依据即视为 not_available 补料
		req.DataSource = "not_available"
	}
	t := &model.GovTicket{
		Title:         req.Title,
		Category:      req.Category,
		SourceType:    req.SourceType,
		SourceKey:     req.SourceKey,
		SourceDesc:    req.SourceDesc,
		DataSource:    req.DataSource,
		Priority:      req.Priority,
		Status:        req.Status,
		College:       req.College,
		AssigneeRole:  req.AssigneeRole,
		AssigneeID:    req.AssigneeID,
		AssigneeName:  req.AssigneeName,
		Deadline:      req.Deadline,
		Remark:        req.Remark,
		CreatedBy:     req.CreatedBy,
		CreatedByRole: req.CreatedByRole,
	}
	id, err := s.repo.Create(t)
	if err != nil {
		return 0, err
	}
	s.addLog(id, "created", req.CreatedBy, req.CreatedByName, "创建督办工单")
	// 若创建时即带分派对象，直接记为分派
	if req.AssigneeID > 0 && req.AssigneeName != "" {
		s.addLog(id, "assigned", req.CreatedBy, req.CreatedByName,
			fmt.Sprintf("分派给 %s(%s)", req.AssigneeName, req.AssigneeRole))
	}
	return id, nil
}

// CreateFromKPI 从育人 KPI 指标生成补料督办工单（D5-1 联动，关键价值点）。
// 仅当指标为真实缺失（data_source=not_available 且 upload_target=kb）时生成，
// 绝不从伪造/已 available 的指标生成工单。
// ownerID 空=全校，非空=本院（与 GetNurtureKPI 同范围语义）。
func (s *GovTicketService) CreateFromKPI(ctx context.Context, kpiKey, ownerID string, req *repository.GovTicketCreateReq) (int64, error) {
	if s.kpi == nil {
		return 0, fmt.Errorf("育人 KPI 数据源未就绪")
	}
	if kpiKey == "" {
		return 0, fmt.Errorf("请指定要生成工单的 KPI 指标")
	}
	kpis := s.kpi.GetNurtureKPI(ownerID)
	var card map[string]interface{}
	for _, k := range kpis {
		if k["key"] == kpiKey {
			card = k
			break
		}
	}
	if card == nil {
		return 0, fmt.Errorf("未找到该 KPI 指标: %s", kpiKey)
	}
	// 诚实边界：只有 not_available 的补料类指标才允许生成补料工单
	if card["data_source"] != "not_available" || card["upload_target"] != "kb" {
		return 0, fmt.Errorf("该指标已有真实数据或非补料类，无需生成补料工单")
	}

	label, _ := card["label"].(string)
	sourceDesc, _ := card["source_desc"].(string)
	hint, _ := card["upload_hint"].(string)
	title := "【补料督办】" + label
	remark := sourceDesc
	if hint != "" {
		remark = sourceDesc + "。" + hint
	}

	if req == nil {
		req = &repository.GovTicketCreateReq{}
	}
	req.Category = TicketCategorySupplement
	req.SourceType = "kpi"
	req.SourceKey = kpiKey
	req.SourceDesc = sourceDesc
	req.DataSource = "not_available"
	req.Title = title
	req.Remark = remark
	req.Status = TicketStatusPending
	req.College = ownerID // 工单归属同步为 KPI 查询范围（全校/本院）

	id, err := s.repo.Create(reqToModel(req))
	if err != nil {
		return 0, err
	}
	s.addLog(id, "created", req.CreatedBy, req.CreatedByName,
		fmt.Sprintf("从育人 KPI %s(%s) 生成补料督办工单", kpiKey, label))
	if req.AssigneeID > 0 && req.AssigneeName != "" {
		s.addLog(id, "assigned", req.CreatedBy, req.CreatedByName,
			fmt.Sprintf("分派给 %s(%s)", req.AssigneeName, req.AssigneeRole))
	}
	return id, nil
}

func reqToModel(req *repository.GovTicketCreateReq) *model.GovTicket {
	if req.SourceType == "" {
		req.SourceType = "insight"
	}
	if req.DataSource == "" {
		req.DataSource = "not_available"
	}
	return &model.GovTicket{
		Title:         req.Title,
		Category:      req.Category,
		SourceType:    req.SourceType,
		SourceKey:     req.SourceKey,
		SourceDesc:    req.SourceDesc,
		DataSource:    req.DataSource,
		Priority:      req.Priority,
		Status:        req.Status,
		College:       req.College,
		AssigneeRole:  req.AssigneeRole,
		AssigneeID:    req.AssigneeID,
		AssigneeName:  req.AssigneeName,
		Deadline:      req.Deadline,
		Remark:        req.Remark,
		CreatedBy:     req.CreatedBy,
		CreatedByRole: req.CreatedByRole,
	}
}

// Assign 分派/改派责任人
func (s *GovTicketService) Assign(ctx context.Context, id, assigneeID int64, assigneeRole, assigneeName, deadline string, opID int64, opName string) error {
	t, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if t == nil {
		return fmt.Errorf("督办工单不存在")
	}
	if t.Status == TicketStatusCompleted || t.Status == TicketStatusClosed {
		return fmt.Errorf("工单已完结，无法再分派")
	}
	if assigneeID <= 0 || assigneeName == "" {
		return fmt.Errorf("分派对象不能为空")
	}
	return s.repo.Assign(id, assigneeID, assigneeRole, assigneeName, deadline, opID, opName)
}

// UpdateStatus 更新工单状态：pending->processing->completed/closed。
// 非完结态允许任意合法流转，完结态（completed/closed）不允许回退。
func (s *GovTicketService) UpdateStatus(ctx context.Context, id, opID int64, opName, status, detail string) error {
	if !validTicketStatuses[status] {
		return fmt.Errorf("未知工单状态: %s", status)
	}
	t, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if t == nil {
		return fmt.Errorf("督办工单不存在")
	}
	if (t.Status == TicketStatusCompleted || t.Status == TicketStatusClosed) && status != t.Status {
		return fmt.Errorf("工单已完结，状态不可变更")
	}
	return s.repo.UpdateStatus(id, opID, opName, status, detail)
}

// Get 获取工单详情（含操作记录）
func (s *GovTicketService) Get(ctx context.Context, id, viewerID int64, isManager bool) (*model.GovTicket, []*model.GovTicketLog, error) {
	t, err := s.repo.GetByID(id)
	if err != nil {
		return nil, nil, err
	}
	if t == nil {
		return nil, nil, fmt.Errorf("督办工单不存在")
	}
	// 非管理端仅能查看分派给本人的工单（责任人视图）
	if !isManager && t.AssigneeID != viewerID {
		return nil, nil, fmt.Errorf("无权查看该督办工单")
	}
	logs, err := s.repo.ListLogs(id)
	if err != nil {
		return nil, nil, err
	}
	if logs == nil {
		logs = []*model.GovTicketLog{}
	}
	return t, logs, nil
}

// List 列出督办工单（书记/学院管理端）
func (s *GovTicketService) List(ctx context.Context, status, college, category string, offset, limit int) ([]*model.GovTicket, int, error) {
	return s.repo.List(status, college, category, 0, offset, limit)
}

// ListMine 列出分派给本人的督办工单（责任人视图）
func (s *GovTicketService) ListMine(ctx context.Context, assigneeID int64, status string, offset, limit int) ([]*model.GovTicket, int, error) {
	return s.repo.List(status, "", "", assigneeID, offset, limit)
}

// Stats 督办总览（书记/管理端）
func (s *GovTicketService) Stats(ctx context.Context, college string) (map[string]int, error) {
	return s.repo.CountByStatus(college)
}
