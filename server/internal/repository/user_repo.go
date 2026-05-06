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

// GetByUsername 根据用户名查询用户
func (r *UserRepo) GetByUsername(username string) (*model.User, error) {
	user := &model.User{}
	err := r.db.QueryRow(
		`SELECT id, username, display_name, role, owner_scope, owner_id,
		 created_at, updated_at
		 FROM users WHERE username = ?`, username,
	).Scan(&user.ID, &user.Username, &user.DisplayName, &user.Role,
		&user.OwnerScope, &user.OwnerID,
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
		`SELECT id, username, display_name, role, owner_scope, owner_id,
		 created_at, updated_at
		 FROM users WHERE id = ?`, id,
	).Scan(&user.ID, &user.Username, &user.DisplayName, &user.Role,
		&user.OwnerScope, &user.OwnerID,
		&user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return user, nil
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
