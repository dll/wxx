package service

import (
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/util"
	"golang.org/x/crypto/bcrypt"
)

// AdminService 管理端业务服务
type AdminService struct {
	userRepo        *repository.UserRepo
	auditRepo       *repository.AuditRepo
	settingsRepo    *repository.SettingsRepo
	chatMetricsRepo *repository.ChatMetricsRepo // 可选：质量指标聚合
}

// NewAdminService 创建管理端服务
func NewAdminService(userRepo *repository.UserRepo, auditRepo *repository.AuditRepo, settingsRepo *repository.SettingsRepo) *AdminService {
	return &AdminService{
		userRepo:     userRepo,
		auditRepo:    auditRepo,
		settingsRepo: settingsRepo,
	}
}

// SetChatMetricsRepo 注入质量指标 repo（可选）
func (s *AdminService) SetChatMetricsRepo(repo *repository.ChatMetricsRepo) {
	s.chatMetricsRepo = repo
}

// GetMetrics 获取质量看板数据（优先从 chat_metrics 聚合真实数据，回退默认值）
func (s *AdminService) GetMetrics() (*model.AdminMetrics, error) {
	m := &model.AdminMetrics{}

	// 统计提问总数（action=chat）
	auditTotal, err := s.auditRepo.Count("", "chat", "", "", "")
	if err != nil {
		log.Printf("[AdminService] 统计提问总数失败: %v", err)
		auditTotal = 0
	}
	m.TotalQuestions = int64(auditTotal)

	// 统计总会话数
	sessionTotal, err := s.auditRepo.Count("", "session", "", "", "")
	if err != nil {
		log.Printf("[AdminService] 统计会话数失败: %v", err)
		sessionTotal = 0
	}
	m.TotalSessions = int64(sessionTotal)

	// 统计今日活跃用户（计算当天日期范围传入）
	today := time.Now().Format("2006-01-02")
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	auditToday, err := s.auditRepo.Count("", "", "", today, tomorrow)
	if err != nil {
		log.Printf("[AdminService] 统计今日活跃用户失败: %v", err)
		auditToday = 0
	}
	m.ActiveUsersNow = int64(auditToday)

	// 这些指标优先从 chat_metrics 表聚合；无真实数据时置 -1，前端显示"暂无数据"而非虚假默认值
	hasMetrics := false
	if s.chatMetricsRepo != nil {
		if agg, err := s.chatMetricsRepo.Aggregate(7); err == nil && agg.TotalQuestions > 0 {
			hasMetrics = true
			m.HitRate = agg.SourceHitRate
			m.FallbackRate = agg.FallbackRate
			m.P95Latency = agg.P95DurationMs
		}
	}
	if !hasMetrics {
		// 无真实数据：用 -1 表示缺失，前端应显示"暂未采集到问答质量数据"，而非编造数字
		m.HitRate = -1
		m.FallbackRate = -1
		m.P95Latency = -1
	}
	// 引用覆盖率暂无独立的真实统计来源，统一置 -1，避免显示未经验证的数值
	m.SourceCoverage = -1

	return m, nil
}

// TopFallbackQuestions 高频兜底问题（知识治理：命中失败高的问题应补录知识库）
func (s *AdminService) TopFallbackQuestions(sinceDays, topN int) ([]repository.TopFallbackQuestion, error) {
	if s.chatMetricsRepo == nil {
		return nil, nil
	}
	return s.chatMetricsRepo.TopFallbackQuestions(sinceDays, topN)
}

// ListUsers 分页查询用户列表，按调用者 scope 过滤
func (s *AdminService) ListUsers(role, ownerScope, ownerID string, page, pageSize int) ([]*model.User, int, error) {
	offset, page, pageSize := util.Paginate(page, pageSize)

	users, err := s.userRepo.List(role, ownerScope, ownerID, offset, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("查询用户列表失败: %w", err)
	}

	total, err := s.userRepo.Count(role, ownerScope, ownerID)
	if err != nil {
		return nil, 0, fmt.Errorf("统计用户总数失败: %w", err)
	}

	return users, total, nil
}

// UpdateUser 修改用户角色/scope
func (s *AdminService) UpdateUser(userID int64, req *model.UserUpdateRequest, operator *model.UserContext) (*model.User, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("用户不存在: id=%d", userID)
	}

	if err := s.checkRoleChangeAuth(operator, user, req); err != nil {
		return nil, err
	}

	if req.DisplayName != nil {
		name := strings.TrimSpace(*req.DisplayName)
		if name == "" {
			return nil, fmt.Errorf("显示名称不能为空")
		}
		user.DisplayName = name
	}
	if req.Role != nil {
		user.Role = *req.Role
	}
	if req.Position != nil {
		user.Position = strings.TrimSpace(*req.Position)
	}
	if req.OwnerScope != nil {
		user.OwnerScope = *req.OwnerScope
	}
	if req.OwnerID != nil {
		user.OwnerID = strings.TrimSpace(*req.OwnerID)
	}
	if req.Status != nil {
		user.Status = *req.Status
	}

	if err := s.userRepo.Update(user); err != nil {
		return nil, fmt.Errorf("更新用户失败: %w", err)
	}

	log.Printf("用户信息已更新 user_id=%d role=%s position=%s scope=%s by=%s", userID, user.Role, user.Position, user.OwnerScope, operator.Username)
	return s.userRepo.GetByID(userID)
}

// checkRoleChangeAuth 角色/职务/归属变更的权限校验（防止越权提权）。
// 规则：
//   - 任何人都不能修改 sys_admin 账号的角色/状态（防止锁死或越权）。
//   - college_admin 只能管理本院（owner_id/college 匹配）用户，且不能授予/转为
//     sys_admin / school_admin 等校级管理员角色，也不能修改其他管理员。
//   - sys_admin / school_admin 可管理任意角色。
func (s *AdminService) checkRoleChangeAuth(operator *model.UserContext, target *model.User, req *model.UserUpdateRequest) error {
	// 操作者必须有效
	if operator == nil {
		return fmt.Errorf("缺少操作者信息")
	}

	// 任何人不能改系统管理员
	if target.Role == "sys_admin" {
		if req.Role != nil && *req.Role != "sys_admin" {
			return fmt.Errorf("不可修改系统管理员角色")
		}
		if req.Status != nil && *req.Status != target.Status {
			return fmt.Errorf("不可修改系统管理员状态")
		}
	}

	// 系统/学校管理员拥有最高权限
	if operator.Role == "sys_admin" || operator.Role == "school_admin" {
		return nil
	}

	// 学院管理员：仅本院 + 不能授予校级管理员 + 不能改管理员
	if operator.Role == "college_admin" {
		if req.Role != nil {
			switch *req.Role {
			case "sys_admin", "school_admin":
				return fmt.Errorf("学院管理员不可授予校级管理员角色")
			case "college_admin":
				if target.ID != operator.UserID {
					return fmt.Errorf("学院管理员不可指派其他学院管理员")
				}
			}
		}
		if target.Role == "college_admin" || target.Role == "school_admin" || target.Role == "sys_admin" {
			if target.ID != operator.UserID {
				return fmt.Errorf("学院管理员不可修改其他管理员账户")
			}
		}
		// 归属范围校验：仅能操作本院用户；无法判定时不误拦（前端已按角色隐藏高阶选项）
		targetCollege := strings.TrimSpace(target.College)
		if targetCollege == "" {
			targetCollege = strings.TrimSpace(target.OwnerID)
		}
		opCollege := strings.TrimSpace(operator.OwnerID)
		if target.ID != operator.UserID && opCollege != "" && targetCollege != "" && targetCollege != opCollege {
			return fmt.Errorf("学院管理员仅可管理本院用户")
		}
		return nil
	}

	return fmt.Errorf("当前角色(%s)无权修改用户角色/职务", operator.Role)
}

// DeleteUser 删除用户及其关联数据
func (s *AdminService) DeleteUser(userID int64, deletedBy string) error {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return fmt.Errorf("查询用户失败: %w", err)
	}
	if user == nil {
		return fmt.Errorf("用户不存在: id=%d", userID)
	}

	// 不允许删除系统管理员
	if user.Role == "sys_admin" {
		return fmt.Errorf("不允许删除系统管理员账户")
	}

	if err := s.userRepo.Delete(userID); err != nil {
		return fmt.Errorf("删除用户失败: %w", err)
	}

	log.Printf("用户已删除 user_id=%d username=%s role=%s by=%s", userID, user.Username, user.Role, deletedBy)
	return nil
}

// ListUsersAdvanced 高级用户查询（搜索+多条件筛选+排序）
func (s *AdminService) ListUsersAdvanced(q *model.UserQuery) ([]*model.User, int, error) {
	users, total, err := s.userRepo.ListAdvanced(q)
	if err != nil {
		return nil, 0, fmt.Errorf("查询用户列表失败: %w", err)
	}
	return users, total, nil
}

// GetUserDictValues 获取用户字典值（学院/专业/班级/入学年份）
func (s *AdminService) GetUserDictValues(column, role, ownerScope, ownerID string) ([]string, error) {
	values, err := s.userRepo.GetDistinctValues(column, role, ownerScope, ownerID)
	if err != nil {
		return nil, fmt.Errorf("获取字典值失败: %w", err)
	}
	return values, nil
}

// BatchUpdateStatus 批量更新用户状态
func (s *AdminService) BatchUpdateStatus(ids []int64, status string, operator string) (int64, error) {
	if len(ids) == 0 {
		return 0, fmt.Errorf("用户ID列表不能为空")
	}
	if status != "active" && status != "disabled" {
		return 0, fmt.Errorf("无效的状态值: %s", status)
	}
	// 操作快照（供管理端恢复）：记录受影响用户操作前后的状态
	// 注意：须在更新前读取 before，更新后再读会得到新状态
	beforeMap := map[int64]string{}
	if s.auditRepo != nil {
		beforeMap, _ = s.userRepo.GetStatusesByIDs(ids)
	}
	count, err := s.userRepo.BatchUpdateStatus(ids, status)
	if err != nil {
		return 0, err
	}
	if s.auditRepo != nil {
		for _, id := range ids {
			before := beforeMap[id]
			if before == "" || before == status {
				continue
			}
			_ = s.auditRepo.CreateSnapshot(&model.AuditSnapshot{
				OpTable:    "users",
				RecordID:   fmt.Sprintf("%d", id),
				Operation:  "user.status",
				BeforeJSON: before,
				AfterJSON:  status,
			})
		}
	}
	log.Printf("批量更新用户状态 count=%d status=%s by=%s", count, status, operator)
	return count, nil
}

// BatchResetPassword 批量重置用户密码
func (s *AdminService) BatchResetPassword(ids []int64, newPassword, operator string) (int64, error) {
	if len(ids) == 0 {
		return 0, fmt.Errorf("用户ID列表不能为空")
	}
	if len(newPassword) < 6 {
		return 0, fmt.Errorf("密码长度至少6位")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return 0, fmt.Errorf("密码加密失败: %w", err)
	}
	count, err := s.userRepo.BatchResetPassword(ids, string(hash))
	if err != nil {
		return 0, err
	}
	log.Printf("批量重置密码 count=%d by=%s", count, operator)
	return count, nil
}

// BatchDelete 批量删除用户
func (s *AdminService) BatchDelete(ids []int64, operator string) (int64, error) {
	if len(ids) == 0 {
		return 0, fmt.Errorf("用户ID列表不能为空")
	}
	count, err := s.userRepo.BatchDelete(ids)
	if err != nil {
		return 0, err
	}
	log.Printf("批量删除用户 count=%d by=%s", count, operator)
	return count, nil
}

// ListAudit 分页查询审计日志
func (s *AdminService) ListAudit(username, action, resource, startDate, endDate string, page, pageSize int) ([]*model.AuditLog, int, error) {
	offset, page, pageSize := util.Paginate(page, pageSize)

	logs, err := s.auditRepo.List(username, action, resource, startDate, endDate, offset, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("查询审计日志失败: %w", err)
	}

	total, err := s.auditRepo.Count(username, action, resource, startDate, endDate)
	if err != nil {
		return nil, 0, fmt.Errorf("统计审计日志总数失败: %w", err)
	}

	return logs, total, nil
}

// DeleteAudit 按条件清理审计日志，返回删除条数。
func (s *AdminService) DeleteAudit(username, action, resource, startDate, endDate string) (int64, error) {
	return s.auditRepo.Delete(username, action, resource, startDate, endDate)
}

// ClearAllAudit 清空全部审计日志。
func (s *AdminService) ClearAllAudit() error {
	return s.auditRepo.ClearAll()
}

// ListMyAudit 查询当前用户自己的操作日志（我的日志）
// actionType: "" 仅用户操作(写操作)；"all" 全部
func (s *AdminService) ListMyAudit(userID int64, actionType, startDate, endDate string, page, pageSize int) ([]*model.AuditLog, int, error) {
	offset, page, pageSize := util.Paginate(page, pageSize)
	logs, err := s.auditRepo.ListByUser(userID, actionType, startDate, endDate, offset, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("查询日志失败: %w", err)
	}
	total, err := s.auditRepo.CountByUser(userID, actionType, startDate, endDate)
	if err != nil {
		return nil, 0, fmt.Errorf("统计日志总数失败: %w", err)
	}
	return logs, total, nil
}

// DeleteMyLog 删除当前用户自己的日志（id<=0 清空操作日志）
func (s *AdminService) DeleteMyLog(userID int64, id int64) (int64, error) {
	return s.auditRepo.DeleteByUser(userID, id)
}

// ListSnapshots 列出可恢复的操作快照
func (s *AdminService) ListSnapshots(limit int) ([]*model.AuditSnapshot, error) {
	return s.auditRepo.ListSnapshots(limit)
}

// RestoreSnapshot 恢复操作（按快照回滚到操作前状态）
// 返回已恢复条数与恢复说明。
func (s *AdminService) RestoreSnapshot(snapshotID int64, operator string) (int64, error) {
	snap, err := s.auditRepo.GetSnapshotByID(snapshotID)
	if err != nil {
		return 0, err
	}
	if snap == nil {
		return 0, fmt.Errorf("快照不存在")
	}
	if snap.Restored == 1 {
		return 0, fmt.Errorf("该操作已恢复过")
	}

	switch snap.Operation {
	case "user.status":
		// 回滚用户状态到 before（仅当非 sys_admin）
		id, _ := strconv.ParseInt(snap.RecordID, 10, 64)
		if id <= 0 {
			return 0, fmt.Errorf("快照记录标识无效")
		}
		before := snap.BeforeJSON
		if before != "active" && before != "disabled" && before != "rejected" && before != "pending" {
			return 0, fmt.Errorf("快照状态值非法: %s", before)
		}
		if err := s.userRepo.RestoreUserStatus(id, before); err != nil {
			return 0, err
		}
	default:
		return 0, fmt.Errorf("不支持恢复该操作: %s", snap.Operation)
	}

	if err := s.auditRepo.MarkSnapshotRestored(snap.ID, operator); err != nil {
		return 0, err
	}
	return 1, nil
}

// GetSettings 获取所有系统配置
func (s *AdminService) GetSettings() ([]*model.SystemSetting, error) {
	return s.settingsRepo.GetAll()
}

// GetFeatureSwitches 获取全部功能开关（feature.* 前缀）
func (s *AdminService) GetFeatureSwitches() (map[string]string, error) {
	return s.settingsRepo.GetByPrefix("feature.")
}

// UpdateFeatureSwitch 更新单个功能开关
func (s *AdminService) UpdateFeatureSwitch(key, value, updatedBy string) error {
	return s.settingsRepo.Upsert(key, value, updatedBy)
}

// UpdateSettings 批量更新系统配置
func (s *AdminService) UpdateSettings(settings map[string]string, updatedBy string) error {
	for key, value := range settings {
		if err := s.settingsRepo.Upsert(key, value, updatedBy); err != nil {
			return fmt.Errorf("更新配置 %s 失败: %w", key, err)
		}
	}
	log.Printf("系统配置已更新 by=%s count=%d", updatedBy, len(settings))
	return nil
}

// ImportStudentRow 导入学生行数据
type ImportStudentRow struct {
	Username        string // 学号
	DisplayName     string // 姓名
	College         string // 院系
	Major           string // 专业
	ClassName       string // 班级
	EnrollmentDate  string // 入学时间
	EnrollmentYear  string // 入学年份
	Role            string // Excel 中的角色，仅允许学生
	Gender          string // 性别
	Campus          string // 校区
	EducationLevel  string // 学历层次
	StudyDuration   string // 学制
	ExpectedGrad    string // 预期毕业时间
	StudyMode       string // 学习形式
	Ethnicity       string // 民族
	PoliticalStatus string // 政治面貌
	BirthDate       string // 出生年月
}

// ImportStudentsResult 导入结果
type ImportStudentsResult struct {
	Total   int                            `json:"total"`
	Success int                            `json:"success"`
	Failed  int                            `json:"failed"`
	Details []repository.BatchCreateResult `json:"details"`
}

// ImportStudents 从 xlsx 批量导入学生
func (s *AdminService) ImportStudents(rows []*ImportStudentRow, defaultPassword, importerRole, importerScope, importerOwnerID string) (*ImportStudentsResult, error) {
	if len(rows) == 0 {
		return nil, fmt.Errorf("导入数据为空")
	}
	if len(rows) > 5000 {
		return nil, fmt.Errorf("单次最多导入 5000 名学生")
	}

	defaultPassword = strings.TrimSpace(defaultPassword)
	if defaultPassword != "" && len([]rune(defaultPassword)) < 6 {
		return nil, fmt.Errorf("统一初始密码不能少于 6 位")
	}

	// 统一密码只计算一次哈希；留空时每名学生使用自己的学号并分别加密。
	sharedHash := ""
	if defaultPassword != "" {
		h, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("密码加密失败: %w", err)
		}
		sharedHash = string(h)
	}

	students := make([]*model.User, 0, len(rows))
	results := make([]repository.BatchCreateResult, 0, len(rows))
	for _, source := range rows {
		username := strings.TrimSpace(source.Username)
		displayName := strings.TrimSpace(source.DisplayName)
		role := strings.ToLower(strings.TrimSpace(source.Role))
		if username == "" {
			results = append(results, repository.BatchCreateResult{
				DisplayName: displayName, Error: "学号不能为空",
			})
			continue
		}
		if displayName == "" {
			results = append(results, repository.BatchCreateResult{
				Username: username, Error: "姓名不能为空",
			})
			continue
		}
		if role != "" && role != "学生" && role != "student" {
			results = append(results, repository.BatchCreateResult{
				Username: username, DisplayName: displayName,
				Error: "角色必须为学生",
			})
			continue
		}

		college := strings.TrimSpace(source.College)
		className := strings.TrimSpace(source.ClassName)
		ownerScope := "college"
		ownerID := college
		if importerRole != "sys_admin" && importerRole != "school_admin" {
			if importerScope != "" {
				ownerScope = importerScope
			}
			if importerOwnerID != "" {
				ownerID = importerOwnerID
			}
		} else if ownerID == "" {
			ownerID = className
		}
		if ownerID == "" {
			ownerID = "default"
		}

		passwordHash := sharedHash
		if passwordHash == "" {
			hash, err := bcrypt.GenerateFromPassword([]byte(username), bcrypt.DefaultCost)
			if err != nil {
				results = append(results, repository.BatchCreateResult{
					Username: username, DisplayName: displayName,
					Error: "密码加密失败",
				})
				continue
			}
			passwordHash = string(hash)
		}

		u := &model.User{
			Username:        username,
			DisplayName:     displayName,
			Role:            "student",
			OwnerScope:      ownerScope,
			OwnerID:         ownerID,
			College:         college,
			Major:           strings.TrimSpace(source.Major),
			ClassName:       className,
			EnrollmentDate:  strings.TrimSpace(source.EnrollmentDate),
			EnrollmentYear:  strings.TrimSpace(source.EnrollmentYear),
			Gender:          strings.TrimSpace(source.Gender),
			Campus:          defaultCampus(strings.TrimSpace(source.Campus)),
			EducationLevel:  strings.TrimSpace(source.EducationLevel),
			StudyDuration:   strings.TrimSpace(source.StudyDuration),
			ExpectedGrad:    strings.TrimSpace(source.ExpectedGrad),
			StudyMode:       strings.TrimSpace(source.StudyMode),
			Ethnicity:       strings.TrimSpace(source.Ethnicity),
			PoliticalStatus: strings.TrimSpace(source.PoliticalStatus),
			BirthDate:       strings.TrimSpace(source.BirthDate),
			Status:          "active",
			PasswordHash:    passwordHash,
		}
		students = append(students, u)
	}

	if len(students) > 0 {
		created, err := s.userRepo.BatchCreateStudents(students)
		if err != nil {
			return nil, fmt.Errorf("批量创建学生失败: %w", err)
		}
		results = append(results, created...)
	}

	successCount := 0
	failedCount := 0
	for _, r := range results {
		if r.Success {
			successCount++
		} else {
			failedCount++
		}
	}

	return &ImportStudentsResult{
		Total:   len(results),
		Success: successCount,
		Failed:  failedCount,
		Details: results,
	}, nil
}

// ParseStudentXLSX 解析学生名单 xlsx 文件，返回结构化行数据
func (s *AdminService) ParseStudentXLSX(r io.ReaderAt, size int64) ([]*ImportStudentRow, error) {
	rows, err := util.ParseXLSX(r, size)
	if err != nil {
		return nil, fmt.Errorf("解析 xlsx 失败: %w", err)
	}

	if len(rows) < 2 {
		return nil, fmt.Errorf("数据不足（至少需要表头+1行数据）")
	}

	// 在前 10 行内定位表头，兼容模板顶部存在标题或说明行。
	headerIndex := -1
	var header util.XLSXRow
	for i := 0; i < len(rows) && i < 10; i++ {
		hasUsername := false
		hasDisplayName := false
		for _, value := range rows[i] {
			v := strings.TrimSpace(strings.TrimPrefix(value, "\ufeff"))
			if v == "学号" || strings.HasPrefix(v, "学号") {
				hasUsername = true
			}
			if v == "姓名" || strings.HasPrefix(v, "姓名") {
				hasDisplayName = true
			}
		}
		if hasUsername && hasDisplayName {
			headerIndex = i
			header = rows[i]
			break
		}
	}
	if headerIndex < 0 {
		return nil, fmt.Errorf("表头缺少必要列：学号、姓名")
	}

	colMap := make(map[string]string)
	for col, name := range header {
		v := strings.TrimSpace(strings.TrimPrefix(name, "\ufeff"))
		switch {
		case v == "学号" || strings.HasPrefix(v, "学号"):
			colMap["username"] = col
		case v == "姓名" || strings.HasPrefix(v, "姓名"):
			colMap["display_name"] = col
		case v == "院系" || strings.HasPrefix(v, "院系") || v == "专业院系" || strings.HasPrefix(v, "专业院系"):
			colMap["college"] = col
		case v == "专业" || strings.HasPrefix(v, "专业"):
			colMap["major"] = col
		case v == "班级" || strings.HasPrefix(v, "班级"):
			colMap["class_name"] = col
		case v == "入学时间" || v == "入学日期" || v == "入校时间" || strings.HasPrefix(v, "入学时间") || strings.HasPrefix(v, "入学日期") || strings.HasPrefix(v, "入校时间"):
			colMap["enrollment_date"] = col
		case v == "入学年份" || strings.HasPrefix(v, "入学年份"):
			colMap["enrollment_year"] = col
		case v == "角色" || strings.HasPrefix(v, "角色"):
			colMap["role"] = col
		case v == "性别" || strings.HasPrefix(v, "性别"):
			colMap["gender"] = col
		case v == "校区" || strings.HasPrefix(v, "校区"):
			colMap["campus"] = col
		case v == "学历层次" || strings.HasPrefix(v, "学历层次"):
			colMap["education_level"] = col
		case v == "学制" || strings.HasPrefix(v, "学制"):
			colMap["study_duration"] = col
		case v == "预期毕业时间" || strings.HasPrefix(v, "预期毕业时间"):
			colMap["expected_graduation"] = col
		case v == "学习形式" || strings.HasPrefix(v, "学习形式"):
			colMap["study_mode"] = col
		case v == "民族" || strings.HasPrefix(v, "民族"):
			colMap["ethnicity"] = col
		case v == "政治面貌" || strings.HasPrefix(v, "政治面貌"):
			colMap["political_status"] = col
		case v == "出生年月" || strings.HasPrefix(v, "出生年月"):
			colMap["birth_date"] = col
		}
	}

	if colMap["username"] == "" || colMap["display_name"] == "" {
		return nil, fmt.Errorf("表头缺少必要列：学号、姓名")
	}

	var result []*ImportStudentRow
	for i := headerIndex + 1; i < len(rows); i++ {
		row := rows[i]
		username := strings.TrimSpace(row[colMap["username"]])
		displayName := strings.TrimSpace(row[colMap["display_name"]])
		if username == "" && displayName == "" {
			continue
		}
		r := &ImportStudentRow{
			Username:        username,
			DisplayName:     displayName,
			College:         cell(row, colMap, "college"),
			Major:           cell(row, colMap, "major"),
			ClassName:       cell(row, colMap, "class_name"),
			EnrollmentDate:  cell(row, colMap, "enrollment_date"),
			EnrollmentYear:  cell(row, colMap, "enrollment_year"),
			Role:            cell(row, colMap, "role"),
			Gender:          cell(row, colMap, "gender"),
			Campus:          cell(row, colMap, "campus"),
			EducationLevel:  cell(row, colMap, "education_level"),
			StudyDuration:   cell(row, colMap, "study_duration"),
			ExpectedGrad:    cell(row, colMap, "expected_graduation"),
			StudyMode:       cell(row, colMap, "study_mode"),
			Ethnicity:       cell(row, colMap, "ethnicity"),
			PoliticalStatus: cell(row, colMap, "political_status"),
			BirthDate:       cell(row, colMap, "birth_date"),
		}
		result = append(result, r)
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("表格中没有可导入的学生数据")
	}

	return result, nil
}

// cell 安全读取 Excel 行单元格：列不存在时返回空串
// colMap 值即为表头名（XLSXRow 以表头名为键）
func cell(row util.XLSXRow, colMap map[string]string, key string) string {
	header, ok := colMap[key]
	if !ok || header == "" {
		return ""
	}
	return strings.TrimSpace(row[header])
}

// defaultCampus 校区为空时默认会峰校区
func defaultCampus(c string) string {
	if c == "" {
		return "会峰校区"
	}
	return c
}
