package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/dll/wxx/server/internal/auth"
	"github.com/dll/wxx/server/internal/model"
)

// UserRepo 用户数据访问
type UserRepo struct {
	db *sql.DB

	// updated_at 活跃刷新节流：同一用户 5 分钟内不重复写库
	lastTouchMu sync.Mutex
	lastTouch   map[int64]time.Time
}

// NewUserRepo 创建用户 repo
func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db, lastTouch: map[int64]time.Time{}}
}

// userCols 统一 SELECT 列名
const userCols = `id, username, display_name, role, owner_scope, owner_id,
	college, major, class_name, enrollment_date, enrollment_year,
	gender, campus, education_level, study_duration, expected_graduation_date,
	study_mode, ethnicity, political_status, birth_date,
	phone, wechat, qq, email,
	password_hash, voice_enabled, status, token_version, consented, must_change_password, ai_api_key_enc, ai_key_provider, created_at, updated_at`

// GetByUsername 根据用户名查询用户
func (r *UserRepo) GetByUsername(username string) (*model.User, error) {
	user := &model.User{}
	err := r.db.QueryRow(
		`SELECT `+userCols+` FROM users WHERE username = ?`, username,
	).Scan(&user.ID, &user.Username, &user.DisplayName, &user.Role,
		&user.OwnerScope, &user.OwnerID, &user.College, &user.Major,
		&user.ClassName, &user.EnrollmentDate, &user.EnrollmentYear,
		&user.Gender, &user.Campus, &user.EducationLevel, &user.StudyDuration, &user.ExpectedGrad,
		&user.StudyMode, &user.Ethnicity, &user.PoliticalStatus, &user.BirthDate,
		&user.Phone, &user.Wechat, &user.QQ, &user.Email,
		&user.PasswordHash, &user.VoiceEnabled,
		&user.Status, &user.TokenVersion, &user.Consented, &user.MustChangePwd, &user.AIAPIKeyEnc, &user.AIKeyProvider, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := r.attachRoles(user); err != nil {
		return nil, err
	}
	return user, nil
}

// GetByID 根据 ID 查询用户
func (r *UserRepo) GetByID(id int64) (*model.User, error) {
	user := &model.User{}
	err := r.db.QueryRow(
		`SELECT `+userCols+` FROM users WHERE id = ?`, id,
	).Scan(&user.ID, &user.Username, &user.DisplayName, &user.Role,
		&user.OwnerScope, &user.OwnerID, &user.College, &user.Major,
		&user.ClassName, &user.EnrollmentDate, &user.EnrollmentYear,
		&user.Gender, &user.Campus, &user.EducationLevel, &user.StudyDuration, &user.ExpectedGrad,
		&user.StudyMode, &user.Ethnicity, &user.PoliticalStatus, &user.BirthDate,
		&user.Phone, &user.Wechat, &user.QQ, &user.Email,
		&user.PasswordHash, &user.VoiceEnabled,
		&user.Status, &user.TokenVersion, &user.Consented, &user.MustChangePwd, &user.AIAPIKeyEnc, &user.AIKeyProvider, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := r.attachRoles(user); err != nil {
		return nil, err
	}
	return user, nil
}

// attachRoles 附带 user_roles 多角色信息（无记录时保持 nil，兼容单角色旧数据）
func (r *UserRepo) attachRoles(user *model.User) error {
	roles, err := r.GetRoles(user.ID)
	if err != nil {
		return err
	}
	if len(roles) > 0 {
		user.Roles = roles
	}
	return nil
}

// List 分页查询用户列表，支持 role/owner_scope/owner_id 过滤
func (r *UserRepo) List(role, ownerScope, ownerID string, offset, limit int) ([]*model.User, error) {
	query := `SELECT ` + userCols + ` FROM users WHERE 1=1`
	var args []interface{}

	if role != "" {
		query += " AND role = ?"
		args = append(args, role)
	}
	if ownerScope != "" {
		query += " AND owner_scope = ?"
		args = append(args, ownerScope)
	}
	if ownerID != "" {
		query += " AND owner_id = ?"
		args = append(args, ownerID)
	}

	query += " ORDER BY id ASC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// ListAdvanced 高级用户查询（搜索+多条件筛选+排序）
func (r *UserRepo) ListAdvanced(q *model.UserQuery) ([]*model.User, int, error) {
	query := `SELECT ` + userCols + ` FROM users WHERE 1=1`
	countQuery := `SELECT COUNT(*) FROM users WHERE 1=1`
	var args []interface{}

	if q.Keyword != "" {
		kw := "%" + q.Keyword + "%"
		query += " AND (display_name LIKE ? OR username LIKE ? OR college LIKE ? OR major LIKE ? OR class_name LIKE ?)"
		countQuery += " AND (display_name LIKE ? OR username LIKE ? OR college LIKE ? OR major LIKE ? OR class_name LIKE ?)"
		args = append(args, kw, kw, kw, kw, kw)
	}
	if q.Role != "" {
		query += " AND role = ?"
		countQuery += " AND role = ?"
		args = append(args, q.Role)
	}
	if q.OwnerScope != "" {
		query += " AND owner_scope = ?"
		countQuery += " AND owner_scope = ?"
		args = append(args, q.OwnerScope)
	}
	if q.OwnerID != "" {
		query += " AND owner_id = ?"
		countQuery += " AND owner_id = ?"
		args = append(args, q.OwnerID)
	}
	if q.College != "" {
		query += " AND college = ?"
		countQuery += " AND college = ?"
		args = append(args, q.College)
	}
	if q.Major != "" {
		query += " AND major = ?"
		countQuery += " AND major = ?"
		args = append(args, q.Major)
	}
	if q.ClassName != "" {
		query += " AND class_name = ?"
		countQuery += " AND class_name = ?"
		args = append(args, q.ClassName)
	}
	if q.EnrollmentYear != "" {
		query += " AND enrollment_year = ?"
		countQuery += " AND enrollment_year = ?"
		args = append(args, q.EnrollmentYear)
	}
	if q.Status != "" {
		query += " AND status = ?"
		countQuery += " AND status = ?"
		args = append(args, q.Status)
	}

	// 总数
	var total int
	if err := r.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计用户数失败: %w", err)
	}

	// 排序
	sortBy := "id"
	switch q.SortBy {
	case "username", "display_name", "role", "college", "major", "class_name", "enrollment_year", "created_at", "status":
		sortBy = q.SortBy
	}
	sortOrder := "ASC"
	if q.SortOrder == "desc" {
		sortOrder = "DESC"
	}
	query += fmt.Sprintf(" ORDER BY %s %s", sortBy, sortOrder)

	// 分页
	if q.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, q.Limit)
	}
	if q.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, q.Offset)
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询用户列表失败: %w", err)
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}
	return users, total, rows.Err()
}

// GetDistinctValues 获取某列的去重值列表（用于筛选项）
func (r *UserRepo) GetDistinctValues(column string, role, ownerScope, ownerID string) ([]string, error) {
	allowedCols := map[string]bool{
		"college":         true,
		"major":           true,
		"class_name":      true,
		"enrollment_year": true,
	}
	if !allowedCols[column] {
		return nil, fmt.Errorf("不支持的列: %s", column)
	}

	query := fmt.Sprintf(`SELECT DISTINCT %s FROM users WHERE %s IS NOT NULL AND %s != ''`, column, column, column)
	var args []interface{}

	if role != "" {
		query += " AND role = ?"
		args = append(args, role)
	}
	if ownerScope != "" {
		query += " AND owner_scope = ?"
		args = append(args, ownerScope)
	}
	if ownerID != "" {
		query += " AND owner_id = ?"
		args = append(args, ownerID)
	}
	query += fmt.Sprintf(" ORDER BY %s ASC", column)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var values []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		values = append(values, v)
	}
	return values, rows.Err()
}

// BatchUpdateStatus 批量更新用户状态
// RestoreUserStatus 恢复用户状态到指定值（审计恢复用；sys_admin 不可被恢复操作改动）
func (r *UserRepo) RestoreUserStatus(id int64, status string) error {
	res, err := r.db.Exec(
		"UPDATE users SET status=?, updated_at=CURRENT_TIMESTAMP WHERE id=? AND role != 'sys_admin'",
		status, id)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("未找到可恢复的用户（或为 sys_admin）: id=%d", id)
	}
	return nil
}

// GetAvatar 查询用户头像 base64（无则空）
func (r *UserRepo) GetAvatar(userID int64) (string, error) {
	var b64 sql.NullString
	err := r.db.QueryRow(
		"SELECT COALESCE(avatar_base64,'') FROM users WHERE id = ?", userID,
	).Scan(&b64)
	if err != nil {
		return "", err
	}
	return b64.String, nil
}

// GetStatusesByIDs 批量查询用户当前状态（用于操作快照）
func (r *UserRepo) GetStatusesByIDs(ids []int64) (map[int64]string, error) {
	if len(ids) == 0 {
		return map[int64]string{}, nil
	}
	placeholders := strings.Repeat("?,", len(ids)-1) + "?"
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}
	rows, err := r.db.Query(
		"SELECT id, status FROM users WHERE id IN ("+placeholders+")", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := make(map[int64]string, len(ids))
	for rows.Next() {
		var id int64
		var status string
		if err := rows.Scan(&id, &status); err != nil {
			return nil, err
		}
		m[id] = status
	}
	return m, rows.Err()
}

func (r *UserRepo) BatchUpdateStatus(ids []int64, status string) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	placeholders := strings.Repeat("?,", len(ids)-1) + "?"
	// 安全修复 S-01：停用/拒绝时递增 token_version 吊销旧令牌；启用等其它状态不动版本。
	setClause := "status=?, updated_at=CURRENT_TIMESTAMP"
	if status == "disabled" || status == "rejected" {
		setClause = "status=?, token_version = token_version + 1, updated_at=CURRENT_TIMESTAMP"
	}
	query := fmt.Sprintf(`UPDATE users SET %s WHERE id IN (%s) AND role != 'sys_admin'`, setClause, placeholders)

	args := make([]interface{}, 0, len(ids)+1)
	args = append(args, status)
	for _, id := range ids {
		args = append(args, id)
	}

	result, err := r.db.Exec(query, args...)
	if err != nil {
		return 0, fmt.Errorf("批量更新状态失败: %w", err)
	}
	return result.RowsAffected()
}

// BatchResetPassword 批量重置用户密码
func (r *UserRepo) BatchResetPassword(ids []int64, hash string) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	placeholders := strings.Repeat("?,", len(ids)-1) + "?"
	// 安全修复 S-01：批量改密递增 token_version，登出相关用户所有旧会话。
	// 任务3（2026-09-01）：管理员重置的密码视为初始密码，同时置位 must_change_password 强制首次登录改密。
	query := fmt.Sprintf(`UPDATE users SET password_hash=?, must_change_password=1, token_version = token_version + 1, updated_at=CURRENT_TIMESTAMP WHERE id IN (%s)`, placeholders)

	args := make([]interface{}, 0, len(ids)+1)
	args = append(args, hash)
	for _, id := range ids {
		args = append(args, id)
	}

	result, err := r.db.Exec(query, args...)
	if err != nil {
		return 0, fmt.Errorf("批量重置密码失败: %w", err)
	}
	return result.RowsAffected()
}

// BatchDelete 批量删除用户
func (r *UserRepo) BatchDelete(ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	tx, err := r.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("开始事务失败: %w", err)
	}
	defer tx.Rollback()

	placeholders := strings.Repeat("?,", len(ids)-1) + "?"
	idArgs := make([]interface{}, len(ids))
	for i, id := range ids {
		idArgs[i] = id
	}

	// 清理关联数据
	assocTables := []string{
		"chat_records",
		"sessions",
		"process_records",
		"plan_progress_records",
		"student_plans",
		"club_members",
		"club_activity_registrations",
		"competition_registrations",
		"student_topic_selections",
		"party_progress",
		"party_study_records",
		"mood_diary",
		"notifications",
		"feedback",
	}
	for _, table := range assocTables {
		if _, err := tx.Exec(fmt.Sprintf(`DELETE FROM %s WHERE user_id IN (%s)`, table, placeholders), idArgs...); err != nil {
			// 表不存在（迁移未覆盖）时跳过，其余错误返回并回滚
			if !isTableNotExist(err) {
				return 0, fmt.Errorf("清理关联表 %s 失败: %w", table, err)
			}
		}
	}

	// 删除用户（排除系统管理员）
	result, err := tx.Exec(fmt.Sprintf(`DELETE FROM users WHERE id IN (%s) AND role != 'sys_admin'`, placeholders), idArgs...)
	if err != nil {
		return 0, fmt.Errorf("批量删除用户失败: %w", err)
	}
	rows, _ := result.RowsAffected()

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("提交事务失败: %w", err)
	}
	return rows, nil
}

// isTableNotExist 判断错误是否为"表不存在"（SQLite: no such table / MySQL: doesn't exist）
func isTableNotExist(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no such table") || strings.Contains(msg, "doesn't exist")
}

// Count 统计用户总数（配合 List 分页）
func (r *UserRepo) Count(role, ownerScope, ownerID string) (int, error) {
	query := `SELECT COUNT(*) FROM users WHERE 1=1`
	var args []interface{}

	if role != "" {
		query += " AND role = ?"
		args = append(args, role)
	}
	if ownerScope != "" {
		query += " AND owner_scope = ?"
		args = append(args, ownerScope)
	}
	if ownerID != "" {
		query += " AND owner_id = ?"
		args = append(args, ownerID)
	}

	var count int
	if err := r.db.QueryRow(query, args...).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// Update 更新用户角色和归属信息
// 安全修复 S-01：角色/归属/状态属敏感变更，一律递增 token_version 使旧令牌失效，
// 避免被降权用户凭旧 JWT 继续以原权限访问。
func (r *UserRepo) Update(user *model.User) error {
	_, err := r.db.Exec(
		`UPDATE users SET role=?, position=?, owner_scope=?, owner_id=?, display_name=?, status=?, token_version = token_version + 1, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
		user.Role, user.Position, user.OwnerScope, user.OwnerID, user.DisplayName, user.Status, user.ID,
	)
	if err != nil {
		return err
	}
	// 多角色：同步 user_roles 关联表（nil 表示本次不改动角色关联）
	if user.Roles != nil {
		if err := r.ReplaceRoles(user.ID, user.Roles, ""); err != nil {
			return err
		}
	}
	return nil
}

// Create 创建用户，返回新用户 ID
func (r *UserRepo) Create(user *model.User) (int64, error) {
	status := user.Status
	if status == "" {
		status = "active"
	}
	result, err := r.db.Exec(
		`INSERT INTO users (
			username, display_name, role, owner_scope, owner_id,
			college, major, class_name, enrollment_date, enrollment_year,
			gender, campus, education_level, study_duration, expected_graduation_date,
			study_mode, ethnicity, political_status, birth_date,
			password_hash, status
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		user.Username, user.DisplayName, user.Role, user.OwnerScope, user.OwnerID,
		user.College, user.Major, user.ClassName, user.EnrollmentDate, user.EnrollmentYear,
		user.Gender, user.Campus, user.EducationLevel, user.StudyDuration, user.ExpectedGrad,
		user.StudyMode, user.Ethnicity, user.PoliticalStatus, user.BirthDate,
		user.PasswordHash, status,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// ListPendingGuests 列出待审核游客
func (r *UserRepo) ListPendingGuests() ([]*model.User, error) {
	rows, err := r.db.Query(
		`SELECT ` + userCols + ` FROM users WHERE role = 'guest' AND status = 'pending' ORDER BY id ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// UpdateStatus 更新用户状态
// 安全修复 S-01：停用/拒绝时递增 token_version，令该用户此前签发的所有 JWT 立即失效。
func (r *UserRepo) UpdateStatus(userID int64, status string) error {
	if status == "disabled" || status == "rejected" {
		_, err := r.db.Exec(
			`UPDATE users SET status = ?, token_version = token_version + 1, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
			status, userID,
		)
		return err
	}
	_, err := r.db.Exec(
		`UPDATE users SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		status, userID,
	)
	return err
}

// UpsertFromContext 根据 JWT 用户上下文 upsert 用户（Vercel 冷启动 JIT 创建）
// 按 username 匹配：存在则更新显示名/角色等信息，不存在则插入。
func (r *UserRepo) UpsertFromContext(userCtx *model.UserContext) error {
	existing, err := r.GetByUsername(userCtx.Username)
	if err != nil {
		return err
	}

	if existing != nil {
		// 用户已存在：数据库为权威来源。
		// 安全修复 S-01：禁止把 JWT 携带的 role/owner_scope/owner_id 回写数据库，
		// 否则被降权/改归属的旧令牌会"复活"过期权限。改为反向以数据库值覆盖 context。

		// 1) 账户状态强制：非 active 状态一律拒绝访问。
		// 修复 GPT56SOL v3 P0-02：此前仅拦截 disabled/rejected，注释明确"pending 游客不拦截"，
		// 使 JIT/审核中账号可带着业务 JWT 访问能力。现在 pending/rejected/disabled 全部拒绝。
		if existing.Status != "active" {
			return model.ErrAccountDisabled
		}

		// 2) 令牌吊销比对：JWT 版本旧于数据库权威版本 → 令牌已被吊销
		if userCtx.TokenVersion < existing.TokenVersion {
			return model.ErrTokenRevoked
		}

		// 3) 以数据库权威值覆盖 context（ID、角色、归属、显示名、状态、令牌版本）
		userCtx.UserID = existing.ID
		userCtx.Role = existing.Role
		userCtx.OwnerScope = existing.OwnerScope
		userCtx.OwnerID = existing.OwnerID
		userCtx.DisplayName = existing.DisplayName
		userCtx.TokenVersion = existing.TokenVersion
		// 安全修复 SEC-02：同意状态以数据库为权威，供 RequireConsent 中间件判断
		userCtx.Consented = existing.Consented == 1

		// 4) 仅刷新 updated_at 以标记活跃，不写回任何权限字段。
		// 节流：同一用户 5 分钟内只写一次，避免高 QPS 下每个请求都触发 UPDATE。
		now := time.Now()
		r.lastTouchMu.Lock()
		last, ok := r.lastTouch[existing.ID]
		if ok && now.Sub(last) < 5*time.Minute {
			r.lastTouchMu.Unlock()
			return nil
		}
		r.lastTouch[existing.ID] = now
		r.lastTouchMu.Unlock()

		_, err = r.db.Exec(
			`UPDATE users SET updated_at=CURRENT_TIMESTAMP WHERE id=?`,
			existing.ID,
		)
		return err
	}

	// 用户不存在，插入新用户（JIT 冷启动兜底）。
	// 修复 GPT56SOL v3 P0-02：此前 JIT 新用户被硬编码为 active+consented，且角色直接取自
	// JWT 自述，凭签名有效的 JWT 即可凭空创建 active 高权限账号。
	// 现在 JIT 创建的账号一律进入 pending 审核态，角色强制降级为 guest（无业务能力），
	// consented 显式 0——必须经管理员审核 + 同意授权后才能获得业务能力。
	// 注意：正常登录链路（密码/SSO）已由 AuthService 显式 Create 并审核，此处仅兜底未知 token。
	status := "pending"
	result, err := r.db.Exec(
		`INSERT INTO users (
			username, display_name, role, owner_scope, owner_id, status, consented
		) VALUES (?, ?, ?, ?, ?, ?, 0)`,
		userCtx.Username, userCtx.DisplayName, "guest",
		userCtx.OwnerScope, userCtx.OwnerID, status,
	)
	if err != nil {
		return err
	}
	newID, _ := result.LastInsertId()
	userCtx.UserID = newID
	// JIT 创建的未知用户：不自动同意授权，role 已强制 guest，consented 以数据库 0 为准，
	// status 同步为 pending 使后续鉴权上下文与数据库一致
	userCtx.Role = "guest"
	userCtx.Status = status
	userCtx.Consented = false
	return nil
}

// SetConsented 持久化用户隐私授权状态
// 安全修复 SEC-02：将同意状态写入数据库，使 RequireConsent 中间件可据此放行。
func (r *UserRepo) SetConsented(userID int64, consented bool) error {
	v := 0
	if consented {
		v = 1
	}
	_, err := r.db.Exec(
		`UPDATE users SET consented = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		v, userID,
	)
	return err
}

// RecordConsent 写入可追溯授权事件，并同步旧字段供兼容中间件读取。
func (r *UserRepo) RecordConsent(userID int64, policyVersion, purpose, vendor, source, traceID string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(`INSERT INTO consent_ledger
		(user_id, policy_version, purpose, vendor, action, source, trace_id)
		VALUES (?, ?, ?, ?, 'granted', ?, ?)`, userID, policyVersion, purpose, vendor, source, traceID); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE users SET consented = 1, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, userID); err != nil {
		return err
	}
	return tx.Commit()
}

// UpdateRole 更新用户角色
// 安全修复 S-01：角色变更递增 token_version 使旧令牌失效。
func (r *UserRepo) UpdateRole(userID int64, role string) error {
	_, err := r.db.Exec(
		`UPDATE users SET role = ?, token_version = token_version + 1, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		role, userID,
	)
	if err != nil {
		return err
	}
	// 多角色：单字段改角色视为"仅保留该角色"，同步关联表
	return r.ReplaceRoles(userID, []string{role}, "")
}

// ── 多角色（user_roles 关联表，2026-09-01）───────────────────────────

// GetRoles 查询用户全部角色（无记录返回空切片，非 error）
func (r *UserRepo) GetRoles(userID int64) ([]string, error) {
	rows, err := r.db.Query(`SELECT role FROM user_roles WHERE user_id = ? ORDER BY id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	roles := []string{}
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, rows.Err()
}

// ReplaceRoles 全量替换用户角色关联（去重去空；主角色同步为权限最高者，变化时吊销旧令牌）。
// grantedBy 为审计人；传空表示系统内部同步（迁移/审核流）。
// 角色合法性由调用方校验（admin_service.checkRoleChangeAuth）。
func (r *UserRepo) ReplaceRoles(userID int64, roles []string, grantedBy string) error {
	seen := make(map[string]bool)
	clean := make([]string, 0, len(roles))
	for _, role := range roles {
		role = strings.TrimSpace(role)
		if role == "" || seen[role] {
			continue
		}
		seen[role] = true
		clean = append(clean, role)
	}

	primary := auth.PrimaryRole(clean)
	var currentRole string
	if err := r.db.QueryRow(`SELECT role FROM users WHERE id = ?`, userID).Scan(&currentRole); err != nil {
		return fmt.Errorf("查询用户主角色失败: %w", err)
	}

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM user_roles WHERE user_id = ?`, userID); err != nil {
		return err
	}
	for _, role := range clean {
		if _, err := tx.Exec(
			`INSERT INTO user_roles (user_id, role, granted_by) VALUES (?, ?, ?)`,
			userID, role, grantedBy,
		); err != nil {
			return err
		}
	}

	if primary != "" && primary != currentRole {
		if _, err := tx.Exec(
			`UPDATE users SET role = ?, token_version = token_version + 1, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
			primary, userID,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// UpdateUsernameAndRole 更新用户名和角色（游客审核通过用）
// 安全修复 S-01：用户名/角色变更递增 token_version，要求用户以新身份重新登录。
func (r *UserRepo) UpdateUsernameAndRole(userID int64, username, role string) error {
	_, err := r.db.Exec(
		`UPDATE users SET username = ?, role = ?, token_version = token_version + 1, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		username, role, userID,
	)
	return err
}

// Delete 删除用户（先清理关联数据，再删除用户记录）
func (r *UserRepo) Delete(userID int64) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("开始事务失败: %w", err)
	}
	defer tx.Rollback()

	// 清理关联数据（按外键依赖顺序）
	assocTables := []string{
		"chat_records",                // 聊天记录
		"sessions",                    // 会话
		"process_records",             // 办事记录
		"plan_progress_records",       // 规划进度
		"student_plans",               // 学生规划
		"club_members",                // 社团成员
		"club_activity_registrations", // 活动报名
		"competition_registrations",   // 竞赛报名
		"student_topic_selections",    // 选题记录
		"party_progress",              // 入党进度
		"party_study_records",         // 党建学习记录
		"mood_diary",                  // 情绪日记（如果表存在）
		"notifications",               // 通知
		"feedback",                    // 反馈
	}
	for _, table := range assocTables {
		// 逐表尝试删除，忽略表不存在的错误
		tx.Exec(fmt.Sprintf(`DELETE FROM %s WHERE user_id = ?`, table), userID)
	}

	// 删除用户本身
	result, err := tx.Exec(`DELETE FROM users WHERE id = ?`, userID)
	if err != nil {
		return fmt.Errorf("删除用户失败: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("用户不存在")
	}

	return tx.Commit()
}

// UpdatePassword 更新用户密码哈希
// 安全修复 S-01：改密递增 token_version，登出该用户所有旧会话（含被盗令牌）。
// UpdatePassword 更新密码（用户自助改密：清除 must_change_password 标记）
func (r *UserRepo) UpdatePassword(userID int64, hash string) error {
	_, err := r.db.Exec(
		`UPDATE users SET password_hash = ?, must_change_password = 0, token_version = token_version + 1, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		hash, userID,
	)
	return err
}

// ResetPasswordWithFlag 管理员重置密码（置位 must_change_password=1，强制首次登录改密）
func (r *UserRepo) ResetPasswordWithFlag(userID int64, hash string) error {
	_, err := r.db.Exec(
		`UPDATE users SET password_hash = ?, must_change_password = 1, token_version = token_version + 1, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		hash, userID,
	)
	return err
}

// UpdateVoiceEnabled 更新用户语音开关
func (r *UserRepo) UpdateVoiceEnabled(userID int64, enabled int) error {
	_, err := r.db.Exec(
		`UPDATE users SET voice_enabled = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		enabled, userID,
	)
	return err
}

// BatchCreateResult 批量创建结果
type BatchCreateResult struct {
	Username    string `json:"username"`     // 用户名
	DisplayName string `json:"display_name"` // 显示名
	Success     bool   `json:"success"`      // 是否成功
	Error       string `json:"error"`        // 错误原因（失败时）
}

// BatchCreateStudents 批量创建学生。PasswordHash 必须由 service 层提前完成 bcrypt 加密。
func (r *UserRepo) BatchCreateStudents(students []*model.User) ([]BatchCreateResult, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("开始事务失败: %w", err)
	}
	defer tx.Rollback()

	results := make([]BatchCreateResult, 0, len(students))
	stmt, err := tx.Prepare(
		`INSERT INTO users (
			username, display_name, role, owner_scope, owner_id,
			college, major, class_name, enrollment_date, enrollment_year,
			gender, campus, education_level, study_duration, expected_graduation_date,
			study_mode, ethnicity, political_status, birth_date,
			password_hash, status, must_change_password
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return nil, fmt.Errorf("预编译语句失败: %w", err)
	}
	defer stmt.Close()

	for _, u := range students {
		// 检查是否已存在
		var exists int
		err := tx.QueryRow(`SELECT COUNT(*) FROM users WHERE username = ?`, u.Username).Scan(&exists)
		if err != nil {
			results = append(results, BatchCreateResult{
				Username: u.Username, DisplayName: u.DisplayName,
				Success: false, Error: fmt.Sprintf("查询冲突失败: %v", err),
			})
			continue
		}
		if exists > 0 {
			results = append(results, BatchCreateResult{
				Username: u.Username, DisplayName: u.DisplayName,
				Success: false, Error: "用户名已存在",
			})
			continue
		}

		role := u.Role
		if role == "" {
			role = "student"
		}
		scope := u.OwnerScope
		if scope == "" {
			scope = "college"
		}
		ownerID := u.OwnerID
		if ownerID == "" {
			ownerID = "default"
		}

		if u.PasswordHash == "" {
			results = append(results, BatchCreateResult{
				Username: u.Username, DisplayName: u.DisplayName,
				Success: false, Error: "密码哈希不能为空",
			})
			continue
		}
		_, err = stmt.Exec(
			u.Username, u.DisplayName, role, scope, ownerID,
			u.College, u.Major, u.ClassName, u.EnrollmentDate, u.EnrollmentYear,
			u.Gender, u.Campus, u.EducationLevel, u.StudyDuration, u.ExpectedGrad,
			u.StudyMode, u.Ethnicity, u.PoliticalStatus, u.BirthDate,
			u.PasswordHash, u.Status, 1, // 导入学生首次登录必须改密
		)
		if err != nil {
			results = append(results, BatchCreateResult{
				Username: u.Username, DisplayName: u.DisplayName,
				Success: false, Error: fmt.Sprintf("插入失败: %v", err),
			})
			continue
		}

		results = append(results, BatchCreateResult{
			Username: u.Username, DisplayName: u.DisplayName,
			Success: true, Error: "",
		})
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交事务失败: %w", err)
	}

	return results, nil
}

// FindRelatedByRole 查找组织关系联系人（可按学院过滤，limit 上限）。
// 用于「我的辅导员/领导」等展示，按 id 升序返回前 limit 名。
func (r *UserRepo) FindRelatedByRole(college, role string, limit int) []model.ContactPerson {
	if limit <= 0 {
		limit = 3
	}
	where := []string{"status = 'active'", "role = ?"}
	args := []any{role}
	if strings.TrimSpace(college) != "" {
		where = append(where, "college = ?")
		args = append(args, strings.TrimSpace(college))
	}
	cond := strings.Join(where, " AND ")
	rows, err := r.db.Query(
		"SELECT id, display_name, role, college, phone, wechat, email FROM users WHERE "+cond+" ORDER BY id ASC LIMIT ?",
		append(args, limit)...,
	)
	if err != nil {
		return []model.ContactPerson{}
	}
	defer rows.Close()
	var list []model.ContactPerson
	for rows.Next() {
		var c model.ContactPerson
		var phone, wechat, email, college sql.NullString
		if err := rows.Scan(&c.ID, &c.Name, &c.Role, &college, &phone, &wechat, &email); err != nil {
			continue
		}
		c.RoleName = c.Role
		c.Phone = phone.String
		c.Wechat = wechat.String
		c.Email = email.String
		list = append(list, c)
	}
	return list
}

// CountByRoleCollege 统计某角色在指定学院的人数（college 为空则全校）。
func (r *UserRepo) CountByRoleCollege(role, college string) int {
	where := []string{"status = 'active'"}
	args := []any{}
	if role != "" {
		where = append(where, "role = ?")
		args = append(args, role)
	}
	if strings.TrimSpace(college) != "" {
		where = append(where, "college = ?")
		args = append(args, strings.TrimSpace(college))
	}
	var n int
	_ = r.db.QueryRow("SELECT COUNT(*) FROM users WHERE "+strings.Join(where, " AND "), args...).Scan(&n)
	return n
}

// CountAllActive 统计全部激活用户数。
func (r *UserRepo) CountAllActive() int {
	var n int
	_ = r.db.QueryRow("SELECT COUNT(*) FROM users WHERE status = 'active'").Scan(&n)
	return n
}

// scanUser 扫描一行用户数据（对齐 userCols 全列）
func scanUser(rows *sql.Rows) (*model.User, error) {
	u := &model.User{}
	err := rows.Scan(&u.ID, &u.Username, &u.DisplayName, &u.Role,
		&u.OwnerScope, &u.OwnerID, &u.College, &u.Major,
		&u.ClassName, &u.EnrollmentDate, &u.EnrollmentYear,
		&u.Gender, &u.Campus, &u.EducationLevel, &u.StudyDuration, &u.ExpectedGrad,
		&u.StudyMode, &u.Ethnicity, &u.PoliticalStatus, &u.BirthDate,
		&u.Phone, &u.Wechat, &u.QQ, &u.Email,
		&u.PasswordHash, &u.VoiceEnabled,
		&u.Status, &u.TokenVersion, &u.Consented, &u.MustChangePwd, &u.AIAPIKeyEnc, &u.AIKeyProvider, &u.CreatedAt, &u.UpdatedAt)
	return u, err
}

// SaveAIKey 保存用户自备 AI API Key（加密存储）
func (r *UserRepo) SaveAIKey(userID int64, encKey, provider string) error {
	_, err := r.db.Exec(
		`UPDATE users SET ai_api_key_enc = ?, ai_key_provider = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		encKey, provider, userID,
	)
	return err
}

// ClearAIKey 清除用户自备 AI API Key
func (r *UserRepo) ClearAIKey(userID int64) error {
	_, err := r.db.Exec(
		`UPDATE users SET ai_api_key_enc = '', ai_key_provider = '', updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		userID,
	)
	return err
}
