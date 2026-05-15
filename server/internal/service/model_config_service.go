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

// Get 获取用户模型配置
func (s *ModelConfigService) Get(userID int64) (*model.UserModelConfig, error) {
	cfg, err := s.repo.GetByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("查询模型配置失败: %w", err)
	}
	return cfg, nil
}

// Save 保存用户模型配置
func (s *ModelConfigService) Save(userID int64, req *model.ModelConfigSaveRequest) (*model.UserModelConfig, error) {
	// 参数验证
	if err := s.validate(req); err != nil {
		return nil, err
	}

	cfg := &model.UserModelConfig{
		UserID:          userID,
		DeepseekKey:     req.DeepseekKey,
		DeepseekModel:   req.DeepseekModel,
		DeepseekTemp:    req.DeepseekTemp,
		DeepseekMaxTok:  req.DeepseekMaxTok,
		ZhipuKey:        req.ZhipuKey,
		ZhipuModel:      req.ZhipuModel,
		ZhipuTemp:       req.ZhipuTemp,
		ZhipuMaxTok:     req.ZhipuMaxTok,
		XunfeiAppID:     req.XunfeiAppID,
		XunfeiKey:       req.XunfeiKey,
		XunfeiSecret:    req.XunfeiSecret,
		XunfeiModel:     req.XunfeiModel,
		XunfeiTemp:      req.XunfeiTemp,
		XunfeiMaxTok:    req.XunfeiMaxTok,
		DefaultProvider: req.DefaultProvider,
	}

	if err := s.repo.Upsert(cfg); err != nil {
		return nil, fmt.Errorf("保存模型配置失败: %w", err)
	}

	// 读回完整记录
	saved, _ := s.repo.GetByUserID(userID)
	log.Printf("用户模型配置已保存 user_id=%d default_provider=%s", userID, req.DefaultProvider)
	return saved, nil
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
