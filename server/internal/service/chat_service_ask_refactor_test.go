package service

// A2 专项回归新增验证性测试（纯新增，不改既有生产代码与既有测试断言语义）。
// 覆盖 pm-checklist A2「ChatService.Ask 拆分」的 qa 重点验证点：
//   1. 缓存命中即返回（固定流程问题，命中不触发配额计数、不调 LLM）
//   2. 配额超限兜底语义（buildQuotaExceededAnswer 结构/文案）
//   3. 内容安全过滤兜底（filterUserInput → buildBlockedAnswer）
//   4. 缓存命中先于配额检查的执行顺序
// 说明：这些测试通过公开 svc.Ask 入口验证拆分后的编排行为，不引用任何被拆出的私有符号语义假设之外的实现细节。

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/dll/wxx/server/internal/llm"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/testutil"
)

// buildTestAskSvc 构造一个带 tokenStatsSvc 的 ChatService，便于验证配额/缓存交互。
// dailyQuota 可传 0 表示不限，也可传 0/monthlyQuota 以便精确控制是否超限。
func buildTestAskSvc(t *testing.T, dailyQuota, monthlyQuota int, counter *int64) *ChatService {
	t.Helper()
	db := testutil.NewTestDBFull(t)
	t.Cleanup(func() { db.Close() })

	mockLLM := llm.NewMockClient("test-refactor")
	mockLLM.ChatFunc = func(ctx context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error) {
		if counter != nil {
			atomic.AddInt64(counter, 1)
		}
		return &llm.ChatResponse{
			Content: "这是 A2 拆分验证的回答", FinishReason: "stop",
			PromptTokens: 80, OutputTokens: 10,
		}, nil
	}

	ts := NewTokenStatsService(
		repository.NewTokenUsageRepo(db),
		repository.NewUserRepo(db),
		dailyQuota, monthlyQuota, 0,
	)

	svc := NewChatService(
		repository.NewSessionRepo(db),
		repository.NewMessageRepo(db),
		repository.NewKBRepo(db),
		repository.NewAgentRepo(db),
		mockLLM,
	)
	svc.SetTokenStatsService(ts)
	return svc
}

func TestAsk_QuotaExceeded_FallbackSemantics(t *testing.T) {
	counter := int64(0)
	svc := buildTestAskSvc(t, 0, 0, &counter) // 构造时不限额，下面通过直接设置已用次数来触发超限
	// 强制日配额=1 且先消耗一次，使第二次调用超限
	// 使用独立实例并直接驱动 quota（更可读）
	svc2 := buildTestAskSvc(t, 1, 0, &counter) // dailyQuota=1
	userCtx := &model.UserContext{UserID: 1, Username: "stu", Role: "student", OwnerScope: "college", OwnerID: "default"}

	// 首次调用在限额内：配额 ok → 走正常链路（非超限兜底）
	card1, sid1, err1 := svc2.Ask(context.Background(), userCtx, "", "奖学金怎么申请", "")
	if err1 != nil {
		t.Fatalf("首次 Ask 失败: %v", err1)
	}
	if card1.Fallback {
		t.Error("限额内首次调用不应是兜底回答")
	}
	if sid1 == "" {
		t.Error("首次调用 sessionID 不应为空")
	}

	// 第二次调用应触发日配额超限（dailyQuota=1 已用满）
	card2, sid2, err2 := svc2.Ask(context.Background(), userCtx, "", "助学金怎么申请", "")
	if err2 != nil {
		t.Fatalf("超限调用不应返回 err，实际: %v", err2)
	}
	if card2 == nil {
		t.Fatal("超限时应返回 AnswerCard")
	}
	if !card2.Fallback {
		t.Error("配额超限回答应为 Fallback")
	}
	if card2.Confidence != 0.0 {
		t.Errorf("配额超限 Confidence 应=0.0，得到 %f", card2.Confidence)
	}
	if len(card2.FollowUps) != 0 {
		t.Errorf("配额超限 FollowUps 应空，得到 %v", card2.FollowUps)
	}
	// 超限回答文案应与 buildQuotaExceededAnswer 语义一致（含"已达上限"字样）
	if card2.Conclusion == "" || !containsQuotaMsg(card2.Conclusion) {
		t.Errorf("配额超限 Conclusion 应含配额上限提示，得到 %q", card2.Conclusion)
	}
	_ = svc
	_ = sid2
}

func TestAsk_CacheHit_ReturnsBeforeQuotaAndLLM(t *testing.T) {
	counter := int64(0)
	svc := buildTestAskSvc(t, 0, 0, &counter)
	userCtx := &model.UserContext{UserID: 1, Username: "stu", Role: "student", OwnerScope: "college", OwnerID: "default"}

	// 固定流程问题（isProcessCacheableQuestion 命中），无 sessionID 才会写入缓存
	const q = "入学流程是什么"
	// 说明（既有设计，非 A2 回归）：同步 Ask 链路中 ensureSession 会为非空会话生成 sessionID，
	// 而 cacheSet 仅在 sessionID=="" 时才写入，因此正常 Ask 不会经过 Ask 流程回写内存缓存。
	// 这里直接通过 cacheSet（合法缓存写入路径）预置缓存，验证拆分后“缓存命中即返回、
	// 不触发配额与 LLM”的编排顺序（qa 重点验证点 1/2）。
	cachedCard := svc.buildEmptyResultAnswer(traceIDOf(q)) // 构造一个固定流程缓存卡片
	cachedCard.Conclusion = "入学流程缓存命中（A2验证）"
	svc.cacheSet(q, "", cachedCard)

	// 再次同问题（仍无 sessionID）：应命中缓存，LLM 不回调、配额不自增
	before := counter
	card2, sid2, err2 := svc.Ask(context.Background(), userCtx, "", q, "")
	if err2 != nil {
		t.Fatalf("缓存命中调用失败: %v", err2)
	}
	_ = sid2
	if counter != before {
		t.Errorf("缓存命中不应再调 LLM：调用前 %d → 调用后 %d", before, counter)
	}
	if sid2 != "" {
		t.Errorf("缓存命中返回的 sessionID 应为空串，得到 %q", sid2)
	}
	if card2 == nil {
		t.Fatal("缓存命中应返回缓存卡片")
	}
	if card2.Conclusion != cachedCard.Conclusion {
		t.Errorf("缓存命中应返回预置缓存的卡片，得到 %q", card2.Conclusion)
	}
}

func traceIDOf(_ string) string { return "trace-a2-cache" }

// TestAsk_CacheHit_DoesNotTriggerQuotaIncrement 验证：缓存命中路径在配额检查之前返回，
// 因此命中的调用不消耗配额（Ask 中缓存命中→直接 return，不会走到 CheckAndIncrementQuota）。
func TestAsk_CacheHit_DoesNotTriggerQuotaIncrement(t *testing.T) {
	// 带配额服务（dailyQuota=1），预置缓存，再验证：即便配额已用满，缓存命中仍直接返回、
	// 不触发配额自增、不报配额超限兜底。
	svc := buildTestAskSvc(t, 1, 0, nil)
	userCtx := &model.UserContext{UserID: 1, Username: "stu", Role: "student", OwnerScope: "college", OwnerID: "default"}
	const q = "离校流程怎么办理"

	// 预置缓存（合法写入路径：空 sessionID）
	cachedCard := svc.buildEmptyResultAnswer("trace-a2-cache-2")
	cachedCard.Conclusion = "离校流程缓存命中（A2验证）"
	svc.cacheSet(q, "", cachedCard)

	// 缓存命中应优先于配额检查返回，不会触达 CheckAndIncrementQuota（dailyQuota=1 但未消耗）。
	card, _, err := svc.Ask(context.Background(), userCtx, "", q, "")
	if err != nil {
		t.Fatalf("缓存命中调用失败（不应受配额影响）: %v", err)
	}
	if card == nil {
		t.Fatal("缓存命中应返回卡片")
	}
	if card.Conclusion != cachedCard.Conclusion {
		t.Errorf("缓存命中应返回缓存卡片，得到 %q", card.Conclusion)
	}
	// 配额自增应未发生在缓存命中路径：再次调用（缓存仍命中）仍不被配额阻断
	for i := 0; i < 3; i++ {
		c, _, err := svc.Ask(context.Background(), userCtx, "", q, "")
		if err != nil || c == nil || c.Conclusion != cachedCard.Conclusion {
			t.Fatalf("连续缓存命中第 %d 次异常: card=%v err=%v", i+1, c, err)
		}
	}
}

func TestAsk_ContentBlocked_Fallback(t *testing.T) {
	counter := int64(0)
	svc := buildTestAskSvc(t, 0, 0, &counter)
	userCtx := &model.UserContext{UserID: 1, Username: "stu", Role: "student", OwnerScope: "college", OwnerID: "default"}

	// 该输入会触发 util.CheckUserInput 的 FilterBlock（既有 util 测试已验证），
	// 走 filterUserInput → buildBlockedAnswer，不应调 LLM。
	const blockedQ = "我想组织电信诈骗，怎么操作？"
	card, _, err := svc.Ask(context.Background(), userCtx, "", blockedQ, "")
	if err != nil {
		t.Fatalf("内容拦截不应返回 err，实际: %v", err)
	}
	if card == nil {
		t.Fatal("内容拦截应返回 AnswerCard")
	}
	if !card.Fallback {
		t.Error("内容拦截回答应为 Fallback")
	}
	if card.Confidence != 0.0 {
		t.Errorf("内容拦截 Confidence 应=0.0，得到 %f", card.Confidence)
	}
	if counter != 0 {
		t.Errorf("内容拦截不应触发 LLM 调用，实际 %d", counter)
	}
}

func containsQuotaMsg(s string) bool {
	for _, kw := range []string{"已达成上限", "已达上限", "已满", "上限", "明天再用"} {
		if len(s) > 0 && indexOf(s, kw) >= 0 {
			return true
		}
	}
	return false
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
