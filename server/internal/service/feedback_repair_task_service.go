// Package service — 反馈修复任务（闭环 MVP）业务逻辑。
//
// 安全边界：服务器绝不执行源码修改、构建或部署；本服务仅做状态机流转与审计。
// 执行端认证：使用独立环境变量 WXX_REPAIR_AGENT_TOKEN 的专用 token 中间件（见
// middleware.RepairAgentTokenAuth），不授予任何业务角色（含 sys_admin）执行能力，
// 与交互式前台用户 JWT 完全隔离；token 不硬编码、不写库、不入日志。
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/google/uuid"
)

// ErrRepairTaskNotFound 任务不存在
var ErrRepairTaskNotFound = errors.New("修复任务不存在")

// ErrRepairTaskBadState 当前状态不允许该操作
var ErrRepairTaskBadState = errors.New("修复任务当前状态不允许此操作")

// ErrRepairTaskConcurrency 全局仅允许一个 running 任务（避免并发改码冲突）
var ErrRepairTaskConcurrency = errors.New("已有修复任务在执行中，请先完成或取消后再领取")

// FeedbackRepairTaskService 反馈修复任务服务
type FeedbackRepairTaskService struct {
	repo        *repository.FeedbackRepairTaskRepo
	feedbackSvc *FeedbackService
}

// NewFeedbackRepairTaskService 创建修复任务服务
func NewFeedbackRepairTaskService(repo *repository.FeedbackRepairTaskRepo, feedbackSvc *FeedbackService) *FeedbackRepairTaskService {
	return &FeedbackRepairTaskService{repo: repo, feedbackSvc: feedbackSvc}
}

// validTransitions 状态机合法流转表
var validTransitions = map[string][]string{
	model.RepairTaskApproved:           {model.RepairTaskRunning, model.RepairTaskCancelled},
	model.RepairTaskRunning:            {model.RepairTaskAwaitingAcceptance, model.RepairTaskVerifyFailed},
	model.RepairTaskVerifyFailed:       {model.RepairTaskRunning, model.RepairTaskCancelled},
	model.RepairTaskAwaitingAcceptance: {model.RepairTaskDeployPending, model.RepairTaskVerifyFailed},
	model.RepairTaskDeployPending:      {model.RepairTaskDeploying, model.RepairTaskVerifyFailed},
	model.RepairTaskDeploying:          {model.RepairTaskDeployed},
	model.RepairTaskDeployed:           {model.RepairTaskClosed},
	model.RepairTaskClosed:             {},
	model.RepairTaskCancelled:          {},
}

// canTransition 判断 from -> to 是否合法
func canTransition(from, to string, task *model.FeedbackRepairTask, report *model.RepairTaskVerifyReport) bool {
	allowed, ok := validTransitions[from]
	if !ok {
		return false
	}
	for _, a := range allowed {
		if a == to {
			return true
		}
	}
	// 特例：allStatus 校验，仅当任一非法目标被特判时才走这里
	_ = task
	_ = report
	return false
}

// Create 管理员审核后创建修复任务（可单条/批量）。
// 创建即意味着"已审核"（creator 记录审核人）；逐条跑 AIRepair 诊断合并 code_files。
func (s *FeedbackRepairTaskService) Create(ctx context.Context, creator string, req *model.RepairTaskCreateRequest) (*model.FeedbackRepairTask, error) {
	if len(req.FeedbackIDs) == 0 {
		return nil, errors.New("请至少选择一条反馈")
	}
	// 去重
	seen := map[string]bool{}
	var ids []string
	for _, id := range req.FeedbackIDs {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, errors.New("无效的反馈列表")
	}

	// 合并诊断：逐条跑 AIRepair（复用现有诊断），收集 code_files 与摘要
	var diagnosis model.AIRepairResponse
	mergedFiles := []string{}
	mergedSummary := []string{}
	for _, fid := range ids {
		resp, err := s.feedbackSvc.AIRepair(ctx, fid, creator)
		if err != nil {
			log.Printf("[repair-task] 诊断失败 feedback_id=%s err=%v", fid, err)
			continue
		}
		if resp == nil {
			continue
		}
		if resp.Summary != "" {
			mergedSummary = append(mergedSummary, resp.Summary)
		}
		if diagnosis.Module == "" {
			diagnosis.Module = resp.Module
		}
		for _, f := range resp.CodeFiles {
			if !containsStr(mergedFiles, f) {
				mergedFiles = append(mergedFiles, f)
			}
		}
	}
	if len(mergedSummary) > 0 {
		diagnosis.Summary = strings.Join(mergedSummary, "；")
	}
	diagnosis.CodeFiles = mergedFiles

	diagJSON, _ := json.Marshal(diagnosis)

	task := &model.FeedbackRepairTask{
		TaskNo:          "rt-" + strings.ReplaceAll(uuid.New().String()[:13], "-", ""),
		Creator:         creator,
		FeedbackIDs:     repository.FeedbackIDsToJSON(ids),
		Title:           strings.TrimSpace(req.Title),
		Diagnosis:       string(diagJSON),
		Status:          model.RepairTaskApproved,
		WorkerTokenNote: "执行端通过专用 token 认领，token 本身不在此记录，请在配置中管理。",
	}
	if task.Title == "" {
		task.Title = fmt.Sprintf("修复 %d 条反馈", len(ids))
	}

	id, err := s.repo.Create(task)
	if err != nil {
		return nil, fmt.Errorf("创建修复任务失败: %w", err)
	}
	task.ID = id

	// 审计：为每条反馈追加 repair_task_created 日志
	for _, fid := range ids {
		_ = s.feedbackSvc.feedbackRepo.AddLog(fid, "repair_task_created", creator,
			fmt.Sprintf("已创建修复任务 %s", task.TaskNo))
	}

	return task, nil
}

// List 管理端任务分页列表
func (s *FeedbackRepairTaskService) List(status string, page, pageSize int) ([]*model.RepairTaskDTO, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	items, total, err := s.repo.List(status, offset, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("查询修复任务失败: %w", err)
	}

	dtos := make([]*model.RepairTaskDTO, 0, len(items))
	for _, it := range items {
		dtos = append(dtos, taskToDTO(it))
	}
	return dtos, total, nil
}

// Get 任务详情（管理端）
func (s *FeedbackRepairTaskService) Get(taskNo string) (*model.RepairTaskDTO, error) {
	t, err := s.repo.GetByTaskNo(taskNo)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrRepairTaskNotFound
	}
	return taskToDTO(t), nil
}

// Cancel 取消任务（仅 approved / verify_failed）
func (s *FeedbackRepairTaskService) Cancel(taskNo, operator string) (*model.RepairTaskDTO, error) {
	t, err := s.repo.GetByTaskNo(taskNo)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrRepairTaskNotFound
	}
	if !canTransition(t.Status, model.RepairTaskCancelled, t, nil) {
		return nil, fmt.Errorf("%w: 当前状态 %s 不可取消", ErrRepairTaskBadState, t.Status)
	}
	if err := s.repo.UpdateStatus(t.ID, model.RepairTaskCancelled); err != nil {
		return nil, fmt.Errorf("取消任务失败: %w", err)
	}
	_ = s.repo.AppendLog(t.ID, fmt.Sprintf("[系统] %s 取消了任务", operator))
	return s.Get(taskNo)
}

// Claim 执行端认领：原子地取最老 approved/verify_failed 任务并置 running。
// MVP 全局仅允许 1 个 running，防并发改码冲突。
func (s *FeedbackRepairTaskService) Claim(ctx context.Context, workerHost, baseCommit, branch string) (*model.RepairTaskPayload, error) {
	// 并发闸门：全局 running + awaiting_acceptance 计数
	n, err := s.repo.CountActiveRunning()
	if err != nil {
		return nil, fmt.Errorf("统计执行中任务失败: %w", err)
	}
	if n > 0 {
		return nil, ErrRepairTaskConcurrency
	}

	t, err := s.repo.NextClaimable()
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrRepairTaskNotFound
	}

	if err := s.repo.UpdateClaim(t.ID, model.RepairTaskRunning, workerHost, baseCommit, branch); err != nil {
		return nil, fmt.Errorf("认领任务失败: %w", err)
	}
	_ = s.repo.AppendLog(t.ID, fmt.Sprintf("[执行端] %s 领取任务 (base=%s branch=%s)", workerHost, baseCommit, branch))

	payload := taskToPayload(t)
	return payload, nil
}

// SubmitVerify 执行端验证结果上报。
// passed=true -> awaiting_acceptance；passed=false -> verify_failed。
func (s *FeedbackRepairTaskService) SubmitVerify(taskNo string, req *model.RepairTaskVerifyRequest, operator string) (*model.RepairTaskDTO, error) {
	t, err := s.repo.GetByTaskNo(taskNo)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrRepairTaskNotFound
	}
	// 仅 running 状态可上报验证
	if t.Status != model.RepairTaskRunning {
		return nil, fmt.Errorf("%w: 仅运行中的任务可上报验证，当前 %s", ErrRepairTaskBadState, t.Status)
	}

	bundle := model.RepairTaskVerifyResultBundle{
		Passed:         req.Passed,
		GoVet:          req.GoVet,
		GoTest:         req.GoTest,
		FlutterAnalyze: req.FlutterAnalyze,
		FlutterTest:    req.FlutterTest,
	}
	bJSON, _ := json.Marshal(bundle)

	next := model.RepairTaskAwaitingAcceptance
	if !req.Passed {
		next = model.RepairTaskVerifyFailed
	}
	if err := s.repo.UpdateVerifyReport(t.ID, next, string(bJSON), req.DiffStat, req.Log); err != nil {
		return nil, fmt.Errorf("上报验证结果失败: %w", err)
	}
	log.Printf("[repair-task] %s 验证上报 passed=%v -> %s by=%s", taskNo, req.Passed, next, operator)
	return s.Get(taskNo)
}

// Accept 管理员验收：awaiting_acceptance -> deploy_pending。
func (s *FeedbackRepairTaskService) Accept(taskNo, acceptedBy, note string) (*model.RepairTaskDTO, error) {
	t, err := s.repo.GetByTaskNo(taskNo)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrRepairTaskNotFound
	}
	if !canTransition(t.Status, model.RepairTaskDeployPending, t, nil) {
		return nil, fmt.Errorf("%w: 仅待验收任务可验收，当前 %s", ErrRepairTaskBadState, t.Status)
	}
	if err := s.repo.UpdateAccept(t.ID, model.RepairTaskDeployPending, acceptedBy, note); err != nil {
		return nil, fmt.Errorf("验收任务失败: %w", err)
	}
	_ = s.repo.AppendLog(t.ID, fmt.Sprintf("[验收] %s 验收任务，备注：%s", acceptedBy, note))
	return s.Get(taskNo)
}

// Reject 管理员驳回/要求整改：awaiting_acceptance|deploy_pending -> verify_failed。
func (s *FeedbackRepairTaskService) Reject(taskNo, rejectedBy, reason string) (*model.RepairTaskDTO, error) {
	t, err := s.repo.GetByTaskNo(taskNo)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrRepairTaskNotFound
	}
	if !canTransition(t.Status, model.RepairTaskVerifyFailed, t, nil) {
		return nil, fmt.Errorf("%w: 当前状态 %s 不可驳回", ErrRepairTaskBadState, t.Status)
	}
	if err := s.repo.UpdateReject(t.ID, model.RepairTaskVerifyFailed, rejectedBy, reason); err != nil {
		return nil, fmt.Errorf("驳回任务失败: %w", err)
	}
	_ = s.repo.AppendLog(t.ID, fmt.Sprintf("[驳回] %s 驳回任务，原因：%s", rejectedBy, reason))
	return s.Get(taskNo)
}

// DeployConfirm 管理员确认开始部署（仅标记，不触发服务器动作）。
func (s *FeedbackRepairTaskService) DeployConfirm(taskNo, confirmedBy, deployRef string) (*model.RepairTaskDTO, error) {
	t, err := s.repo.GetByTaskNo(taskNo)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrRepairTaskNotFound
	}
	if !canTransition(t.Status, model.RepairTaskDeploying, t, nil) {
		return nil, fmt.Errorf("%w: 仅已验收任务可确认部署，当前 %s", ErrRepairTaskBadState, t.Status)
	}
	if err := s.repo.UpdateDeployConfirm(t.ID, model.RepairTaskDeploying, confirmedBy, deployRef); err != nil {
		return nil, fmt.Errorf("确认部署失败: %w", err)
	}
	_ = s.repo.AppendLog(t.ID, fmt.Sprintf("[部署确认] %s 开始部署：%s", confirmedBy, deployRef))
	return s.Get(taskNo)
}

// DeployDone 部署完成并记录；可选联动：把关联反馈批量置 resolved。
func (s *FeedbackRepairTaskService) DeployDone(taskNo, doneBy, reply string, resolveFeedback bool) (*model.RepairTaskDTO, error) {
	t, err := s.repo.GetByTaskNo(taskNo)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrRepairTaskNotFound
	}
	if !canTransition(t.Status, model.RepairTaskDeployed, t, nil) {
		return nil, fmt.Errorf("%w: 仅部署中状态可完成部署，当前 %s", ErrRepairTaskBadState, t.Status)
	}
	if err := s.repo.UpdateDeployDone(t.ID, model.RepairTaskDeployed); err != nil {
		return nil, fmt.Errorf("登记部署完成失败: %w", err)
	}
	_ = s.repo.AppendLog(t.ID, fmt.Sprintf("[部署完成] %s 确认已上线", doneBy))

	if resolveFeedback {
		// 复用现有 Resolve 自带状态机 + 站内通知
		for _, fid := range s.parseFeedbackIDs(t.FeedbackIDs) {
			if _, rerr := s.feedbackSvc.Resolve(fid, doneBy, "resolved", reply); rerr != nil {
				log.Printf("[repair-task] 联动解决反馈失败 fid=%s err=%v", fid, rerr)
			}
		}
	}
	// 关闭任务
	if err := s.repo.UpdateDeployDone(t.ID, model.RepairTaskClosed); err != nil {
		log.Printf("[repair-task] 标记关闭失败（不影响部署记录）task=%s err=%v", taskNo, err)
	}
	_ = s.repo.AppendLog(t.ID, "[系统] 任务已关闭")
	return s.Get(taskNo)
}

func (s *FeedbackRepairTaskService) parseFeedbackIDs(raw string) []string {
	var ids []string
	if raw == "" {
		return ids
	}
	if err := json.Unmarshal([]byte(raw), &ids); err != nil {
		// 兜底：逗号/空白分隔
		for _, p := range strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == ' ' }) {
			ids = append(ids, p)
		}
	}
	return ids
}

func taskToDTO(t *model.FeedbackRepairTask) *model.RepairTaskDTO {
	dto := &model.RepairTaskDTO{
		ID:                t.ID,
		TaskNo:            t.TaskNo,
		Creator:           t.Creator,
		FeedbackIDs:       []string{},
		Title:             t.Title,
		Diagnosis:         t.Diagnosis,
		Status:            t.Status,
		WorkerHost:        t.WorkerHost,
		BaseCommit:        t.BaseCommit,
		Branch:            t.Branch,
		VerifyResult:      t.VerifyResult,
		DiffStat:          t.DiffStat,
		LogText:           t.LogText,
		AcceptNote:        t.AcceptNote,
		AcceptedBy:        t.AcceptedBy,
		RejectReason:      t.RejectReason,
		DeployConfirmedBy: t.DeployConfirmedBy,
		DeployRef:         t.DeployRef,
		CreatedAt:         t.CreatedAt,
		UpdatedAt:         t.UpdatedAt,
	}
	if t.FeedbackIDs != "" {
		var ids []string
		if err := json.Unmarshal([]byte(t.FeedbackIDs), &ids); err == nil {
			dto.FeedbackIDs = ids
		}
	}
	return dto
}

func taskToPayload(t *model.FeedbackRepairTask) *model.RepairTaskPayload {
	p := &model.RepairTaskPayload{
		TaskNo:      t.TaskNo,
		Title:       t.Title,
		Status:      t.Status,
		FeedbackIDs: []string{},
		BaseCommit:  t.BaseCommit,
		Branch:      t.Branch,
		LogText:     t.LogText,
		CreatedAt:   t.CreatedAt,
	}
	if t.FeedbackIDs != "" {
		var ids []string
		if err := json.Unmarshal([]byte(t.FeedbackIDs), &ids); err == nil {
			p.FeedbackIDs = ids
		}
	}
	if t.Diagnosis != "" {
		var diag model.AIRepairResponse
		if err := json.Unmarshal([]byte(t.Diagnosis), &diag); err == nil {
			p.Diagnosis = &diag
		}
	}
	return p
}

func containsStr(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
