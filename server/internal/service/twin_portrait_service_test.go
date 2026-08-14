package service

import (
	"strings"
	"testing"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/testutil"
)

// TestBuildPortraitPrompt 校验画像提示词包含 Q 版可爱精灵风与蔚小芯特色
func TestBuildPortraitPrompt(t *testing.T) {
	p := BuildPortraitPrompt(PrototypePhoto, PortraitPersonalization{
		DisplayName: "张三",
		Major:       "计算机科学与技术",
		Role:        "student",
		Highlights:  "学业优秀、社交活跃",
	})

	// 精简3段式提示词核心关键词
	required := []string{
		"Q版",   // 风格标识
		"大头小身", // 比例
		"超星",   // 参考风格
		"3D",   // 渲染质感
		"可爱",   // 风格基调
		"大眼睛",  // 核心五官
		"腮红",   // 萌点
		"学士帽",  // 校园元素
		"微光粒子", // 精灵感
	}
	for _, kw := range required {
		if !strings.Contains(p, kw) {
			t.Errorf("提示词缺少关键词: %s\n---\n%s", kw, p)
		}
	}
}

// TestTwinPortraitRepo_Upsert 校验画像 upsert 幂等
func TestTwinPortraitRepo_Upsert(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()
	r := repository.NewTwinPortraitRepo(db)

	p1 := &model.TwinPortrait{
		UserID: 1, PrototypeType: "photo", PromptVersion: "1.0",
		ImageBase64: "AAA", ImageMIME: "image/png",
	}
	if _, err := r.Upsert(p1); err != nil {
		t.Fatalf("upsert 失败: %v", err)
	}
	// 同类型覆盖
	p2 := &model.TwinPortrait{
		UserID: 1, PrototypeType: "photo", PromptVersion: "1.0",
		ImageBase64: "BBB", ImageMIME: "image/png",
	}
	if _, err := r.Upsert(p2); err != nil {
		t.Fatalf("upsert2 失败: %v", err)
	}
	got, _ := r.GetByUserAndType(1, "photo")
	if got == nil || got.ImageBase64 != "BBB" {
		t.Fatalf("upsert 应覆盖旧图: %+v", got)
	}
	// 不同类型并存
	if _, err := r.Upsert(&model.TwinPortrait{
		UserID: 1, PrototypeType: "chao_xing", PromptVersion: "1.0",
		ImageBase64: "CCC", ImageMIME: "image/png",
	}); err != nil {
		t.Fatalf("upsert3 失败: %v", err)
	}
	list, _ := r.ListByUser(1)
	if len(list) != 2 {
		t.Fatalf("应存 2 条画像，实际 %d", len(list))
	}
}
