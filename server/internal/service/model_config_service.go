package service

import (
	"fmt"
	"log"
	"strings"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
)

// 支持的模型列表
var validModels = map[string][]string{
	"deepseek": {"deepseek-chat", "deepseek-reasoner"},
	"zhipu":    {"glm-4", "glm-4-flash", "glm-4-plus"},
	"xunfei":   {"spark-v3.5", "spark-v4.0", "spark-lite"},
}

// ModelConfigService 用户 AI 模型配置业务服务
type ModelConfigService struct {
	repo *repository.ModelConfigRepo
}

// NewModelConfigService 创建模型配置服务
func NewModelConfigService(repo *repository.ModelConfigRepo) *ModelConfigService {
	return &ModelConfigService{repo: repo}
}

// Get 获取用户模型配置（脱敏视图，密钥仅返回掩码）
// 安全修复 SEC-05：绝不向前端回显密钥明文。
func (s *ModelConfigService) Get(userID int64) (*model.ModelConfigView, error) {
	cfg, err := s.repo.GetByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("查询模型配置失败: %w", err)
	}
	if cfg == nil {
		return nil, nil
	}
	return cfg.ToMaskedView(), nil
}

// isMaskedOrEmpty 判断入参是否为空或掩码占位（前端回显未改动的字段）
func isMaskedOrEmpty(v string) bool {
	return v == "" || strings.HasPrefix(v, "****")
}

// Save 保存用户模型配置（返回脱敏视图）
func (s *ModelConfigService) Save(userID int64, req *model.ModelConfigSaveRequest) (*model.ModelConfigView, error) {
	// 参数验证
	if err := s.validate(req); err != nil {
		return nil, err
	}

	// 安全修复 SEC-05：前端可能回显掩码值（未改动密钥）。
	// 若入参为空或掩码占位，则保留数据库中已有密钥，避免用掩码覆盖真实密钥。
	existing, _ := s.repo.GetByUserID(userID)
	deepseekKey := req.DeepseekKey
	zhipuKey := req.ZhipuKey
	xunfeiKey := req.XunfeiKey
	xunfeiSecret := req.XunfeiSecret
	if existing != nil {
		if isMaskedOrEmpty(deepseekKey) {
			deepseekKey = existing.DeepseekKey
		}
		if isMaskedOrEmpty(zhipuKey) {
			zhipuKey = existing.ZhipuKey
		}
		if isMaskedOrEmpty(xunfeiKey) {
			xunfeiKey = existing.XunfeiKey
		}
		if isMaskedOrEmpty(xunfeiSecret) {
			xunfeiSecret = existing.XunfeiSecret
		}
	}

	cfg := &model.UserModelConfig{
		UserID:          userID,
		DeepseekKey:     deepseekKey,
		DeepseekModel:   req.DeepseekModel,
		DeepseekTemp:    req.DeepseekTemp,
		DeepseekMaxTok:  req.DeepseekMaxTok,
		ZhipuKey:        zhipuKey,
		ZhipuModel:      req.ZhipuModel,
		ZhipuTemp:       req.ZhipuTemp,
		ZhipuMaxTok:     req.ZhipuMaxTok,
		XunfeiAppID:     req.XunfeiAppID,
		XunfeiKey:       xunfeiKey,
		XunfeiSecret:    xunfeiSecret,
		XunfeiModel:     req.XunfeiModel,
		XunfeiTemp:      req.XunfeiTemp,
		XunfeiMaxTok:    req.XunfeiMaxTok,
		DefaultProvider: req.DefaultProvider,
	}

	if err := s.repo.Upsert(cfg); err != nil {
		return nil, fmt.Errorf("保存模型配置失败: %w", err)
	}

	// 读回完整记录并脱敏
	saved, _ := s.repo.GetByUserID(userID)
	log.Printf("用户模型配置已保存 user_id=%d default_provider=%s", userID, req.DefaultProvider)
	if saved == nil {
		return nil, nil
	}
	return saved.ToMaskedView(), nil
}

// validate 校验模型参数
func (s *ModelConfigService) validate(req *model.ModelConfigSaveRequest) error {
	// 校验默认模型
	req.DefaultProvider = strings.TrimSpace(req.DefaultProvider)
	if req.DefaultProvider == "" {
		req.DefaultProvider = "deepseek"
	}

	// 校验温度范围
	if req.DeepseekTemp < 0 || req.DeepseekTemp > 2 {
		return fmt.Errorf("DeepSeek 温度参数需在 0~2 之间")
	}
	if req.ZhipuTemp < 0 || req.ZhipuTemp > 2 {
		return fmt.Errorf("智谱温度参数需在 0~2 之间")
	}
	if req.XunfeiTemp < 0 || req.XunfeiTemp > 2 {
		return fmt.Errorf("讯飞温度参数需在 0~2 之间")
	}

	// 校验最大 Token
	if req.DeepseekMaxTok < 1 || req.DeepseekMaxTok > 32768 {
		return fmt.Errorf("DeepSeek 最大 Token 需在 1~32768 之间")
	}
	if req.ZhipuMaxTok < 1 || req.ZhipuMaxTok > 32768 {
		return fmt.Errorf("智谱最大 Token 需在 1~32768 之间")
	}
	if req.XunfeiMaxTok < 1 || req.XunfeiMaxTok > 32768 {
		return fmt.Errorf("讯飞最大 Token 需在 1~32768 之间")
	}

	// 填充默认模型名
	if req.DeepseekModel == "" {
		req.DeepseekModel = "deepseek-chat"
	}
	if req.ZhipuModel == "" {
		req.ZhipuModel = "glm-4"
	}
	if req.XunfeiModel == "" {
		req.XunfeiModel = "spark-v3.5"
	}

	return nil
}
