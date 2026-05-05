package repository

import (
	"os"
	"testing"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/testutil"
)

func setupAgentTestDB(t *testing.T) *AgentRepo {
	t.Helper()

	db := testutil.NewTestDB(t)
	t.Cleanup(func() { db.Close() })

	// 额外执行 agents 表迁移（005_agents.sql）
	migrationPath := "../../migrations/005_agents.sql"
	sqlContent, err := os.ReadFile(migrationPath)
	if err != nil {
		migrationPath = "../migrations/005_agents.sql"
		sqlContent, err = os.ReadFile(migrationPath)
		if err != nil {
			t.Fatalf("读取 agents 迁移文件失败: %v", err)
		}
	}
	for _, stmt := range testutil.SplitSQL(string(sqlContent)) {
		if stmt == "" {
			continue
		}
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("执行 agents 迁移失败: %v\nSQL: %s", err, stmt)
		}
	}

	return NewAgentRepo(db)
}

func TestAgentRepo_Create(t *testing.T) {
	repo := setupAgentTestDB(t)

	agent := &model.Agent{
		AgentID:     "test-agent-1",
		Name:        "测试智能体",
		Description: "用于测试的智能体",
		AgentType:   "qa",
		Temperature: 0.7,
		MaxTokens:   2048,
		Status:      "active",
		ConfigJSON:  "{}",
	}
	id, err := repo.Create(agent)
	if err != nil {
		t.Fatalf("Create 失败: %v", err)
	}
	if id <= 0 {
		t.Errorf("期望有效 id，得到 %d", id)
	}

	// 回查验证
	created, err := repo.GetByAgentID("test-agent-1")
	if err != nil {
		t.Fatalf("GetByAgentID 失败: %v", err)
	}
	if created.Name != "测试智能体" {
		t.Errorf("期望 Name=测试智能体，得到 %s", created.Name)
	}
	if created.AgentType != "qa" {
		t.Errorf("期望 AgentType=qa，得到 %s", created.AgentType)
	}
}

func TestAgentRepo_GetByAgentID_NotFound(t *testing.T) {
	repo := setupAgentTestDB(t)

	agent, err := repo.GetByAgentID("nonexistent")
	if err != nil {
		t.Fatalf("GetByAgentID 失败: %v", err)
	}
	if agent != nil {
		t.Error("不存在的 agent_id 应返回 nil")
	}
}

func TestAgentRepo_ListAll(t *testing.T) {
	repo := setupAgentTestDB(t)

	// 创建两个智能体
	repo.Create(&model.Agent{AgentID: "agent-a", Name: "智能体A", AgentType: "qa", Status: "active", ConfigJSON: "{}"})
	repo.Create(&model.Agent{AgentID: "agent-b", Name: "智能体B", AgentType: "emotion", Status: "active", ConfigJSON: "{}"})

	agents, err := repo.ListAll()
	if err != nil {
		t.Fatalf("ListAll 失败: %v", err)
	}
	// 005_agents.sql 包含 3 条种子数据，加上我们创建的 2 条
	if len(agents) < 2 {
		t.Errorf("期望至少 2 个智能体，得到 %d", len(agents))
	}
	// qa 类型应排在 emotion 前面
	if agents[0].AgentType != "qa" {
		t.Errorf("期望第一个是 qa 类型，得到 %s", agents[0].AgentType)
	}
}

func TestAgentRepo_Update(t *testing.T) {
	repo := setupAgentTestDB(t)

	repo.Create(&model.Agent{AgentID: "agent-upd", Name: "原始名称", AgentType: "qa", Status: "active", ConfigJSON: "{}"})

	updates := map[string]interface{}{
		"name":   "更新后名称",
		"status": "inactive",
	}
	if err := repo.Update("agent-upd", updates); err != nil {
		t.Fatalf("Update 失败: %v", err)
	}

	updated, _ := repo.GetByAgentID("agent-upd")
	if updated.Name != "更新后名称" {
		t.Errorf("期望 Name=更新后名称，得到 %s", updated.Name)
	}
	if updated.Status != "inactive" {
		t.Errorf("期望 Status=inactive，得到 %s", updated.Status)
	}
}

func TestAgentRepo_Delete(t *testing.T) {
	repo := setupAgentTestDB(t)

	repo.Create(&model.Agent{AgentID: "agent-del", Name: "待删除", AgentType: "qa", Status: "active", ConfigJSON: "{}"})

	if err := repo.Delete("agent-del"); err != nil {
		t.Fatalf("Delete 失败: %v", err)
	}

	// 确认已删除
	agent, err := repo.GetByAgentID("agent-del")
	if err != nil {
		t.Fatalf("GetByAgentID 失败: %v", err)
	}
	if agent != nil {
		t.Error("删除后应返回 nil")
	}
}
