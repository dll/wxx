package auth

import "testing"

// TestGuestRoleNoBusinessCapability P0-01 纵深防御回归：
// 修复前 guest 角色持有 SelfChat/SelfKnowledgeRead/SelfProcessRead 三项业务能力，
// pending 游客即使被签发 JWT 也能调用对话/知识库等接口。
// 修复后 guest 仅保留 SelfGuestRead（浏览公开信息），与中间件 status 校验形成双保险。
func TestGuestRoleNoBusinessCapability(t *testing.T) {
	businessCaps := []Capability{
		SelfChat,
		SelfKnowledgeRead,
		SelfProcessRead,
		SelfBriefingRead,
		SelfSessionRead,
		SelfVoice,
	}
	for _, cap := range businessCaps {
		if HasCapability("guest", cap) {
			t.Errorf("guest 角色不应拥有业务能力 %s", cap)
		}
	}
	if !HasCapability("guest", SelfGuestRead) {
		t.Error("guest 应保留 SelfGuestRead（浏览公开信息）")
	}
}
