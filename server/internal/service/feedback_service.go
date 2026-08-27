package service

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/dll/wxx/server/internal/llm"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/util"
	"github.com/google/uuid"
)

// FeedbackService 用户反馈业务服务
type FeedbackService struct {
	feedbackRepo   *repository.FeedbackRepo
	userRepo       *repository.UserRepo
	screenshotRepo *repository.FeedbackScreenshotRepo
	repairRepo     *repository.FeedbackRepairRepo
	db             *sql.DB
	// 可选：用户反馈"回答有误"时回调（用于失效 FAQ 缓存等）
	// messageID 是用户在前端记录的消息 id，由调用方决定如何用它定位原问题
	onAnswerError func(messageID, content string)
	// 可选：AI 在线修复依赖（视觉模型解析截图 + 文本模型诊断）
	visionClient llm.VisionClient
	llmClient    llm.ChatClient
}

// NewFeedbackService 创建反馈服务
func NewFeedbackService(feedbackRepo *repository.FeedbackRepo, userRepo *repository.UserRepo, screenshotRepo *repository.FeedbackScreenshotRepo) *FeedbackService {
	return &FeedbackService{feedbackRepo: feedbackRepo, userRepo: userRepo, screenshotRepo: screenshotRepo}
}

// SetDB 设置数据库连接（用于发送站内通知）
func (s *FeedbackService) SetDB(db *sql.DB) {
	s.db = db
}

// SetRepairRepo 注入 AI 修复工单持久化仓库（可为 nil，注入后续记录每次修复工单）
func (s *FeedbackService) SetRepairRepo(repo *repository.FeedbackRepairRepo) {
	s.repairRepo = repo
}

// SetAnswerErrorHook 注入"回答有误"反馈钩子（如失效 FAQ 缓存）
// 钩子在反馈成功保存后异步执行，不影响反馈提交结果
func (s *FeedbackService) SetAnswerErrorHook(fn func(messageID, content string)) {
	s.onAnswerError = fn
}

// SetAIRepairClients 注入 AI 在线修复依赖（视觉模型解析截图 + 文本模型诊断）。
// 未注入时 AIRepair 仅返回基于本地模块映射的兜底结果。
func (s *FeedbackService) SetAIRepairClients(vision llm.VisionClient, chat llm.ChatClient) {
	s.visionClient = vision
	s.llmClient = chat
}

// moduleFilesMap 模块 → 项目代码文件映射（与前端 feedback_repair._moduleMap 保持一致，
// 作为 LLM 不可用时的离线兜底 + LLM 判定的对照表）
var moduleFilesMap = map[string][]string{
	"登录 / 认证": {
		"frontend/lib/pages/login/login_page.dart",
		"frontend/lib/providers/auth_provider.dart",
		"server/internal/handler/auth_handler.go",
	},
	"对话 / 问答": {
		"frontend/lib/pages/chat/chat_page.dart",
		"frontend/lib/providers/chat_provider.dart",
		"server/internal/service/chat_service.go",
		"server/internal/context_engine/engine.go",
	},
	"知识库 / 检索": {
		"server/internal/context_engine/",
		"server/internal/repository/kb_repo.go",
		"frontend/lib/pages/knowledge/",
	},
	"办事流程": {
		"frontend/lib/pages/process/",
		"server/internal/handler/process_handler.go",
		"frontend/lib/providers/process_provider.dart",
	},
	"报到 / 校园导航": {
		"frontend/lib/pages/campus/campus_map_page.dart",
		"frontend/lib/widgets/baidu_campus_map_embed_web.dart",
		"server/internal/handler/campus_handler.go",
	},
	"语音": {
		"frontend/lib/services/voice/",
		"server/internal/handler/voice_handler.go",
	},
	"我的 / 个人中心": {
		"frontend/lib/pages/profile/profile_page.dart",
		"frontend/lib/providers/auth_provider.dart",
	},
	"反馈系统": {
		"frontend/lib/pages/admin/feedback_page.dart",
		"frontend/lib/providers/feedback_provider.dart",
		"server/internal/handler/feedback_handler.go",
	},
	"消息 / 通知": {
		"frontend/lib/pages/notifications/",
		"server/internal/handler/notification_handler.go",
	},
	"学生服务": {
		"frontend/lib/pages/student/",
		"server/internal/service/student_service.go",
	},
	"教务 / 课表": {
		"frontend/lib/pages/student/",
		"server/internal/service/study_service.go",
	},
	"心理 / 情感": {
		"frontend/lib/pages/student/mental/",
		"server/internal/service/emotion_service.go",
	},
	"管理端 / 数据": {
		"frontend/lib/pages/admin/",
		"server/internal/handler/admin_handler.go",
	},
}

// moduleKeywords 模块 → 关键词（本地兜底匹配，与前端 _moduleMap.keywords 一致）
var moduleKeywords = map[string][]string{
	"登录 / 认证":   {"登录", "认证", "账号", "密码", "扫码", "token", "验证码", "login", "auth"},
	"对话 / 问答":   {"回答", "问答", "对话", "聊天", "回复", "答案", "chat", "AI", "智能"},
	"知识库 / 检索":  {"知识", "检索", "搜索", "词条", "知识库", "FTS", "搜索结果", "查不到"},
	"办事流程":      {"办事", "流程", "手续", "申请", "审批", "process"},
	"报到 / 校园导航": {"报到", "地图", "导航", "校园", "节点", "校区", "campus", "map"},
	"语音":        {"语音", "说话", "录音", "麦克风", "TTS", "ASR", "voice"},
	"我的 / 个人中心": {"我的", "个人", "资料", "头像", "设置", "profile", "个人信息"},
	"反馈系统":      {"反馈", "意见", "投诉", "feedback"},
	"消息 / 通知":   {"通知", "消息", "提醒", "公告", "notification"},
	"学生服务":      {"学生", "学情", "打卡", "日记", "日报", "晨报", "student"},
	"教务 / 课表":   {"课表", "课程", "成绩", "选课", "考试", "排课", "study", "schedule"},
	"心理 / 情感":   {"心理", "情感", "心情", "咨询", "焦虑", "emotion", "mental"},
	"管理端 / 数据":  {"管理", "统计", "看板", "用户管理", "导入", "admin", "仪表"},
}

// matchModuleByContent 本地关键词匹配模块（兜底，返回命中的模块名列表，按命中数排序）
func matchModuleByContent(content string) []string {
	text := strings.ToLower(content)
	type scored struct {
		module string
		score  int
	}
	var list []scored
	for module, kws := range moduleKeywords {
		score := 0
		for _, kw := range kws {
			if strings.Contains(text, strings.ToLower(kw)) {
				score++
			}
		}
		if score > 0 {
			list = append(list, scored{module: module, score: score})
		}
	}
	// 稳定排序（模块名作次级 key）
	sort.SliceStable(list, func(i, j int) bool {
		if list[i].score != list[j].score {
			return list[i].score > list[j].score
		}
		return list[i].module < list[j].module
	})
	top := make([]string, 0, len(list))
	for i := 0; i < len(list) && i < 4; i++ {
		top = append(top, list[i].module)
	}
	return top
}

// AIRepair AI 在线修复：解析截图 + 智能定位模块与代码文件。
// 返回诊断结果；LLM/视觉不可用时自动降级为本地关键词匹配。
// 若已注入 repairRepo，则把本次诊断持久化为修复工单（供前端轮询与审计）。
func (s *FeedbackService) AIRepair(ctx context.Context, feedbackID, operator string) (*model.AIRepairResponse, error) {
	fb, err := s.feedbackRepo.GetByFeedbackID(feedbackID)
	if err != nil {
		return nil, fmt.Errorf("查询反馈失败: %w", err)
	}
	if fb == nil {
		return nil, fmt.Errorf("反馈不存在: %s", feedbackID)
	}

	// 0. 创建修复工单（审计 + 前端轮询）
	jobID := int64(0)
	runID := ""
	if s.repairRepo != nil {
		runID = "rp-" + strings.ReplaceAll(uuid.New().String()[:13], "-", "")
		jobID, err = s.repairRepo.Create(&model.FeedbackRepairJob{
			RunID:      runID,
			FeedbackID: feedbackID,
			Operator:   operator,
			Status:     model.RepairStatusRunning,
			Stage:      model.RepairStageDiagnose,
			Summary:    fb.Content,
		})
		if err != nil {
			log.Printf("反馈修复工单创建失败 feedback_id=%s err=%v", feedbackID, err)
		} else {
			_ = s.repairRepo.AppendLog(jobID, "初始化修复工单，开始 AI 诊断")
		}
	}
	finish := func(status, stage, detail string, files []string) {
		if s.repairRepo == nil || jobID == 0 {
			return
		}
		if filesJSON, ferr := json.Marshal(files); ferr == nil {
			_ = s.repairRepo.SetEditedFiles(jobID, string(filesJSON))
		}
		_ = s.repairRepo.UpdateStage(jobID, stage)
		_ = s.repairRepo.Finalize(jobID, status, detail)
	}

	resp := &model.AIRepairResponse{Module: fb.Module}

	// 1. 若反馈带截图 → 用视觉模型解析截图文字
	var ocrText string
	if fb.ScreenshotURL != "" && s.visionClient != nil {
		dataB64, mime, gerr := s.screenshotRepo.GetByFilename(filepath.Base(fb.ScreenshotURL))
		if gerr == nil && dataB64 != "" {
			imgBytes, derr := base64.StdEncoding.DecodeString(dataB64)
			if derr == nil {
				if mime == "" {
					mime = "image/png"
				}
				if t, oerr := s.visionClient.OCR(ctx, []llm.OCRImage{{Data: imgBytes, MIME: mime}}); oerr == nil {
					ocrText = strings.TrimSpace(t)
				} else {
					log.Printf("反馈截图 OCR 失败 feedback_id=%s err=%v", feedbackID, oerr)
				}
			}
		}
	}
	resp.OCRText = ocrText

	// 2. 本地兜底：按内容 + 模块 匹配代码文件
	localModules := matchModuleByContent(fb.Content + " " + fb.Module)
	if len(localModules) == 0 {
		localModules = []string{fb.Module}
	}
	for _, m := range localModules {
		if files, ok := moduleFilesMap[m]; ok {
			resp.MatchedFiles = append(resp.MatchedFiles, files...)
		}
	}

	// 3. 文本模型诊断（有对话客户端时，用内容+OCR 智能化生成）
	if s.llmClient != nil {
		diagnosis, derr := s.aiDiagnose(ctx, fb, ocrText)
		if derr == nil && diagnosis != nil {
			if diagnosis.Module != "" {
				resp.Module = diagnosis.Module
			}
			resp.Summary = diagnosis.Summary
			resp.CodeFiles = diagnosis.CodeFiles
			resp.RootCause = diagnosis.RootCause
			resp.RepairHint = diagnosis.RepairHint
		} else if derr != nil {
			log.Printf("AI 诊断失败 feedback_id=%s err=%v", feedbackID, derr)
		}
	}

	// 4. 兜底默认值
	if resp.Summary == "" {
		resp.Summary = fb.Content
	}
	if resp.Module == "" {
		resp.Module = "未识别"
	}
	if len(resp.CodeFiles) == 0 {
		resp.CodeFiles = resp.MatchedFiles
	}
	if resp.RepairHint == "" {
		resp.RepairHint = "请结合上方代码定位，在本机复现并修复；修复完成后回到本页将该反馈标记为已解决。"
	}

	// 5. 持久化工单：诊断成功
	finish(model.RepairStatusSucceeded, model.RepairStageDone,
		resp.Summary+" ｜ 根因："+resp.RootCause, resp.CodeFiles)

	// 记录处理日志
	_ = s.feedbackRepo.AddLog(feedbackID, "ai_repair", operator, "AI 在线修复诊断")

	resp.RunID = runID
	return resp, nil
}

// LatestRepairJob 查询反馈的最新一次修复工单（用于前端轮询/审计）。
// 未注入 repairRepo 时返回 nil。
func (s *FeedbackService) LatestRepairJob(feedbackID string) (*model.FeedbackRepairJob, error) {
	if s.repairRepo == nil {
		return nil, nil
	}
	return s.repairRepo.LatestByFeedback(feedbackID)
}

// aiDiagnoseResult 文本模型诊断输出结构
type aiDiagnoseResult struct {
	Module     string   `json:"module"`
	Summary    string   `json:"summary"`
	CodeFiles  []string `json:"code_files"`
	RootCause  string   `json:"root_cause"`
	RepairHint string   `json:"repair_hint"`
}

// aiDiagnose 调用文本模型，把反馈内容（+截图 OCR）转化为结构化诊断。
func (s *FeedbackService) aiDiagnose(ctx context.Context, fb *model.Feedback, ocrText string) (*aiDiagnoseResult, error) {
	catalog := s.fileCatalog()
	ocrPart := ""
	if strings.TrimSpace(ocrText) != "" {
		ocrPart = "\n截图 OCR 识别文本：\n" + ocrText
	}
	userPrompt := fmt.Sprintf(`反馈内容：%s
反馈分类：%s
用户标注模块：%s%s

请分析问题并按如下 JSON 返回（不要输出 JSON 以外的内容）：
{
  "module": "最可能的模块名（从下方候选模块中选择，不要编造）",
  "summary": "一句话问题摘要（30字内）",
  "code_files": ["相对仓库根的项目文件路径，1-4个，从下方候选文件中挑选"],
  "root_cause": "可能根因分析（100字内）",
  "repair_hint": "具体修复建议（可操作，200字内）"
}

候选模块与对应代码文件：
%s`,
		fb.Content, fb.Category, fb.Module, ocrPart, catalog,
	)

	resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
		Messages: []llm.ChatMessage{
			{Role: "system", Content: "你是校园智能问答助手「蔚小芯」的研发工程师，擅长根据用户反馈快速定位前后端代码问题。严格按照用户要求的 JSON 格式输出。"},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.2,
		MaxTokens:   800,
	})
	if err != nil {
		return nil, err
	}
	return parseAIRepairJSON(resp.Content)
}

// moduleListForPrompt 生成候选模块清单（用于 AI 诊断约束）
func moduleListForPrompt() string {
	keys := make([]string, 0, len(moduleFilesMap))
	for k := range moduleFilesMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return strings.Join(keys, "、")
}

// fileCatalog 拼装模块→代码文件 清单，供 AI 判定最相关文件
func (s *FeedbackService) fileCatalog() string {
	keys := make([]string, 0, len(moduleFilesMap))
	for k := range moduleFilesMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		b.WriteString("- " + k + ": " + strings.Join(moduleFilesMap[k], ", ") + "\n")
	}
	return b.String()
}

// parseAIRepairJSON 解析 AI 返回的 JSON（容忍首尾噪声，从第一个 { 截取到最后一个 }）
func parseAIRepairJSON(raw string) (*aiDiagnoseResult, error) {
	s, e := strings.Index(raw, "{"), strings.LastIndex(raw, "}")
	if s < 0 || e <= s {
		return nil, fmt.Errorf("AI 输出非 JSON: %s", truncateStr(raw, 120))
	}
	var out aiDiagnoseResult
	if err := json.Unmarshal([]byte(raw[s:e+1]), &out); err != nil {
		return nil, fmt.Errorf("解析 AI JSON 失败: %v", err)
	}
	return &out, nil
}

// truncateStr 截断字符串用于日志
func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// Submit 提交反馈（含可选截图）
func (s *FeedbackService) Submit(userID int64, username string, req *model.FeedbackCreateRequest) (*model.Feedback, error) {
	// 校验 token 中的用户是否仍存在于当前数据库（Vercel /tmp 数据库冷启动后旧 token 可能失效）
	if s.userRepo != nil {
		u, err := s.userRepo.GetByID(userID)
		if err != nil {
			return nil, ErrUserNotFound
		}
		if u == nil {
			return nil, ErrUserNotFound
		}
	}

	fb := &model.Feedback{
		FeedbackID:    "fb-" + uuid.New().String()[:8],
		UserID:        userID,
		Username:      username,
		MessageID:     req.MessageID,
		ResourceID:    req.ResourceID,
		Category:      req.Category,
		Module:        req.Module,
		Content:       req.Content,
		ScreenshotURL: req.ScreenshotURL,
		Status:        "pending",
	}

	id, err := s.feedbackRepo.Create(fb)
	if err != nil {
		return nil, fmt.Errorf("保存反馈失败: %w", err)
	}

	fb.ID = id
	log.Printf("用户反馈已提交 feedback_id=%s category=%s has_screenshot=%v by=%s",
		fb.FeedbackID, fb.Category, fb.ScreenshotURL != "", username)

	// 记录处理日志
	_ = s.feedbackRepo.AddLog(fb.FeedbackID, "submit", username, "用户提交反馈")

	// 仅 "回答有误" 类反馈触发钩子（异步，不影响响应）
	if req.Category == "answer_error" && s.onAnswerError != nil {
		go s.onAnswerError(req.MessageID, req.Content)
	}
	return fb, nil
}

// List 分页查询反馈列表
func (s *FeedbackService) List(status string, page, pageSize int) ([]*model.Feedback, int, error) {
	offset, _, _ := util.Paginate(page, pageSize)

	items, err := s.feedbackRepo.List(status, offset, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("查询反馈列表失败: %w", err)
	}

	total, err := s.feedbackRepo.Count(status)
	if err != nil {
		return nil, 0, fmt.Errorf("统计反馈总数失败: %w", err)
	}

	return items, total, nil
}

// ListMine 查询指定用户自己提交的反馈（用于"我的反馈"页面）
func (s *FeedbackService) ListMine(userID int64, status string, page, pageSize int) ([]*model.Feedback, int, error) {
	offset, _, _ := util.Paginate(page, pageSize)

	items, err := s.feedbackRepo.ListByUser(userID, status, offset, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("查询我的反馈失败: %w", err)
	}

	total, err := s.feedbackRepo.CountByUser(userID, status)
	if err != nil {
		return nil, 0, fmt.Errorf("统计我的反馈总数失败: %w", err)
	}

	return items, total, nil
}

// Get 获取单条反馈详情（无归属校验，仅供内部/既有调用方使用；
// 对外 HTTP 入口请使用 GetAuthorized，见下方安全修复 G1）。
func (s *FeedbackService) Get(feedbackID string) (*model.Feedback, error) {
	fb, err := s.feedbackRepo.GetByFeedbackID(feedbackID)
	if err != nil {
		return nil, fmt.Errorf("查询反馈失败: %w", err)
	}
	return fb, nil
}

// GetAuthorized 获取单条反馈详情（带归属/权限校验，修复 G1 水平越权）。
// canManageAll 为 true 时（反馈管理员，持有 union.feedback.list）可查看任意反馈；
// 否则仅当 userID 等于反馈提交者本人时返回，其余返回 (nil, nil)。
func (s *FeedbackService) GetAuthorized(feedbackID string, userID int64, canManageAll bool) (*model.Feedback, error) {
	fb, err := s.feedbackRepo.GetByFeedbackID(feedbackID)
	if err != nil {
		return nil, fmt.Errorf("查询反馈失败: %w", err)
	}
	if fb == nil {
		return nil, nil
	}
	if canManageAll || fb.UserID == userID {
		return fb, nil
	}
	return nil, nil
}

// CanAccessScreenshot 判断用户是否有权访问某张反馈截图（修复 G1 截图越权）。
// 允许：截图上传者（feedback_screenshots.uploaded_by == username）或截图被当前用户
// 提交的反馈引用（feedback.screenshot_url 含该文件名且 feedback.user_id == userID）。
func (s *FeedbackService) CanAccessScreenshot(filename string, userID int64) (bool, error) {
	// 路径 1：该用户是否上传过此截图
	uploader, err := s.screenshotRepo.OwnerByFilename(filename)
	if err != nil {
		return false, err
	}
	if uploader != "" {
		// 用 userRepo 反查 username 是否对应当前 userID
		if s.userRepo != nil {
			u, uerr := s.userRepo.GetByID(userID)
			if uerr == nil && u != nil && u.Username == uploader {
				return true, nil
			}
		}
	}
	// 路径 2：该用户提交的反馈是否引用了此截图
	refCount, rerr := s.feedbackRepo.CountScreenshotRefsByUser(filename, userID)
	if rerr != nil {
		return false, rerr
	}
	return refCount > 0, nil
}

// Resolve 处理反馈（标记为处理中/已解决/驳回，含可选回复）
func (s *FeedbackService) Resolve(feedbackID, resolvedBy, status, reply string) (*model.Feedback, error) {
	fb, err := s.feedbackRepo.GetByFeedbackID(feedbackID)
	if err != nil {
		return nil, fmt.Errorf("查询反馈失败: %w", err)
	}
	if fb == nil {
		return nil, fmt.Errorf("反馈不存在: %s", feedbackID)
	}
	// 允许的状态流转：pending→processing, pending→resolved, pending→dismissed, processing→resolved, processing→dismissed
	validTransitions := map[string][]string{
		"pending":    {"processing", "resolved", "dismissed"},
		"processing": {"resolved", "dismissed"},
		"resolved":   {},
		"dismissed":  {},
	}
	allowed, ok := validTransitions[fb.Status]
	if !ok {
		return nil, fmt.Errorf("未知的反馈状态: %s", fb.Status)
	}
	valid := false
	for _, s := range allowed {
		if s == status {
			valid = true
			break
		}
	}
	if !valid {
		return nil, fmt.Errorf("反馈状态为 %s，不可变更为 %s", fb.Status, status)
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	oldStatus := fb.Status
	fb.Status = status
	fb.ResolvedBy = resolvedBy
	fb.ResolvedAt = &now
	if reply != "" {
		fb.Reply = reply
	}

	if err := s.feedbackRepo.Update(fb); err != nil {
		return nil, fmt.Errorf("更新反馈状态失败: %w", err)
	}

	// 记录处理日志
	action := "status_change"
	detail := fmt.Sprintf("状态从 %s 变更为 %s", oldStatus, status)
	if reply != "" {
		detail += fmt.Sprintf("，回复：%s", reply)
	}
	_ = s.feedbackRepo.AddLog(feedbackID, action, resolvedBy, detail)

	log.Printf("反馈已处理 feedback_id=%s status=%s reply=%s by=%s",
		feedbackID, status, reply, resolvedBy)

	// 如果状态变为 resolved，异步发送站内通知
	if status == "resolved" {
		go s.sendResolveNotification(fb)
	}

	return fb, nil
}

// sendResolveNotification 发送反馈解决通知
func (s *FeedbackService) sendResolveNotification(fb *model.Feedback) {
	if s.db == nil {
		return
	}
	title := "您的反馈已解决"
	content := fmt.Sprintf("您提交的反馈（%s）已处理完成，回复：%s", fb.FeedbackID, fb.Reply)
	if fb.Reply == "" {
		content = fmt.Sprintf("您提交的反馈（%s）已处理完成。", fb.FeedbackID)
	}

	_, err := s.db.Exec(
		`INSERT INTO user_notifications (user_id, title, content, type, related_type, related_id, is_read)
		 VALUES (?, ?, ?, 'feedback', 'feedback', ?, 0)`,
		fb.UserID, title, content, fb.ID,
	)
	if err != nil {
		log.Printf("发送反馈解决通知失败: feedback_id=%s user_id=%d err=%v", fb.FeedbackID, fb.UserID, err)
	}
}

// GetStats 获取反馈统计数据
func (s *FeedbackService) GetStats() (*model.FeedbackStats, error) {
	total, err := s.feedbackRepo.Count("")
	if err != nil {
		return nil, fmt.Errorf("统计总数失败: %w", err)
	}

	byStatus, err := s.feedbackRepo.CountByStatus()
	if err != nil {
		return nil, fmt.Errorf("按状态统计失败: %w", err)
	}

	byCategory, err := s.feedbackRepo.CountByCategory()
	if err != nil {
		return nil, fmt.Errorf("按分类统计失败: %w", err)
	}

	weekTrend, err := s.feedbackRepo.WeekTrend()
	if err != nil {
		return nil, fmt.Errorf("获取周趋势失败: %w", err)
	}

	topIssues, err := s.feedbackRepo.TopIssues(10)
	if err != nil {
		return nil, fmt.Errorf("获取热门问题失败: %w", err)
	}

	avgHours, err := s.feedbackRepo.AvgResolveHours()
	if err != nil {
		return nil, fmt.Errorf("获取平均解决时长失败: %w", err)
	}

	return &model.FeedbackStats{
		Total:           total,
		ByStatus:        byStatus,
		ByCategory:      byCategory,
		WeekTrend:       weekTrend,
		TopIssues:       topIssues,
		AvgResolveHours: avgHours,
	}, nil
}

// LinkResource 关联知识库资源
func (s *FeedbackService) LinkResource(feedbackID, resourceID, note, operator string) error {
	fb, err := s.feedbackRepo.GetByFeedbackID(feedbackID)
	if err != nil {
		return fmt.Errorf("查询反馈失败: %w", err)
	}
	if fb == nil {
		return fmt.Errorf("反馈不存在: %s", feedbackID)
	}

	if err := s.feedbackRepo.LinkResource(feedbackID, resourceID, note, operator); err != nil {
		return fmt.Errorf("关联知识资源失败: %w", err)
	}

	// 记录处理日志
	detail := fmt.Sprintf("关联资源 %s", resourceID)
	if note != "" {
		detail += fmt.Sprintf("，备注：%s", note)
	}
	_ = s.feedbackRepo.AddLog(feedbackID, "link_resource", operator, detail)

	log.Printf("反馈已关联知识资源 feedback_id=%s resource_id=%s by=%s",
		feedbackID, resourceID, operator)
	return nil
}

// Rate 满意度评价
func (s *FeedbackService) Rate(feedbackID string, userID int64, rating int, comment string) error {
	fb, err := s.feedbackRepo.GetByFeedbackID(feedbackID)
	if err != nil {
		return fmt.Errorf("查询反馈失败: %w", err)
	}
	if fb == nil {
		return fmt.Errorf("反馈不存在: %s", feedbackID)
	}
	if fb.UserID != userID {
		return fmt.Errorf("无权评价此反馈")
	}
	if fb.Status != "resolved" {
		return fmt.Errorf("仅已解决的反馈可评价")
	}
	if fb.Rating > 0 {
		return fmt.Errorf("该反馈已评价过")
	}

	if rating < 1 || rating > 5 {
		return fmt.Errorf("评分必须在 1-5 之间")
	}

	if err := s.feedbackRepo.UpdateRating(feedbackID, rating, comment); err != nil {
		return fmt.Errorf("保存评分失败: %w", err)
	}

	// 记录处理日志
	detail := fmt.Sprintf("用户评分 %d 星", rating)
	if comment != "" {
		detail += fmt.Sprintf("，评价：%s", comment)
	}
	_ = s.feedbackRepo.AddLog(feedbackID, "rate", fb.Username, detail)

	log.Printf("反馈已评价 feedback_id=%s rating=%d by=user_%d",
		feedbackID, rating, userID)
	return nil
}

// ListLogs 获取反馈处理记录
func (s *FeedbackService) ListLogs(feedbackID string) ([]*model.FeedbackLog, error) {
	logs, err := s.feedbackRepo.ListLogs(feedbackID)
	if err != nil {
		return nil, fmt.Errorf("获取处理记录失败: %w", err)
	}
	return logs, nil
}

// SaveScreenshot 保存截图数据
func (s *FeedbackService) SaveScreenshot(filename, mime, encoded, uploader string, size int64) error {
	return s.screenshotRepo.Save(filename, mime, encoded, uploader, size)
}

// GetScreenshot 按文件名获取截图
func (s *FeedbackService) GetScreenshot(filename string) (dataB64, mime string, err error) {
	return s.screenshotRepo.GetByFilename(filename)
}
