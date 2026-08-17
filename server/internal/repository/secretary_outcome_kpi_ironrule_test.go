package repository

// D5-1 专项补充验证（qa-regression-wxx, 2026-08-16）
//
// 纯新增验证性测试：对 GetNurtureKPI 的「铁律」做显式枚举——
// 每类 KPI 必须是 real（source_desc 指向真实表）或 not_available（value 恒 nil + 上传入口）。
// 若出现任何 real 但 value 为估算/写死/编造的伪装值，本测试即失败（提前暴露造假）。

import (
	"sort"
	"testing"
	"time"

	"github.com/dll/wxx/server/internal/testutil"
)

// sortedKeys 返回 map 的 key 排序列表（仅用于日志展示）
func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// TestNurtureKPIIronRuleEnumeration 空库（无任何真实记录）下枚举全量 KPI 卡：
// 统计每张卡的 data_source，且任何 not_available 卡的 value 必须是 nil、upload_target 必须是 kb。
// 这保证后端绝不就「岗位应有但无数据源」的指标伪造数值，前端据此渲染上传入口而非假数字。
// P1-2（2026-08-17）：登记合法 data_source 枚举为 real / not_available / trend，并强制 trend 卡
// value 非空 + sample_count>0（缺纵向样本时必须回落 not_available，绝不硬画空趋势）。
func TestNurtureKPIIronRuleEnumeration(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()
	createNurtureKPITables(t, db)

	// 清空所有学生，确保全校范围内学生基数为 0，逼出最朴素的空库场景
	_, _ = db.Exec(`DELETE FROM users WHERE role='student'`)

	repo := NewSecretaryOutcomeRepo(db)
	kpis := repo.GetNurtureKPI("")

	if len(kpis) == 0 {
		t.Fatalf("GetNurtureKPI 空库应仍返回指标卡清单（real 卡片 value=0 或 not_available 卡片），而非空")
	}

	// 空库下不应出现 trend 卡：无纵向历史时 growth_trend 必须回落 not_available
	if c := findKPI(kpis, "nurture.growth_trend"); c == nil {
		t.Fatalf("nurture.growth_trend 卡必须存在")
	} else if c["data_source"] != "not_available" {
		t.Errorf("空库下 nurture.growth_trend data_source 应为 not_available，得到 %v（铁律：无纵向历史绝不返回 trend 或伪趋势）", c["data_source"])
	} else if c["source_desc"] == nil || c["source_desc"] == "" {
		t.Errorf("空库下 nurture.growth_trend 应为诚实「数据积累中/需满 N 周」文案，source_desc 不应为空")
	}

	realKeys := map[string]bool{}
	naKeys := map[string]bool{}
	trendKeys := map[string]bool{}
	for _, k := range kpis {
		key, _ := k["key"].(string)
		switch k["data_source"] {
		case "real":
			realKeys[key] = true
		case "not_available":
			naKeys[key] = true
			// 铁律：not_available → value 必须为 nil（绝不伪造数字）
			if k["value"] != nil {
				t.Errorf("KPI %s data_source=not_available 但 value=%v（违反铁律：不造数）", key, k["value"])
			}
			if k["upload_target"] != "kb" {
				t.Errorf("KPI %s data_source=not_available 缺知识库上传入口 upload_target=kb", key)
			}
		case "trend":
			trendKeys[key] = true
			// 铁律：trend 必属实趋势 → value 非空、无上传入口、sample_count>0
			if k["value"] == nil {
				t.Errorf("KPI %s data_source=trend 但 value 为空（铁律：trend 不空壳，缺样本应回落 not_available）", key)
			}
			if k["upload_target"] != nil {
				t.Errorf("KPI %s data_source=trend 不应带上传补料入口 upload_target=%v", key, k["upload_target"])
			}
			if sc, ok := k["sample_count"].(int); !ok || sc <= 0 {
				t.Errorf("KPI %s data_source=trend 必须标注 sample_count>0，得到 %v（诚实：样本诚实）", key, k["sample_count"])
			}
		default:
			t.Errorf("KPI %s 存在未知 data_source=%v（只能 real / not_available / trend）", key, k["data_source"])
		}
	}

	// 明确要求：职责应有但无数据源的专属卡必须存在且恒为 not_available
	for _, key := range []string{
		"nurture.intervention_total",     // 干预执行（AI 生成文本、未落统计表）
		"nurture.second_class_pass_rate", // 二课达标率（无达标阈值配置）
		"nurture.growth_trend",           // 成长度归因（纵向留痕积累中→not_available 占位）
		"nurture.employment_target",      // 就业目标达成（无目标值配置）
	} {
		if !naKeys[key] {
			t.Errorf("职责应有但无数据源的指标 %s 应为 not_available 卡，未出现", key)
		}
	}

	t.Logf("real 卡(%d): %v", len(realKeys), sortedKeys(realKeys))
	t.Logf("not_available 卡(%d): %v", len(naKeys), sortedKeys(naKeys))
	t.Logf("trend 卡(%d): %v", len(trendKeys), sortedKeys(trendKeys))
}

// TestNurtureKPI_GrowthTrend_TrendAvailable 注入 ≥N 周纵向双端历史后，growth_trend 应返回
// data_source=trend，且五维平均变化正确、sample_count 正确、无上传补料入口；随后清空历史应回落
// not_available（铁律：不造数，无样本绝不硬画趋势）。
func TestNurtureKPI_GrowthTrend_TrendAvailable(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()
	createNurtureKPITables(t, db)

	_, _ = db.Exec(`DELETE FROM users WHERE role='student'`)
	_, err := db.Exec(`INSERT INTO users (username, role, display_name, owner_scope, owner_id, college)
		VALUES ('cs_t1','student','趋势甲','college','cs','cs'),
		       ('cs_t2','student','趋势乙','college','cs','cs')`)
	if err != nil {
		t.Fatalf("插入学生失败: %v", err)
	}
	var uidA, uidB int64
	_ = db.QueryRow(`SELECT id FROM users WHERE username='cs_t1'`).Scan(&uidA)
	_ = db.QueryRow(`SELECT id FROM users WHERE username='cs_t2'`).Scan(&uidB)

	repo := NewSecretaryOutcomeRepo(db)

	// 无历史 → 回落 not_available（value=nil，诚实积累中文案）
	na := findKPI(repo.GetNurtureKPI("cs"), "nurture.growth_trend")
	if na["data_source"] != "not_available" || na["value"] != nil {
		t.Fatalf("无历史时 growth_trend 应为 not_available 且 value=nil，得到 %v / %v", na["data_source"], na["value"])
	}

	// 注入两端历史（窗口 4 周内：3 周前 vs 今天），computed_at 归一化日值
	threeWeeksAgoDay := time.Now().AddDate(0, 0, -21).Format("2006-01-02") + " 00:00:00"
	nowDay := time.Now().Format("2006-01-02") + " 00:00:00"
	insertHist := func(uid int64, day string, base, bonus float64) {
		_, e := db.Exec(`INSERT INTO snapshot_history
			(user_id, owner_scope, owner_id, college, major, class_name,
			 academic_score, ability_score, ideological_score, emotional_score, social_score, computed_at)
			VALUES (?, 'college', 'cs', 'cs', '', '', ?, ?, ?, ?, ?, ?)`,
			uid, base+bonus, base+bonus, base+bonus, base+bonus, base+bonus, day)
		if e != nil {
			t.Fatalf("插入快照历史失败: %v", e)
		}
	}
	// A: 早期 50，近期 55（各维 +5）；B: 早期 40，近期 38（各维 -2）
	insertHist(uidA, threeWeeksAgoDay, 50, 0)
	insertHist(uidA, nowDay, 50, 5)
	insertHist(uidB, threeWeeksAgoDay, 40, 0)
	insertHist(uidB, nowDay, 40, -2)

	trend := findKPI(repo.GetNurtureKPI("cs"), "nurture.growth_trend")
	if trend["data_source"] != "trend" {
		t.Fatalf("有纵向双端历史时 growth_trend 应为 trend，得到 %v", trend["data_source"])
	}
	if sc, _ := trend["sample_count"].(int); sc != 2 {
		t.Fatalf("sample_count 应为 2，得到 %v", trend["sample_count"])
	}
	if trend["upload_target"] != nil {
		t.Fatalf("trend 卡不应带上传补料入口 upload_target=%v", trend["upload_target"])
	}
	val, ok := trend["value"].(map[string]interface{})
	if !ok {
		t.Fatalf("trend value 应为五维 map，得到 %v", trend["value"])
	}
	for _, dim := range []string{"academic", "ability", "ideological", "emotional", "social"} {
		if d := numOf(val, dim); d != 1.5 {
			t.Fatalf("维度 %s 平均变化应为 1.5，得到 %v", dim, d)
		}
	}

	// 清空历史 → 回落 not_available（诚实，不造数）
	_, _ = db.Exec(`DELETE FROM snapshot_history`)
	na2 := findKPI(repo.GetNurtureKPI("cs"), "nurture.growth_trend")
	if na2["data_source"] != "not_available" {
		t.Fatalf("清空历史后 growth_trend 应回落 not_available，得到 %v", na2["data_source"])
	}
}

// TestGetGrowthTrend_CrossOwnerIsolation 越权红线：getGrowthTrend 按 owner_id 收窄后只读本院历史，绝不跨院。
func TestGetGrowthTrend_CrossOwnerIsolation(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()
	createNurtureKPITables(t, db)

	_, _ = db.Exec(`DELETE FROM users WHERE role='student'`)
	_, err := db.Exec(`INSERT INTO users (username, role, display_name, owner_scope, owner_id, college)
		VALUES ('iso_a1','student','本院生','college','cs','cs'),
		       ('iso_b1','student','外院生','college','zd','zd')`)
	if err != nil {
		t.Fatalf("插入学生失败: %v", err)
	}
	var uidA, uidB int64
	_ = db.QueryRow(`SELECT id FROM users WHERE username='iso_a1'`).Scan(&uidA)
	_ = db.QueryRow(`SELECT id FROM users WHERE username='iso_b1'`).Scan(&uidB)

	threeWeeksAgoDay := time.Now().AddDate(0, 0, -21).Format("2006-01-02") + " 00:00:00"
	nowDay := time.Now().Format("2006-01-02") + " 00:00:00"
	insertHist := func(uid int64, owner, day string, base, bonus float64) {
		_, e := db.Exec(`INSERT INTO snapshot_history
			(user_id, owner_scope, owner_id, college, major, class_name,
			 academic_score, ability_score, ideological_score, emotional_score, social_score, computed_at)
			VALUES (?, 'college', ?, ?, '', '', ?, ?, ?, ?, ?, ?)`,
			uid, owner, owner, base+bonus, base+bonus, base+bonus, base+bonus, base+bonus, day)
		if e != nil {
			t.Fatalf("插入快照历史失败: %v", e)
		}
	}
	// 本院一生双端；外院生仅有单端（且属不同 owner，应被 owner 过滤）
	insertHist(uidA, "cs", threeWeeksAgoDay, 50, 0)
	insertHist(uidA, "cs", nowDay, 50, 5)
	insertHist(uidB, "zd", nowDay, 40, 0)

	repo := NewSecretaryOutcomeRepo(db)
	gt, err := repo.getGrowthTrend("cs", 4)
	if err != nil {
		t.Fatalf("查询趋势失败: %v", err)
	}
	// 只统计本院 1 名双端学生，外院不入计
	if gt.SampleCount != 1 || !gt.HasData {
		t.Fatalf("越权隔离：仅应统计本院 1 名双端学生，得到 sample_count=%d hasData=%v", gt.SampleCount, gt.HasData)
	}
	if gt.Academic != 5 {
		t.Fatalf("本院学业平均变化应为 5，得到 %v", gt.Academic)
	}
}
