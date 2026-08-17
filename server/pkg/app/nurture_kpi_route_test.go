package app

// D5-1 专项验证（qa-regression-wxx, 2026-08-16）
//
// 纯新增验证性测试：仅验证 D5-1 新增 `/college/nurture-kpi` 路由的注册与能力门控，
// 不修改任何既有生产代码，不改动既有测试断言语义。

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/dll/wxx/server/internal/auth"
)

// TestNurtureKPIRouteRegistered 校验 `/college/nurture-kpi` 已注册，且挂在
// secured /college 组下、由 auth.OutcomeDashboard 能力门控（与 education-outcome / party-dashboard 对齐）。
func TestNurtureKPIRouteRegistered(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("routes.go"))
	if err != nil {
		t.Fatalf("读取 routes.go 失败: %v", err)
	}
	s := string(src)

	if !strings.Contains(s, `collegeGroup.GET("/nurture-kpi"`) {
		t.Errorf("routes.go 未注册 collegeGroup.GET(\"/nurture-kpi\")")
	}
	if !strings.Contains(s, `auth.RequireCapability(auth.OutcomeDashboard), d.secretaryH.NurtureKPI`) {
		t.Errorf("nurture-kpi 未按 outcome.dashboard 能力门控（应与 education-outcome/party-dashboard 对齐）")
	}
	// 不新增独立能力：能力值应沿用既有 outcome.dashboard
	if auth.OutcomeDashboard != "outcome.dashboard" {
		t.Errorf("OutcomeDashboard 能力值应为 outcome.dashboard，得到 %q", auth.OutcomeDashboard)
	}
}

// TestNurtureKPIUnchangedSemantics 回归再确认：GetNurtureKPI 的 not_available 卡片 value 恒为 nil、不伪造数字。
// 通过静态检查确认 helper 的语义恒定（若有人改 helper 改为写死数字/估算，这里会先暴露）。
func TestNurtureKPIHelperSemantics(t *testing.T) {
	// 锚定本文件所在目录(pkg/app)定位 repo 源码，避免依赖 go test 的 CWD。
	// pkg/app -> pkg -> server -> server/internal/repository/secretary_outcome_repo.go
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("无法获取当前测试文件路径")
	}
	repoPath := filepath.Join(filepath.Dir(thisFile), "..", "..", "internal", "repository", "secretary_outcome_repo.go")
	src, err := os.ReadFile(repoPath)
	if err != nil {
		t.Fatalf("读取 repo 源码失败(路径=%s): %v", repoPath, err)
	}
	s := string(src)
	// notAvailableCard 恒返回 value nil，绝不返回伪数字
	if !strings.Contains(s, `"value": nil`) {
		t.Error("notAvailableCard 未将 value 固定为 nil（存在伪造数字风险）")
	}
	if !strings.Contains(s, `"data_source": "not_available"`) {
		t.Error("未找到 not_available 数据源标记")
	}
	if !strings.Contains(s, `"upload_target": "kb"`) {
		t.Error("not_available 卡片未配置知识库上传入口 upload_target=kb")
	}
}
