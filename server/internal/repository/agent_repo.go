package repository

import (
	"database/sql"
	"fmt"

	"github.com/dll/wxx/server/internal/model"
)

var agentAllowedUpdateColumns = map[string]bool{
	"name":           true,
	"description":    true,
	"agent_type":     true,
	"system_prompt":  true,
	"model_provider": true,
	"model_name":     true,
	"temperature":    true,
	"max_tokens":     true,
	"status":         true,
}

func sanitizeUpdateColumn(col string) string {
	if agentAllowedUpdateColumns[col] {
		return col
	}
	return ""
}

// AgentRepo 智能体数据访问
type AgentRepo struct {
	db *sql.DB
}

// NewAgentRepo 创建智能体 repo
func NewAgentRepo(db *sql.DB) *AgentRepo {
	return &AgentRepo{db: db}
}

// Create 创建智能体
func (r *AgentRepo) Create(agent *model.Agent) (int64, error) {
	result, err := r.db.Exec(
		`INSERT INTO agents (agent_id, name, description, agent_type, system_prompt,
		 model_provider, model_name, temperature, max_tokens, status, config_json)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		agent.AgentID, agent.Name, agent.Description, agent.AgentType,
		agent.SystemPrompt, agent.ModelProvider, agent.ModelName,
		agent.Temperature, agent.MaxTokens, agent.Status, agent.ConfigJSON,
	)
	if err != nil {
		return 0, fmt.Errorf("创建智能体失败: %w", err)
	}
	return result.LastInsertId()
}

// Update 更新智能体（按 agent_id）
func (r *AgentRepo) Update(agentID string, updates map[string]interface{}) error {
	sets := ""
	args := make([]interface{}, 0)

	for col, val := range updates {
		col = sanitizeUpdateColumn(col)
		if col == "" {
			continue
		}
		if sets != "" {
			sets += ", "
		}
		sets += col + " = ?"
		args = append(args, val)
	}

	if sets == "" {
		return fmt.Errorf("无更新字段")
	}

	query := "UPDATE agents SET " + sets + ", updated_at = CURRENT_TIMESTAMP WHERE agent_id = ?"
	args = append(args, agentID)

	_, err := r.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("更新智能体失败: %w", err)
	}
	return nil
}

// ListAll 列出所有智能体
func (r *AgentRepo) ListAll() ([]*model.Agent, error) {
	rows, err := r.db.Query(
		`SELECT id, agent_id, name, description, agent_type, system_prompt,
		        model_provider, model_name, temperature, max_tokens,
		        status, config_json, created_at, updated_at
		 FROM agents
		 ORDER BY
		   CASE agent_type WHEN 'qa' THEN 0 WHEN 'policy' THEN 1 WHEN 'emotion' THEN 2 ELSE 3 END,
		   created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("查询智能体列表失败: %w", err)
	}
	defer rows.Close()

	var agents []*model.Agent
	for rows.Next() {
		a := &model.Agent{}
		if err := rows.Scan(
			&a.ID, &a.AgentID, &a.Name, &a.Description, &a.AgentType,
			&a.SystemPrompt, &a.ModelProvider, &a.ModelName,
			&a.Temperature, &a.MaxTokens, &a.Status, &a.ConfigJSON,
			&a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, err
		}
		agents = append(agents, a)
	}
	return agents, rows.Err()
}

// GetByAgentID 根据 agent_id 查询
func (r *AgentRepo) GetByAgentID(agentID string) (*model.Agent, error) {
	a := &model.Agent{}
	err := r.db.QueryRow(
		`SELECT id, agent_id, name, description, agent_type, system_prompt,
		        model_provider, model_name, temperature, max_tokens,
		        status, config_json, created_at, updated_at
		 FROM agents
		 WHERE agent_id = ?`, agentID,
	).Scan(
		&a.ID, &a.AgentID, &a.Name, &a.Description, &a.AgentType,
		&a.SystemPrompt, &a.ModelProvider, &a.ModelName,
		&a.Temperature, &a.MaxTokens, &a.Status, &a.ConfigJSON,
		&a.CreatedAt, &a.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return a, nil
}

// Delete 删除智能体
func (r *AgentRepo) Delete(agentID string) error {
	_, err := r.db.Exec("DELETE FROM agents WHERE agent_id = ?", agentID)
	if err != nil {
		return fmt.Errorf("删除智能体失败: %w", err)
	}
	return nil
}
