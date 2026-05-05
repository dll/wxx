package service

import (
	"fmt"
	"log"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
)

// AgentService 智能体管理服务
type AgentService struct {
	agentRepo *repository.AgentRepo
}

// NewAgentService 创建智能体管理服务
func NewAgentService(agentRepo *repository.AgentRepo) *AgentService {
	return &AgentService{agentRepo: agentRepo}
}

// Create 创建智能体
func (s *AgentService) Create(req *model.AgentCreateRequest) (*model.Agent, error) {
	// 校验 agent_id 唯一性
	exist, err := s.agentRepo.GetByAgentID(req.AgentID)
	if err != nil {
		return nil, fmt.Errorf("校验 agent_id 失败: %w", err)
	}
	if exist != nil {
		return nil, fmt.Errorf("智能体 ID %s 已存在", req.AgentID)
	}

	// 填充默认值
	temperature := req.Temperature
	if temperature == 0 {
		temperature = 0.7
	}
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 2048
	}

	agent := &model.Agent{
		AgentID:       req.AgentID,
		Name:          req.Name,
		Description:   req.Description,
		AgentType:     req.AgentType,
		SystemPrompt:  req.SystemPrompt,
		ModelProvider: req.ModelProvider,
		ModelName:     req.ModelName,
		Temperature:   temperature,
		MaxTokens:     maxTokens,
		Status:        "active",
		ConfigJSON:    "{}",
	}

	id, err := s.agentRepo.Create(agent)
	if err != nil {
		return nil, err
	}
	agent.ID = id

	log.Printf("智能体已创建 id=%s name=%s type=%s", agent.AgentID, agent.Name, agent.AgentType)
	return s.agentRepo.GetByAgentID(req.AgentID)
}

// Update 更新智能体
func (s *AgentService) Update(agentID string, req *model.AgentUpdateRequest) (*model.Agent, error) {
	exist, err := s.agentRepo.GetByAgentID(agentID)
	if err != nil {
		return nil, fmt.Errorf("查询智能体失败: %w", err)
	}
	if exist == nil {
		return nil, fmt.Errorf("智能体 %s 不存在", agentID)
	}

	updates := make(map[string]interface{})

	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.AgentType != nil {
		updates["agent_type"] = *req.AgentType
	}
	if req.SystemPrompt != nil {
		updates["system_prompt"] = *req.SystemPrompt
	}
	if req.ModelProvider != nil {
		updates["model_provider"] = *req.ModelProvider
	}
	if req.ModelName != nil {
		updates["model_name"] = *req.ModelName
	}
	if req.Temperature != nil {
		updates["temperature"] = *req.Temperature
	}
	if req.MaxTokens != nil {
		updates["max_tokens"] = *req.MaxTokens
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}

	if len(updates) == 0 {
		return exist, nil // 没有要更新的字段
	}

	if err := s.agentRepo.Update(agentID, updates); err != nil {
		return nil, err
	}

	log.Printf("智能体已更新 id=%s", agentID)
	return s.agentRepo.GetByAgentID(agentID)
}

// List 列出所有智能体
func (s *AgentService) List() ([]*model.Agent, error) {
	return s.agentRepo.ListAll()
}

// Get 获取单个智能体
func (s *AgentService) Get(agentID string) (*model.Agent, error) {
	agent, err := s.agentRepo.GetByAgentID(agentID)
	if err != nil {
		return nil, err
	}
	if agent == nil {
		return nil, fmt.Errorf("智能体 %s 不存在", agentID)
	}
	return agent, nil
}

// Delete 删除智能体
func (s *AgentService) Delete(agentID string) error {
	exist, err := s.agentRepo.GetByAgentID(agentID)
	if err != nil {
		return err
	}
	if exist == nil {
		return fmt.Errorf("智能体 %s 不存在", agentID)
	}

	if err := s.agentRepo.Delete(agentID); err != nil {
		return err
	}
	log.Printf("智能体已删除 id=%s", agentID)
	return nil
}

// GetActiveAgents 获取所有激活的智能体（供 ChatService 调度）
func (s *AgentService) GetActiveAgents() ([]*model.Agent, error) {
	all, err := s.agentRepo.ListAll()
	if err != nil {
		return nil, err
	}
	active := make([]*model.Agent, 0)
	for _, a := range all {
		if a.Status == "active" {
			active = append(active, a)
		}
	}
	return active, nil
}
