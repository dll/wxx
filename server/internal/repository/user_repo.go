package repository

import (
	"database/sql"

	"github.com/dll/wxx/server/internal/model"
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
const userCols = `id, username, display_name, role, owner_scope, owner_id, password_hash, voice_enabled, created_at, updated_at`

// GetByUsername 根据用户名查询用户
func (r *UserRepo) GetByUsername(username string) (*model.User, error) {
	user := &model.User{}
	err := r.db.QueryRow(
		`SELECT `+userCols+` FROM users WHERE username = ?`, username,
	).Scan(&user.ID, &user.Username, &user.DisplayName, &user.Role,
		&user.OwnerScope, &user.OwnerID, &user.PasswordHash, &user.VoiceEnabled,
		&user.CreatedAt, &user.UpdatedAt)

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
		&user.CreatedAt, &user.UpdatedAt)

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
			&u.CreatedAt, &u.UpdatedAt); err != nil {
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
	result, err := r.db.Exec(
		`INSERT INTO users (username, display_name, role, owner_scope, owner_id)
		 VALUES (?, ?, ?, ?, ?)`,
		user.Username, user.DisplayName, user.Role, user.OwnerScope, user.OwnerID,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
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
