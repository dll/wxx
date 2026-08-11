package repository

import (
	"database/sql"

	dbutil "github.com/dll/wxx/server/internal/db"
	"github.com/dll/wxx/server/internal/model"
)

// ModelConfigRepo 用户 AI 模型配置数据访问
type ModelConfigRepo struct {
	db *sql.DB
}

// NewModelConfigRepo 创建模型配置 repo
func NewModelConfigRepo(db *sql.DB) *ModelConfigRepo {
	return &ModelConfigRepo{db: db}
}

// GetByUserID 获取用户模型配置，不存在则返回 nil
func (r *ModelConfigRepo) GetByUserID(userID int64) (*model.UserModelConfig, error) {
	cfg := &model.UserModelConfig{}
	err := r.db.QueryRow(
		`SELECT id, user_id, deepseek_key, deepseek_model, deepseek_temp, deepseek_max_tokens,
		 zhipu_key, zhipu_model, zhipu_temp, zhipu_max_tokens,
		 xunfei_app_id, xunfei_key, xunfei_secret, xunfei_model, xunfei_temp, xunfei_max_tokens,
		 default_provider, created_at, updated_at
		 FROM user_model_configs WHERE user_id = ?`, userID,
	).Scan(&cfg.ID, &cfg.UserID,
		&cfg.DeepseekKey, &cfg.DeepseekModel, &cfg.DeepseekTemp, &cfg.DeepseekMaxTok,
		&cfg.ZhipuKey, &cfg.ZhipuModel, &cfg.ZhipuTemp, &cfg.ZhipuMaxTok,
		&cfg.XunfeiAppID, &cfg.XunfeiKey, &cfg.XunfeiSecret, &cfg.XunfeiModel, &cfg.XunfeiTemp, &cfg.XunfeiMaxTok,
		&cfg.DefaultProvider, &cfg.CreatedAt, &cfg.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	cfg.DeepseekKey, err = decrypt(cfg.DeepseekKey)
	if err != nil {
		return nil, err
	}
	cfg.ZhipuKey, err = decrypt(cfg.ZhipuKey)
	if err != nil {
		return nil, err
	}
	cfg.XunfeiKey, err = decrypt(cfg.XunfeiKey)
	if err != nil {
		return nil, err
	}
	cfg.XunfeiSecret, err = decrypt(cfg.XunfeiSecret)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

// Upsert 创建或更新用户模型配置
func (r *ModelConfigRepo) Upsert(cfg *model.UserModelConfig) error {
	var err error
	dk, err := encrypt(cfg.DeepseekKey)
	if err != nil {
		return err
	}
	zk, err := encrypt(cfg.ZhipuKey)
	if err != nil {
		return err
	}
	xk, err := encrypt(cfg.XunfeiKey)
	if err != nil {
		return err
	}
	xs, err := encrypt(cfg.XunfeiSecret)
	if err != nil {
		return err
	}

	stmt := `INSERT INTO user_model_configs (user_id, deepseek_key, deepseek_model, deepseek_temp, deepseek_max_tokens,
		 zhipu_key, zhipu_model, zhipu_temp, zhipu_max_tokens,
		 xunfei_app_id, xunfei_key, xunfei_secret, xunfei_model, xunfei_temp, xunfei_max_tokens,
		 default_provider, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		 ON CONFLICT(user_id) DO UPDATE SET
		 deepseek_key=excluded.deepseek_key,
		 deepseek_model=excluded.deepseek_model,
		 deepseek_temp=excluded.deepseek_temp,
		 deepseek_max_tokens=excluded.deepseek_max_tokens,
		 zhipu_key=excluded.zhipu_key,
		 zhipu_model=excluded.zhipu_model,
		 zhipu_temp=excluded.zhipu_temp,
		 zhipu_max_tokens=excluded.zhipu_max_tokens,
		 xunfei_app_id=excluded.xunfei_app_id,
		 xunfei_key=excluded.xunfei_key,
		 xunfei_secret=excluded.xunfei_secret,
		 xunfei_model=excluded.xunfei_model,
		 xunfei_temp=excluded.xunfei_temp,
		 xunfei_max_tokens=excluded.xunfei_max_tokens,
		 default_provider=excluded.default_provider,
		 updated_at=CURRENT_TIMESTAMP`
	_, err = r.db.Exec(dbutil.AdaptForDriver(stmt, dbutil.DriverOf(r.db)),
		cfg.UserID,
		dk, cfg.DeepseekModel, cfg.DeepseekTemp, cfg.DeepseekMaxTok,
		zk, cfg.ZhipuModel, cfg.ZhipuTemp, cfg.ZhipuMaxTok,
		cfg.XunfeiAppID, xk, xs, cfg.XunfeiModel, cfg.XunfeiTemp, cfg.XunfeiMaxTok,
		cfg.DefaultProvider,
	)
	return err
}
