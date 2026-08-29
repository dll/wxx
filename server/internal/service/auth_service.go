package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/dll/wxx/server/internal/config"
	"github.com/dll/wxx/server/internal/jwtutil"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

// smsCodeStore 短信验证码存储：phone -> smsCodeEntry。
// 生产环境应替换为 Redis 等带 TTL 的共享存储，以支持多实例部署。
var smsCodeStore sync.Map

// smsCodeTTL 验证码有效期。超过该时长的验证码视为过期，需重新获取。
const smsCodeTTL = 5 * time.Minute

// smsCodeEntry 保存一条验证码及其过期时间，用于 TTL 与单次消费控制。
type smsCodeEntry struct {
	code      string
	expiresAt time.Time
}

// maskPhone 对手机号脱敏：138****1234
func maskPhone(phone string) string {
	if len(phone) < 7 {
		return phone
	}
	return phone[:3] + "****" + phone[len(phone)-4:]
}

// AuthService 认证业务服务
type AuthService struct {
	cfg      *config.Config
	userRepo *repository.UserRepo
}

// ErrUserNotFound 用户不存在 sentinel error，调用方可用 errors.Is 识别后单独处理
var ErrUserNotFound = errors.New("用户不存在")

// ErrInvalidCredentials 表示账号或密码不正确。
var ErrInvalidCredentials = errors.New("账号或密码错误")

// ErrAccountUnavailable 表示账号尚未获准登录或已被停用。
var ErrAccountUnavailable = errors.New("账号不可用")
var ErrSSONotConfigured = errors.New("SSO 未配置")

// NewAuthService 创建认证服务
func NewAuthService(cfg *config.Config, userRepo *repository.UserRepo) *AuthService {
	return &AuthService{
		cfg:      cfg,
		userRepo: userRepo,
	}
}

// DebugCodeEcho 是否允许在响应中回显短信验证码。
// 仅在非生产（debug）模式下为真，用于本地联调；生产环境永远返回 false，
// 验证码只能通过真实短信通道下发，杜绝验证码经接口泄露。
func (s *AuthService) DebugCodeEcho() bool {
	return !s.cfg.IsRelease()
}

// GetProfile 获取用户完整资料（含学院/专业/入学年份等学业字段）。
// 返回 nil 表示用户不存在。
func (s *AuthService) GetProfile(userID int64) (*model.User, error) {
	return s.userRepo.GetByID(userID)
}

// ProfileDetail 个人详细信息聚合：基本信息 + 联系方式 + 组织关系
type ProfileDetail struct {
	// 基本信息
	UserID         int64  `json:"user_id"`
	Username       string `json:"username"`        // 登录账号
	DisplayName    string `json:"display_name"`    // 姓名
	Role           string `json:"role"`            // 角色
	College        string `json:"college"`         // 学院
	Major          string `json:"major"`           // 专业
	ClassName      string `json:"class_name"`      // 班级
	EnrollmentDate string `json:"enrollment_date"` // 入学时间
	EnrollmentYear string `json:"enrollment_year"` // 入学年份

	// 联系方式
	Phone  string `json:"phone"`
	Wechat string `json:"wechat"`
	QQ     string `json:"qq"`
	Email  string `json:"email"`

	// 头像
	AvatarBase64 string `json:"avatar_base64"`
	AvatarMIME   string `json:"avatar_mime"`

	// 组织关系（按角色不同）
	Supervisors  []model.ContactPerson `json:"supervisors"`  // 上级/领导/辅导员
	Subordinates int                   `json:"subordinates"` // 下级/管辖人数（辅导员带学生数等）
}

// GetProfileDetail 获取个人详细信息（含组织关系）。
// 组织关系：学生 → 同学院辅导员；教师/教辅 → 同学院/学校管理员领导；
// 辅导员 → 所带学生数；管理员 → 管辖用户数。
func (s *AuthService) GetProfileDetail(userID int64) (*ProfileDetail, error) {
	u, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, ErrUserNotFound
	}

	d := &ProfileDetail{
		UserID:         u.ID,
		Username:       u.Username,
		DisplayName:    u.DisplayName,
		Role:           u.Role,
		College:        u.College,
		Major:          u.Major,
		ClassName:      u.ClassName,
		EnrollmentDate: u.EnrollmentDate,
		EnrollmentYear: u.EnrollmentYear,
		Phone:          u.Phone,
		Wechat:         u.Wechat,
		QQ:             u.QQ,
		Email:          u.Email,
		Supervisors:    []model.ContactPerson{},
	}

	// 头像 base64（供前端展示与数字孪生画像图生图原型）
	if avatar, err := s.userRepo.GetAvatar(u.ID); err == nil {
		d.AvatarBase64 = avatar
	}

	switch u.Role {
	case "student", "student_union":
		// 学生：查找同学院辅导员作为"我的辅导员"
		d.Supervisors = s.userRepo.FindRelatedByRole(u.College, "counselor", 3)
	case "counselor":
		// 辅导员：学生是下属，领导是学院管理员
		d.Subordinates = s.userRepo.CountByRoleCollege("student", u.College)
		d.Supervisors = s.userRepo.FindRelatedByRole(u.College, "college_admin", 3)
	case "teacher", "assistant":
		// 教师/教辅：查找同学院管理员领导
		d.Supervisors = s.userRepo.FindRelatedByRole(u.College, "college_admin", 3)
	case "college_admin":
		d.Subordinates = s.userRepo.CountByRoleCollege("", u.College)
		d.Supervisors = s.userRepo.FindRelatedByRole("", "school_admin", 3)
	case "school_admin", "sys_admin":
		d.Subordinates = s.userRepo.CountAllActive()
		d.Supervisors = s.userRepo.FindRelatedByRole("", "sys_admin", 3)
	}

	return d, nil
}

// GuestRegisterEnabled 游客手机注册是否开放（预研期默认关闭，账号走管理员导入）。
func (s *AuthService) GuestRegisterEnabled() bool {
	return s.cfg.EnableGuestRegister
}

// LoginResult 登录结果
type LoginResult struct {
	Token              string `json:"token"`
	ExpiresIn          int    `json:"expires_in"` // 过期时间（秒）
	DisplayName        string `json:"display_name"`
	Role               string `json:"role"`
	MustChangePassword bool   `json:"must_change_password"` // 首次登录需强制改密
}

// SendCode 发送短信验证码（开发环境：仅日志，后续对接短信通道）
func (s *AuthService) SendCode(phone string) (string, error) {
	if phone == "" {
		return "", fmt.Errorf("手机号不能为空")
	}
	if len(phone) != 11 || phone[0] != '1' {
		return "", fmt.Errorf("手机号格式不正确")
	}
	code := fmt.Sprintf("%06d", rand.Intn(1000000))
	smsCodeStore.Store(phone, smsCodeEntry{code: code, expiresAt: time.Now().Add(smsCodeTTL)})
	// 仅记录脱敏手机号，绝不在日志或响应中输出完整验证码，避免验证码泄露。
	log.Printf("[短信验证码] 已下发 手机=%s（%d 分钟内有效）", maskPhone(phone), int(smsCodeTTL.Minutes()))
	return code, nil
}

// VerifyCode 校验短信验证码。
// 安全约束：必须与服务端已下发的验证码精确匹配，且未过期；校验后立即消费（单次有效）。
// 不再存在“任意 6 位数字均通过”的开发后门。
func (s *AuthService) VerifyCode(phone, code string) bool {
	if code == "" || phone == "" {
		return false
	}
	v, ok := smsCodeStore.Load(phone)
	if !ok {
		return false
	}
	entry, ok := v.(smsCodeEntry)
	if !ok {
		smsCodeStore.Delete(phone)
		return false
	}
	// 过期即失效并清除，需重新获取验证码。
	if time.Now().After(entry.expiresAt) {
		smsCodeStore.Delete(phone)
		return false
	}
	// 单次消费：无论成功与否都删除，防止暴力重试。
	smsCodeStore.Delete(phone)
	return entry.code == code
}

// GuestRegister 游客注册
func (s *AuthService) GuestRegister(displayName, phone, code string) (*LoginResult, error) {
	if displayName == "" {
		return nil, fmt.Errorf("昵称不能为空")
	}
	if phone == "" {
		return nil, fmt.Errorf("手机号不能为空")
	}
	if !s.VerifyCode(phone, code) {
		return nil, fmt.Errorf("验证码错误或已过期")
	}

	// 生成唯一用户名
	username := fmt.Sprintf("guest_%s_%d", phone, time.Now().UnixMilli()%10000)

	user := &model.User{
		Username:    username,
		DisplayName: displayName,
		Role:        "guest",
		OwnerScope:  "college",
		OwnerID:     "default",
		Status:      "pending", // 游客默认为 pending，需审核
	}
	id, err := s.userRepo.Create(user)
	if err != nil {
		return nil, fmt.Errorf("创建游客失败: %w", err)
	}
	user.ID = id

	token, err := jwtutil.GenerateToken(s.cfg, user)
	if err != nil {
		return nil, fmt.Errorf("签发 token 失败: %w", err)
	}

	return &LoginResult{
		Token:       token,
		ExpiresIn:   s.cfg.JWTExpireHours * 3600,
		DisplayName: user.DisplayName,
		Role:        user.Role,
	}, nil
}

// ApproveGuest 审核通过游客->学生
func (s *AuthService) ApproveGuest(guestID int64, newUsername string) error {
	if newUsername == "" {
		return fmt.Errorf("学号不能为空")
	}

	user, err := s.userRepo.GetByID(guestID)
	if err != nil {
		return fmt.Errorf("查询用户失败: %w", err)
	}
	if user == nil {
		return ErrUserNotFound
	}
	if user.Role != "guest" {
		return fmt.Errorf("该用户不是游客")
	}
	if user.Status != "pending" {
		return fmt.Errorf("该游客状态不是待审核: %s", user.Status)
	}

	// 检查学号是否已被使用
	existing, err := s.userRepo.GetByUsername(newUsername)
	if err != nil {
		return fmt.Errorf("查询学号失败: %w", err)
	}
	if existing != nil {
		return fmt.Errorf("学号 %s 已被使用", newUsername)
	}

	// 更新用户名和角色
	user.Username = newUsername
	user.Role = "student"
	user.Status = "active"
	if err := s.userRepo.UpdateUsernameAndRole(user.ID, newUsername, "student"); err != nil {
		return fmt.Errorf("更新用户信息失败: %w", err)
	}
	if err := s.userRepo.UpdateStatus(user.ID, "active"); err != nil {
		return fmt.Errorf("更新状态失败: %w", err)
	}

	return nil
}

// RejectGuest 拒绝游客申请
func (s *AuthService) RejectGuest(guestID int64) error {
	user, err := s.userRepo.GetByID(guestID)
	if err != nil {
		return fmt.Errorf("查询用户失败: %w", err)
	}
	if user == nil {
		return ErrUserNotFound
	}
	if user.Role != "guest" {
		return fmt.Errorf("该用户不是游客")
	}

	return s.userRepo.UpdateStatus(user.ID, "rejected")
}

// ListPendingGuests 列出待审核游客
func (s *AuthService) ListPendingGuests() ([]*model.User, error) {
	return s.userRepo.ListPendingGuests()
}

// LoginBySSOTicket 通过统一身份认证票据换取 JWT。
// 支持真实 SSO（OAuth2 authorization_code）与 SSO_MOCK=true 的演示模式。
func (s *AuthService) LoginBySSOTicket(ctx context.Context, ticket string) (*LoginResult, error) {
	if strings.TrimSpace(ticket) == "" {
		return nil, errors.New("SSO ticket 不能为空")
	}
	if s.cfg.SSOMock {
		return s.loginSSOMock(ticket)
	}
	if s.cfg.SSOBaseURL == "" {
		return nil, ErrSSONotConfigured
	}

	tokenResp, err := s.ssoExchangeToken(ctx, ticket)
	if err != nil {
		return nil, err
	}
	accessToken, _ := tokenResp["access_token"].(string)
	if accessToken == "" {
		return nil, errors.New("SSO 未返回 access_token")
	}
	userInfo, err := s.ssoFetchUser(ctx, accessToken)
	if err != nil {
		return nil, err
	}

	username := firstString(userInfo, "username", "student_id", "account", "userId", "user_id")
	if username == "" {
		return nil, errors.New("SSO 用户信息缺少 username/student_id")
	}
	displayName := firstString(userInfo, "name", "display_name", "real_name")
	if displayName == "" {
		displayName = username
	}
	role := firstString(userInfo, "role")
	if !isKnownSSORole(role) {
		role = "student"
	}
	ownerScope := firstString(userInfo, "owner_scope")
	if ownerScope != "school" && ownerScope != "college" && ownerScope != "class" {
		ownerScope = "college"
	}
	ownerID := firstString(userInfo, "owner_id", "college", "dept")
	if ownerID == "" {
		ownerID = "default"
	}

	user, err := s.userRepo.GetByUsername(username)
	if err != nil {
		return nil, fmt.Errorf("查询 SSO 用户失败: %w", err)
	}
	if user == nil {
		user = &model.User{
			Username:    username,
			DisplayName: displayName,
			Role:        role,
			OwnerScope:  ownerScope,
			OwnerID:     ownerID,
			Status:      "active",
		}
		id, err := s.userRepo.Create(user)
		if err != nil {
			return nil, fmt.Errorf("创建 SSO 用户失败: %w", err)
		}
		user.ID = id
	}

	token, err := jwtutil.GenerateToken(s.cfg, user)
	if err != nil {
		return nil, fmt.Errorf("签发 JWT 失败: %w", err)
	}
	return &LoginResult{
		Token:       token,
		ExpiresIn:   s.cfg.JWTExpireHours * 3600,
		DisplayName: user.DisplayName,
		Role:        user.Role,
	}, nil
}

func (s *AuthService) loginSSOMock(ticket string) (*LoginResult, error) {
	sum := sha256.Sum256([]byte(ticket))
	username := fmt.Sprintf("sso_%x", sum[:6])
	user, err := s.userRepo.GetByUsername(username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		user = &model.User{
			Username:    username,
			DisplayName: "SSO 演示用户",
			Role:        "student",
			OwnerScope:  "college",
			OwnerID:     "default",
			Status:      "active",
		}
		id, err := s.userRepo.Create(user)
		if err != nil {
			return nil, err
		}
		user.ID = id
	}
	token, err := jwtutil.GenerateToken(s.cfg, user)
	if err != nil {
		return nil, err
	}
	return &LoginResult{Token: token, ExpiresIn: s.cfg.JWTExpireHours * 3600, DisplayName: user.DisplayName, Role: user.Role}, nil
}

func (s *AuthService) ssoExchangeToken(ctx context.Context, ticket string) (map[string]interface{}, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", ticket)
	form.Set("client_id", s.cfg.SSOClientID)
	form.Set("client_secret", s.cfg.SSOClientSecret)
	form.Set("redirect_uri", s.cfg.SSOCallbackURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(s.cfg.SSOBaseURL, "/")+"/oauth/token", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// 显式超时：ctx 无 deadline 时避免登录请求无限挂起
	ssoClient := &http.Client{Timeout: 10 * time.Second}
	resp, err := ssoClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("SSO token 交换失败: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("SSO token 交换失败 HTTP %d: %s", resp.StatusCode, maskPhone(string(body)))
	}
	var out map[string]interface{}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *AuthService) ssoFetchUser(ctx context.Context, accessToken string) (map[string]interface{}, error) {
	userInfoPath := s.cfg.SSOUserInfoPath
	if userInfoPath == "" {
		userInfoPath = "/userinfo"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(s.cfg.SSOBaseURL, "/")+userInfoPath, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	ssoClient := &http.Client{Timeout: 10 * time.Second}
	resp, err := ssoClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("SSO 用户信息获取失败: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("SSO 用户信息获取失败 HTTP %d", resp.StatusCode)
	}
	var out map[string]interface{}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func firstString(m map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if v, ok := m[key]; ok && v != nil {
			return fmt.Sprint(v)
		}
	}
	return ""
}

func isKnownSSORole(role string) bool {
	switch role {
	case "sys_admin", "school_admin", "college_admin", "counselor", "teacher", "assistant", "student_union", "student", "guest":
		return true
	default:
		return false
	}
}

// RecordConsent 记录用户同意隐私政策与用户协议
// 安全修复 SEC-02：将同意状态持久化到数据库 consented 列，供 RequireConsent 中间件放行。
func (s *AuthService) RecordConsent(userID int64, policyVersion, purpose, vendor, source, traceID string) error {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		log.Printf("查询用户失败(consent): %v", err)
		return fmt.Errorf("查询用户失败(consent): %w", err)
	}
	if user == nil {
		log.Printf("用户不存在(consent): id=%d", userID)
		return fmt.Errorf("用户不存在")
	}
	if strings.TrimSpace(policyVersion) == "" {
		policyVersion = "2026-08-29"
	}
	if strings.TrimSpace(purpose) == "" {
		purpose = "general_service"
	}
	if strings.TrimSpace(source) == "" {
		source = "api"
	}
	if err := s.userRepo.RecordConsent(userID, policyVersion, purpose, vendor, source, traceID); err != nil {
		log.Printf("持久化同意状态失败: user=%s err=%v", user.Username, err)
		return fmt.Errorf("持久化同意状态失败: %w", err)
	}
	log.Printf("用户同意隐私政策与用户协议: user=%s role=%s", user.Username, user.Role)
	return nil
}

// LoginByUsername 通过用户名登录（开发环境简化登录，生产环境走 SSO）
func (s *AuthService) LoginByUsername(username string, _ string, password string) (*LoginResult, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return nil, ErrInvalidCredentials
	}

	// 查询用户
	user, err := s.userRepo.GetByUsername(username)
	if err != nil {
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}

	// 用户不存在 → 返回错误（预发布环境，必须先导入）
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	if user.Status != "active" {
		return nil, fmt.Errorf("%w：当前状态为 %s", ErrAccountUnavailable, user.Status)
	}

	// 所有可登录账号都必须保存 bcrypt 哈希，不允许空密码兜底。
	if user.PasswordHash == "" {
		return nil, ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// 签发 JWT
	token, err := jwtutil.GenerateToken(s.cfg, user)
	if err != nil {
		return nil, fmt.Errorf("签发 token 失败: %w", err)
	}

	return &LoginResult{
		Token:              token,
		ExpiresIn:          s.cfg.JWTExpireHours * 3600,
		DisplayName:        user.DisplayName,
		Role:               user.Role,
		MustChangePassword: user.MustChangePwd == 1,
	}, nil
}

// ChangePassword 用户自助修改密码
func (s *AuthService) ChangePassword(userID int64, oldPassword, newPassword string) error {
	if len(newPassword) < 6 {
		return fmt.Errorf("新密码长度不能少于 6 位")
	}

	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return fmt.Errorf("查询用户失败: %w", err)
	}
	if user == nil {
		return fmt.Errorf("用户不存在")
	}

	// 若已有旧密码，验证旧密码
	if user.PasswordHash != "" {
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword)); err != nil {
			if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
				return fmt.Errorf("旧密码错误")
			}
			return fmt.Errorf("密码验证失败: %w", err)
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("密码加密失败: %w", err)
	}

	if err := s.userRepo.UpdatePassword(userID, string(hash)); err != nil {
		return fmt.Errorf("更新密码失败: %w", err)
	}

	log.Printf("用户修改密码成功: userID=%d", userID)
	return nil
}

// ResetPassword 管理员重置用户密码（仅 sys_admin 可用，调用方已做角色校验）
func (s *AuthService) ResetPassword(adminID int64, targetUserID int64, newPassword string) error {
	if len(newPassword) < 6 {
		return fmt.Errorf("新密码长度不能少于 6 位")
	}

	targetUser, err := s.userRepo.GetByID(targetUserID)
	if err != nil {
		return fmt.Errorf("查询目标用户失败: %w", err)
	}
	if targetUser == nil {
		return fmt.Errorf("目标用户不存在")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("密码加密失败: %w", err)
	}

	if err := s.userRepo.UpdatePassword(targetUserID, string(hash)); err != nil {
		return fmt.Errorf("重置密码失败: %w", err)
	}

	log.Printf("管理员重置密码: adminID=%d targetUser=%s", adminID, targetUser.Username)
	return nil
}

// GetVoiceConfig 获取用户语音开关配置
// 用户不存在时返回 ErrUserNotFound（sentinel），由 handler 决定如何降级
func (s *AuthService) GetVoiceConfig(userID int64) (int, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return 0, fmt.Errorf("查询用户失败: %w", err)
	}
	if user == nil {
		return 0, ErrUserNotFound
	}
	return user.VoiceEnabled, nil
}

// UpdateVoiceConfig 更新用户语音开关
func (s *AuthService) UpdateVoiceConfig(userID int64, enabled int) error {
	if enabled != 0 && enabled != 1 {
		return fmt.Errorf("语音开关值需为 0 或 1")
	}
	if err := s.userRepo.UpdateVoiceEnabled(userID, enabled); err != nil {
		return fmt.Errorf("更新语音配置失败: %w", err)
	}
	log.Printf("用户语音配置已更新: userID=%d voice_enabled=%d", userID, enabled)
	return nil
}

// AIKeyInfo 用户自备 AI Key 信息（不输出明文 Key）
type AIKeyInfo struct {
	Bound    bool   `json:"bound"`    // 是否已绑定
	Provider string `json:"provider"` // zhipu / deepseek
}

// GetAIKeyInfo 获取当前用户 AI Key 绑定状态
func (s *AuthService) GetAIKeyInfo(userID int64) (*AIKeyInfo, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("用户不存在")
	}
	return &AIKeyInfo{Bound: user.AIAPIKeyEnc != "", Provider: user.AIKeyProvider}, nil
}

// SaveAIKey 保存用户自备 AI Key（加密后入库）
func (s *AuthService) SaveAIKey(userID int64, provider, apiKey string) error {
	if apiKey == "" {
		return fmt.Errorf("API Key 不能为空")
	}
	if provider != "zhipu" && provider != "deepseek" {
		return fmt.Errorf("仅支持 zhipu / deepseek 提供商")
	}
	enc, err := repository.EncryptField(strings.TrimSpace(apiKey))
	if err != nil {
		return fmt.Errorf("加密失败: %w", err)
	}
	return s.userRepo.SaveAIKey(userID, enc, provider)
}

// ClearAIKey 清除用户自备 AI Key
func (s *AuthService) ClearAIKey(userID int64) error {
	return s.userRepo.ClearAIKey(userID)
}
