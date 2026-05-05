package service

import (
	"os"
	"testing"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/testutil"
)

func setupAgentServiceTestDB(t *testing.T) *AgentService {
	t.Helper()

	db := testutil.NewTestDB(t)
	t.Cleanup(func() { db.Close() })

	// 执行 agents 表迁移（005_agents.sql）
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

	return NewAgentService(repository.NewAgentRepo(db))
}

func TestAgentService_Create_Success(t *testing.T) {
	svc := setupAgentServiceTestDB(t)

	req := &model.AgentCreateRequest{
		AgentID:   "agent-1",
		Name:      "测试智能体",
		AgentType: "qa",
	}
	agent, err := svc.Create(req)
	if err != nil {
		t.Fatalf("Create 失败: %v", err)
	}
	if agent.AgentID != "agent-1" {
		t.Errorf("期望 AgentID=agent-1，得到 %s", agent.AgentID)
	}
	if agent.Status != "active" {
		t.Errorf("新智能体 status 应为 active，得到 %s", agent.Status)
	}
	if agent.ConfigJSON != "{}" {
		t.Errorf("ConfigJSON 应为 {}，得到 %s", agent.ConfigJSON)
	}
}

func TestAgentService_Create_Duplicate(t *testing.T) {
	svc := setupAgentServiceTestDB(t)

	req := &model.AgentCreateRequest{AgentID: "dup", Name: "重复", AgentType: "qa"}
	svc.Create(req) // 第一次成功

	_, err := svc.Create(req) // 第二次应报错
	if err == nil {
		t.Fatal("重复 agent_id 应返回错误")
	}
}

func TestAgentService_Create_DefaultValues(t *testing.T) {
	svc := setupAgentServiceTestDB(t)

	req := &model.AgentCreateRequest{
		AgentID:   "agent-default",
		Name:      "默认值测试",
		AgentType: "emotion",
	}
	agent, err := svc.Create(req)
	if err != nil {
		t.Fatalf("Create 失败: %v", err)
	}
	if agent.Temperature != 0.7 {
		t.Errorf("默认温度应为 0.7，得到 %f", agent.Temperature)
	}
	if agent.MaxTokens != 2048 {
		t.Errorf("默认 MaxTokens 应为 2048，得到 %d", agent.MaxTokens)
	}
}

func TestAgentService_Update_Success(t *testing.T) {
	svc := setupAgentServiceTestDB(t)

	svc.Create(&model.AgentCreateRequest{AgentID: "upd", Name: "原始", AgentType: "qa"})

	newName := "更新后"
	updateReq := &model.AgentUpdateRequest{Name: &newName}
	updated, err := svc.Update("upd", updateReq)
	if err != nil {
		t.Fatalf("Update 失败: %v", err)
	}
	if updated.Name != "更新后" {
		t.Errorf("期望 Name=更新后，得到 %s", updated.Name)
	}
}

func TestAgentService_Update_NotFound(t *testing.T) {
	svc := setupAgentServiceTestDB(t)

	newName := "x"
	_, err := svc.Update("nonexistent", &model.AgentUpdateRequest{Name: &newName})
	if err == nil {
		t.Fatal("更新不存在的智能体应返回错误")
	}
}

func TestAgentService_Update_NoFields(t *testing.T) {
	svc := setupAgentServiceTestDB(t)

	svc.Create(&model.AgentCreateRequest{AgentID: "nochg", Name: "不变", AgentType: "qa"})

	updated, err := svc.Update("nochg", &model.AgentUpdateRequest{})
	if err != nil {
		t.Fatalf("无字段更新不应报错: %v", err)
	}
	if updated.Name != "不变" {
		t.Errorf("无字段更新不应修改数据，得到 %s", updated.Name)
	}
}

func TestAgentService_List(t *testing.T) {
	svc := setupAgentServiceTestDB(t)

	svc.Create(&model.AgentCreateRequest{AgentID: "a1", Name: "A1", AgentType: "qa"})
	svc.Create(&model.AgentCreateRequest{AgentID: "a2", Name: "A2", AgentType: "emotion"})

	agents, err := svc.List()
	if err != nil {
		t.Fatalf("List 失败: %v", err)
	}
	// 005_agents.sql 种子数据包含 3 条，加上我们创建的 2 条
	if len(agents) < 2 {
		t.Errorf("期望至少 2 个智能体，得到 %d", len(agents))
	}
}

func TestAgentService_Get_Success(t *testing.T) {
	svc := setupAgentServiceTestDB(t)

	svc.Create(&model.AgentCreateRequest{AgentID: "get-me", Name: "目标", AgentType: "policy"})

	agent, err := svc.Get("get-me")
	if err != nil {
		t.Fatalf("Get 失败: %v", err)
	}
	if agent.Name != "目标" {
		t.Errorf("期望 Name=目标，得到 %s", agent.Name)
	}
}

func TestAgentService_Get_NotFound(t *testing.T) {
	svc := setupAgentServiceTestDB(t)

	_, err := svc.Get("nonexistent")
	if err == nil {
		t.Fatal("不存在的智能体应返回错误")
	}
}

func TestAgentService_Delete_Success(t *testing.T) {
	svc := setupAgentServiceTestDB(t)

	svc.Create(&model.AgentCreateRequest{AgentID: "del-me", Name: "待删", AgentType: "qa"})

	if err := svc.Delete("del-me"); err != nil {
		t.Fatalf("Delete 失败: %v", err)
	}

	_, err := svc.Get("del-me")
	if err == nil {
		t.Error("删除后应无法获取")
	}
}

func TestAgentService_Delete_NotFound(t *testing.T) {
	svc := setupAgentServiceTestDB(t)

	err := svc.Delete("nonexistent")
	if err == nil {
		t.Fatal("删除不存在的智能体应返回错误")
	}
}

func TestAgentService_GetActiveAgents(t *testing.T) {
	svc := setupAgentServiceTestDB(t)

	svc.Create(&model.AgentCreateRequest{AgentID: "act-1", Name: "活跃1", AgentType: "qa"})
	svc.Create(&model.AgentCreateRequest{AgentID: "act-2", Name: "活跃2", AgentType: "emotion"})

	// 创建第三个然后手动停用
	svc.Create(&model.AgentCreateRequest{AgentID: "inact", Name: "停用", AgentType: "qa"})
	inactive := "inactive"
	svc.Update("inact", &model.AgentUpdateRequest{Status: &inactive})

	active, err := svc.GetActiveAgents()
	if err != nil {
		t.Fatalf("GetActiveAgents 失败: %v", err)
	}
	// 只统计我们自己创建的 active 智能体 + 种子数据中的 active
	if len(active) < 2 {
		t.Errorf("期望至少 2 个活跃智能体，得到 %d", len(active))
	}
	for _, a := range active {
		if a.Status != "active" {
			t.Errorf("所有结果 status 应=active，得到 %s: %s", a.AgentID, a.Status)
		}
	}
}
