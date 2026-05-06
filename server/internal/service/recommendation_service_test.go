package service

import (
	"testing"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/testutil"
)

func TestExtractKeywords_Empty(t *testing.T) {
	keywords := extractKeywords(nil)
	if len(keywords) != 0 {
		t.Errorf("空输入应返回 nil，得到 %v", keywords)
	}
	keywords = extractKeywords([]string{})
	if len(keywords) != 0 {
		t.Errorf("空切片应返回 nil，得到 %v", keywords)
	}
}

func TestExtractKeywords_Basic(t *testing.T) {
	questions := []string{
		"奖学金如何申请",
		"奖学金需要什么条件",
		"请问奖学金评定标准是什么",
	}
	keywords := extractKeywords(questions)
	if len(keywords) == 0 {
		t.Fatal("应提取到关键词")
	}
	// "奖学" 或 "学金" 应该出现（bigram）
	foundScholarship := false
	for _, kw := range keywords {
		if kw == "奖学" || kw == "学金" || kw == "奖" || kw == "金" {
			foundScholarship = true
			break
		}
	}
	if !foundScholarship {
		t.Errorf("应包含奖学金相关关键词，得到 %v", keywords)
	}
}

func TestExtractKeywords_StopWordsFiltered(t *testing.T) {
	questions := []string{
		"请问怎么办理入学手续",
		"请问如何申请",
		"你好我想知道哪里可以办理",
	}
	keywords := extractKeywords(questions)
	// "请问"、"如何"、"怎么"、"哪里"、"你好"、"我想"、"知道"、"可以" 都是停用词，应被过滤
	for _, kw := range keywords {
		if isStopWord(kw) {
			t.Errorf("停用词 %q 不应出现在关键词中: %v", kw, keywords)
		}
	}
}

func TestExtractKeywords_MaxFive(t *testing.T) {
	// 生成多个不同的问题，确保关键词数量不超过 5
	questions := []string{
		"入学手续办理流程",
		"离校手续办理流程",
		"奖学金申请条件",
		"宿舍管理规定",
		"图书馆开放时间",
		"食堂就餐指南",
		"社团活动报名",
	}
	keywords := extractKeywords(questions)
	if len(keywords) > 5 {
		t.Errorf("关键词不应超过 5 个，得到 %d: %v", len(keywords), keywords)
	}
}

func TestSegmentChinese_Simple(t *testing.T) {
	words := segmentChinese("奖学金申请")
	if len(words) == 0 {
		t.Fatal("应提取到词语")
	}
	// 应该包含单字和 bigram
	hasUnigram := false
	hasBigram := false
	for _, w := range words {
		runes := []rune(w)
		if len(runes) == 1 {
			hasUnigram = true
		}
		if len(runes) == 2 {
			hasBigram = true
		}
	}
	if !hasUnigram {
		t.Errorf("应包含单字: %v", words)
	}
	if !hasBigram {
		t.Errorf("应包含 bigram: %v", words)
	}
}

func TestSegmentChinese_Punctuation(t *testing.T) {
	words := segmentChinese("你好，请问：奖学金如何申请？")
	// 标点应被移除，不应包含标点字符
	for _, w := range words {
		for _, r := range w {
			if r == '，' || r == '：' || r == '？' {
				t.Errorf("不应包含标点: %q in %v", r, words)
			}
		}
	}
}

func TestSegmentChinese_English(t *testing.T) {
	words := segmentChinese("申请CET4考试")
	hasBigram := false
	hasUnigram := false
	for _, w := range words {
		runes := []rune(w)
		if len(runes) == 2 && isASCII(w) {
			hasBigram = true
		}
		if len(runes) == 1 && isASCII(w) {
			hasUnigram = true
		}
	}
	if !hasBigram {
		t.Errorf("应包含英文 bigram: %v", words)
	}
	if !hasUnigram {
		t.Errorf("应包含英文 unigram: %v", words)
	}
}

func isASCII(s string) bool {
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			continue
		}
		return false
	}
	return len(s) > 0
}

func TestIsStopWord(t *testing.T) {
	stopWords := []string{"请问", "你好", "谢谢", "我想", "知道", "什么", "怎么", "如何", "哪里", "为什么"}
	for _, w := range stopWords {
		if !isStopWord(w) {
			t.Errorf("%q 应为停用词", w)
		}
	}
}

func TestIsStopWord_ContentWords(t *testing.T) {
	contentWords := []string{"奖学金", "入学", "离校", "申请", "宿舍", "图书"}
	for _, w := range contentWords {
		if isStopWord(w) {
			t.Errorf("%q 不应为停用词", w)
		}
	}
}

// seedKBForRec 向知识库插入指定数量的已发布测试资源
func seedKBForRec(t *testing.T, kbRepo *repository.KBRepo, count int) {
	t.Helper()
	for i := 0; i < count; i++ {
		kbRepo.Create(&model.KBResource{
			ResourceID:   "rec-test-" + string(rune('0'+i%10)) + string(rune('0'+i/10)),
			ResourceType: "Policy",
			OwnerScope:   "school",
			OwnerID:      "school-1",
			RoleScope:    "student",
			Version:      "1.0",
			Status:       "published",
			Title:        "测试政策文档 " + string(rune('0'+i%10)),
			Summary:      "包含奖学金和入学信息",
			Content:      "详细政策内容",
			UpdatedBy:    "test",
		})
	}
}

// TestRecommendationService_GetRecommendations_NoHistory 无历史提问时应走冷启动
func TestRecommendationService_GetRecommendations_NoHistory(t *testing.T) {
	db := testutil.NewTestDBFull(t)
	defer db.Close()

	kbRepo := repository.NewKBRepo(db)
	messageRepo := repository.NewMessageRepo(db)
	svc := NewRecommendationService(kbRepo, messageRepo)

	seedKBForRec(t, kbRepo, 5)

	userCtx := &model.UserContext{
		UserID: 1, Username: "test", Role: "student",
		OwnerScope: "school", OwnerID: "school-1", DisplayName: "测试",
	}

	result, err := svc.GetRecommendations(userCtx, 10)
	if err != nil {
		t.Fatalf("GetRecommendations 失败: %v", err)
	}
	if result.Total == 0 {
		t.Error("至少应有冷启动热门内容")
	}
}

// TestRecommendationService_GetRecommendations_LimitValidation 边界校验
func TestRecommendationService_GetRecommendations_LimitValidation(t *testing.T) {
	db := testutil.NewTestDBFull(t)
	defer db.Close()

	kbRepo := repository.NewKBRepo(db)
	messageRepo := repository.NewMessageRepo(db)
	svc := NewRecommendationService(kbRepo, messageRepo)

	seedKBForRec(t, kbRepo, 20)

	userCtx := &model.UserContext{
		UserID: 1, Username: "test", Role: "student",
		OwnerScope: "school", OwnerID: "school-1", DisplayName: "测试",
	}

	result, err := svc.GetRecommendations(userCtx, 0)
	if err != nil {
		t.Fatalf("GetRecommendations(limit=0) 失败: %v", err)
	}
	if result.Total > 10 {
		t.Errorf("limit=0 时不应超过 10 条，得到 %d", result.Total)
	}
}

// TestRecommendationService_GetRecommendations_NoDuplicates 验证去重
func TestRecommendationService_GetRecommendations_NoDuplicates(t *testing.T) {
	db := testutil.NewTestDBFull(t)
	defer db.Close()

	kbRepo := repository.NewKBRepo(db)
	messageRepo := repository.NewMessageRepo(db)
	svc := NewRecommendationService(kbRepo, messageRepo)

	seedKBForRec(t, kbRepo, 3)

	userCtx := &model.UserContext{
		UserID: 1, Username: "test", Role: "student",
		OwnerScope: "school", OwnerID: "school-1", DisplayName: "测试",
	}

	result, err := svc.GetRecommendations(userCtx, 10)
	if err != nil {
		t.Fatalf("GetRecommendations 失败: %v", err)
	}

	seen := make(map[string]bool)
	for _, item := range result.Items {
		if seen[item.ResourceID] {
			t.Errorf("重复资源: %s", item.ResourceID)
		}
		seen[item.ResourceID] = true
	}
}
