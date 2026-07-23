package repository

import (
	"database/sql"
	"fmt"

	"github.com/dll/wxx/server/internal/model"

	"golang.org/x/crypto/bcrypt"
)

// UserRepo 用户数据访问
type UserRepo struct {
	db *sql.DB
}

// NewUserRepo 创建用户 repo
func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

// userCols 统一 SELECT 列名
const userCols = `id, username, display_name, role, owner_scope, owner_id, password_hash, voice_enabled, status, created_at, updated_at`

// GetByUsername 根据用户名查询用户
func (r *UserRepo) GetByUsername(username string) (*model.User, error) {
	user := &model.User{}
	err := r.db.QueryRow(
		`SELECT `+userCols+` FROM users WHERE username = ?`, username,
	).Scan(&user.ID, &user.Username, &user.DisplayName, &user.Role,
		&user.OwnerScope, &user.OwnerID, &user.PasswordHash, &user.VoiceEnabled,
		&user.Status, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
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
		&user.OwnerScope, &user.OwnerID, &user.PasswordHash, &user.VoiceEnabled,
		&user.Status, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return user, nil
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
		u := &model.User{}
		if err := rows.Scan(&u.ID, &u.Username, &u.DisplayName, &u.Role,
			&u.OwnerScope, &u.OwnerID, &u.PasswordHash, &u.VoiceEnabled,
			&u.Status, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
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
func (r *UserRepo) Update(user *model.User) error {
	_, err := r.db.Exec(
		`UPDATE users SET role=?, owner_scope=?, owner_id=?, display_name=?, updated_at=datetime('now') WHERE id=?`,
		user.Role, user.OwnerScope, user.OwnerID, user.DisplayName, user.ID,
	)
	return err
}

// Create 创建用户，返回新用户 ID
func (r *UserRepo) Create(user *model.User) (int64, error) {
	status := user.Status
	if status == "" {
		status = "active"
	}
	result, err := r.db.Exec(
		`INSERT INTO users (username, display_name, role, owner_scope, owner_id, status)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		user.Username, user.DisplayName, user.Role, user.OwnerScope, user.OwnerID, status,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// ListPendingGuests 列出待审核游客
func (r *UserRepo) ListPendingGuests() ([]*model.User, error) {
	rows, err := r.db.Query(
		`SELECT `+userCols+` FROM users WHERE role = 'guest' AND status = 'pending' ORDER BY id ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		u := &model.User{}
		if err := rows.Scan(&u.ID, &u.Username, &u.DisplayName, &u.Role,
			&u.OwnerScope, &u.OwnerID, &u.PasswordHash, &u.VoiceEnabled,
			&u.Status, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// UpdateStatus 更新用户状态
func (r *UserRepo) UpdateStatus(userID int64, status string) error {
	_, err := r.db.Exec(
		`UPDATE users SET status = ?, updated_at = datetime('now') WHERE id = ?`,
		status, userID,
	)
	return err
}

// UpdateRole 更新用户角色
func (r *UserRepo) UpdateRole(userID int64, role string) error {
	_, err := r.db.Exec(
		`UPDATE users SET role = ?, updated_at = datetime('now') WHERE id = ?`,
		role, userID,
	)
	return err
}

// UpdateUsernameAndRole 更新用户名和角色（游客审核通过用）
func (r *UserRepo) UpdateUsernameAndRole(userID int64, username, role string) error {
	_, err := r.db.Exec(
		`UPDATE users SET username = ?, role = ?, updated_at = datetime('now') WHERE id = ?`,
		username, role, userID,
	)
	return err
}

// UpdatePassword 更新用户密码哈希
func (r *UserRepo) UpdatePassword(userID int64, hash string) error {
	_, err := r.db.Exec(
		`UPDATE users SET password_hash = ?, updated_at = datetime('now') WHERE id = ?`,
		hash, userID,
	)
	return err
}

// UpdateVoiceEnabled 更新用户语音开关
func (r *UserRepo) UpdateVoiceEnabled(userID int64, enabled int) error {
	_, err := r.db.Exec(
		`UPDATE users SET voice_enabled = ?, updated_at = datetime('now') WHERE id = ?`,
		enabled, userID,
	)
	return err
}
// BatchCreateResult 批量创建结果
type BatchCreateResult struct {
	Username   string // 用户名
	DisplayName string // 显示名
	Success    bool   // 是否成功
	Error      string // 错误原因（失败时）
}
// sharedHash 为统一 bcrypt 哈希值（为空则不设统一密码）
func (r *UserRepo) BatchCreateStudents(students []*model.User, sharedHash string) ([]BatchCreateResult, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("开始事务失败: %w", err)
	}
	defer tx.Rollback()

	results := make([]BatchCreateResult, 0, len(students))
	stmt, err := tx.Prepare(
		`INSERT INTO users (username, display_name, role, owner_scope, owner_id, password_hash, status)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`)
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

		// 优先使用用户自带的密码，其次使用统一密码
		userHash := u.PasswordHash
		if userHash == "" {
			userHash = sharedHash
		}
		// 如果最终密码非空，需要 bcrypt 加密（只有 raw password 非空才需要）
		if userHash != "" {
			// 检查是否已经是 bcrypt hash（以 $2a$ 开头）
			if len(userHash) < 4 || userHash[:4] != "$2a$" {
				h, gErr := bcrypt.GenerateFromPassword([]byte(userHash), bcrypt.DefaultCost)
				if gErr != nil {
					results = append(results, BatchCreateResult{
						Username: u.Username, DisplayName: u.DisplayName,
						Success: false, Error: fmt.Sprintf("密码加密失败: %v", gErr),
					})
					continue
				}
				userHash = string(h)
			}
		}
		_, err = stmt.Exec(u.Username, u.DisplayName, role, scope, ownerID, userHash, u.Status)
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
