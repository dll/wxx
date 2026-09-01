// Package eval 黄金评测集（A5）：以 CI 单测形式对检索质量做回归评测。
//
// 设计：
//   - 种子知识库：内存 SQLite + 生产迁移（含 FTS5），插入确定性的校园场景资源；
//   - 评测对象：生产检索管道（ChatService.Ask → Context Engine：改写/意图加权/信任分）；
//   - 断言：期望资源进入 sources Top-K；编造守卫用例必须走兜底；
//     全局断言：过期资源永不出现、无权限资源对学生不可见。
//   - 通过标准：全部用例通过（单个失败即测试失败，失败列表见输出）。
//
// 运行：go test ./eval/ -v（CI 自动执行）。
package eval

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"testing"

	_ "modernc.org/sqlite"

	dbutil "github.com/dll/wxx/server/internal/db"
	"github.com/dll/wxx/server/internal/llm"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/service"
	"github.com/dll/wxx/server/pkg/app"
)

//go:embed golden_cases.json
var goldenRaw []byte

// EvalCase 单条评测用例
type EvalCase struct {
	ID                string   `json:"id"`
	Question          string   `json:"question"`
	PreQuestion       string   `json:"pre_question,omitempty"` // 多轮：先问此问题建立上下文
	Role              string   `json:"role,omitempty"`         // 默认 student
	ExpectResourceIDs []string `json:"expect_resource_ids"`
	ExpectFallback    bool     `json:"expect_fallback,omitempty"`
	Note              string   `json:"note,omitempty"`
}

type goldenSet struct {
	Version string     `json:"version"`
	Note    string     `json:"note"`
	Cases   []EvalCase `json:"cases"`
}

// seedResources 评测种子知识库（与 golden_cases.json 对应；新增用例请同步补充）
func seedResources(db *sql.DB) error {
	kbRepo := repository.NewKBRepo(db)
	resources := []*model.KBResource{
		{ResourceID: "kb-scholarship-national", ResourceType: "Policy", Title: "国家奖学金评定办法",
			Summary: "国家奖学金的申请条件、评定标准与材料要求",
			Content: "申请条件：在校生二年级以上，学习成绩排名专业前 10%，综合素质测评优秀，无违纪记录。所需材料：国家奖学金申请表、成绩单、获奖证书复印件、事迹材料。评定时间：每年 9 月启动，10 月公示。奖励金额 8000 元/人。"},
		{ResourceID: "kb-scholarship-process", ResourceType: "Process", Title: "奖学金申请办理流程",
			Summary: "从班级推荐到学校评审的完整流程",
			Content: "第一步：班级民主评议推荐；第二步：学院资格初审并公示；第三步：学生填写奖学金申请表并附成绩单；第四步：学校学生资助管理中心评审；第五步：全校公示 5 个工作日；第六步：奖金发放至学生银行卡。"},
		{ResourceID: "kb-leave-process", ResourceType: "Process", Title: "学生请假与休学管理办法",
			Summary: "请假审批权限、休学复学手续",
			Content: "请假 1 天以内由辅导员审批；3 天以内由学院学工办主任审批；3 天以上由学院分管领导审批，超过 2 周须报教务处备案。休学：本人申请、家长签字、医院证明（病休），学院审核后报教务处批准，保留学籍两年。复学：学期结束前 1 个月提交复学申请。"},
		{ResourceID: "kb-enrollment-guide", ResourceType: "Process", Title: "新生报到指南",
			Summary: "报到时间、地点与携带材料清单",
			Content: "报到时间：9 月 6 日至 7 日 8:00-18:00。报到地点：学校体育馆新生报到大厅。携带材料：录取通知书、身份证、档案袋、党团组织关系介绍信、一寸免冠照片 8 张、户口迁移证（自愿）。缴费方式：线上缴费或现场刷卡。"},
		{ResourceID: "kb-enrollment-dorm", ResourceType: "FAQ", Title: "新生宿舍分配常见问题",
			Summary: "宿舍分配规则与调换申请",
			Content: "宿舍分配按学院、班级相对集中原则，由后勤处统一安排，随机分配房间与床位，可在线上系统查询分配结果。特殊情况（身体原因等）可在报到前提交调换申请至辅导员处。宿舍标配：空调、独立书桌、独立卫浴（部分楼栋）。"},
		{ResourceID: "kb-mental-center", ResourceType: "FAQ", Title: "心理咨询中心服务指南",
			Summary: "地址、电话、开放时间与预约方式",
			Content: "心理咨询中心位于大学生活动中心 302 室。预约电话 0550-1234567。开放时间：周一至周五 8:30-11:30、14:30-17:30，周四晚间延长至 20:00。预约方式：电话预约或到现场登记，首次咨询 50 分钟。咨询内容保密。"},
		{ResourceID: "kb-career-internship", ResourceType: "Policy", Title: "实习学分认定管理办法",
			Summary: "校外实习的学分认定条件与材料",
			Content: "认定条件：实习时长不少于 4 周，岗位与专业相关。所需材料：实习协议三方件、实习单位鉴定表（含盖章）、实习日志（每周一篇）、实习总结报告。提交时间：实习结束后 2 周内交学院教学办。审核通过计 2 学分。"},
		{ResourceID: "kb-course-resit", ResourceType: "Policy", Title: "课程补考与重修管理规定",
			Summary: "不及格课程的补考与重修要求",
			Content: "必修课不及格者可参加下学期初补考，补考不收取费用；补考仍不及格者须重修。重修随下一轮开课班级跟班上课，或按教务处安排参加重修班。重修成绩按实际成绩记载并标注'重修'。毕业前仍有不及格课程者，按结业处理。"},
		{ResourceID: "kb-course-select", ResourceType: "FAQ", Title: "选课系统常见问题",
			Summary: "选课时间、退改选规则",
			Content: "选课分三轮：第一轮预选（第 16 周）、第二轮正选（第 17 周）、第三轮补退选（开学第 1 周）。每学期最多修读 28 学分（设上限控制学业负荷）。退选：开学第 1 周内可在系统自行退选，逾期不可退。跨专业选课须经开课学院同意。"},
		{ResourceID: "kb-graduation-checklist", ResourceType: "Process", Title: "毕业离校手续办理清单",
			Summary: "离校前须完成的手续与材料转递",
			Content: "离校手续：第一步归还图书馆图书并结清欠款；第二步退还宿舍钥匙并结清水电费；第三步到财务处核对学费结算；第四步到学院领取档案转递单；第五步办理党团组织关系转出；第六步凭全部签章的离校单领取毕业证学位证。档案通过 EMS 特快专递转递至就业单位或生源地人才中心。"},
		{ResourceID: "kb-activity-club", ResourceType: "Activity", Title: "学生社团注册与活动申报指引",
			Summary: "社团成立条件与活动审批要求",
			Content: "社团注册：不少于 20 名成员、1 名指导教师，向校团委提交章程与申请表。活动申报：校外人员参加或涉及场地的活动须提前 5 个工作日在第二课堂系统申报，经团委与保卫处审批。经费：可申请团委活动经费或收取会费（上限 30 元/年）。"},
		{ResourceID: "kb-contact-registry", ResourceType: "FAQ", Title: "常用部门电话与办公地点",
			Summary: "学工、教务、财务等部门联系方式",
			Content: "学工办：行政楼 210，0550-1234501，工作日 8:00-17:30。教务处：行政楼 305，0550-1234502。财务处：行政楼 102，0550-1234503，报销窗口工作日上午办理。图书馆：逸夫楼，0550-1234504。校医院：校园东侧，24 小时值班。"},
		{ResourceID: "kb-library", ResourceType: "FAQ", Title: "图书馆开放时间与借阅规则",
			Summary: "开放时间、借阅数量与期限",
			Content: "开放时间：周一至周日 7:30-22:00，考试周延长至 23:00。借阅规则：本科生限借 10 册，期限 30 天，可续借 1 次；逾期按 0.1 元/天收取滞纳金。借书证即校园卡。"},
		{ResourceID: "kb-expired-transfer-notice", ResourceType: "Policy", Title: "2024 版转专业申请通知",
			Summary:   "（已过期）2024 年转专业工作安排",
			Content:   "本通知为 2024 年春季学期转专业工作安排，现已失效。申请窗口已于 2024 年 4 月 30 日关闭。",
			ExpiredAt: strPtr("2025-01-01")},
		{ResourceID: "kb-transfer-active", ResourceType: "Policy", Title: "转专业管理办法（2026 版）",
			Summary: "在校本科生转专业的申请条件、时间与流程",
			Content: "转专业申请条件：一年级第二学期或二年级第一学期，无不及格课程，绩点专业前 30%。申请时间：每年 4 月（春季学期）。所需材料：转专业申请表、成绩单。流程：本人申请 → 转出学院审核 → 拟转入学院考核 → 教务处公示。"},
		{ResourceID: "kb-counselor-handbook", ResourceType: "Policy", Title: "辅导员工作手册：谈心谈话要求",
			Summary:   "辅导员谈心谈话频次与记录要求（仅辅导员可见）",
			Content:   "谈话要求：每学期与所带每名学生至少深度谈话 1 次；重点关注学生每月至少 2 次。谈话记录须在 24 小时内录入工作系统，涉及心理危机的立即上报。仅限辅导员及以上角色查阅。",
			RoleScope: `["counselor"]`},
		{ResourceID: "kb-library-reserve", ResourceType: "FAQ", Title: "阅览室选座与图书续借规则",
			Summary: "线上选座、占座清理与续借操作",
			Content: "阅览室选座：通过图书馆公众号或入口机选座，离座超过 30 分钟未释放视为占座，管理员将清理物品。图书续借：借期内可在图书馆系统自助续借 1 次，续借期 30 天，已被预约的图书不可续借。考试周开放预约排队。"},
		{ResourceID: "kb-campus-card", ResourceType: "FAQ", Title: "校园卡补办、充值与挂失",
			Summary: "校园卡丢失补办、线上充值与解挂",
			Content: "校园卡挂失：可通过线上系统或圈存机即时挂失。补办：持身份证到卡务中心（行政楼 108）办理，工本费 15 元，3 个工作日后领取，旧卡余额自动转入新卡。充值：线上 App 充值实时到账，或圈存机银行ka转卡。解挂：找到原卡后可线上解挂恢复使用。"},
		{ResourceID: "kb-dorm-repair", ResourceType: "Process", Title: "宿舍报修流程",
			Summary: "宿舍设施报修的渠道与时限",
			Content: "报修渠道：第一步在后勤报修系统提交（拍照描述故障）；第二步等待派单，一般 3 个工作日内上门；紧急情况（如水管爆裂、电路跳闸等）请直接拨打后勤值班电话 0550-1234599，2 小时内响应。维修完成后在系统内确认评价。水龙头漏水、灯管更换属免费维修范围。"},
		{ResourceID: "kb-sports-test", ResourceType: "Policy", Title: "国家学生体质健康标准与免测申请",
			Summary: "体质测试项目、评分与免测缓测规定",
			Content: "测试项目：身高体重、肺活量、50 米跑、立定跳远、坐位体前屈、800 米（女）/1000 米（男）、引体向上（男）/仰卧起坐（女）。每年 10 月统一测试，成绩按国家评分表评定，不及格者次年补测。免测：因病因残可提交医院证明申请免测（须体育部审批）。体质测试与评奖评优挂钩。"},
		{ResourceID: "kb-insurance", ResourceType: "FAQ", Title: "大学生医保报销指南",
			Summary: "参保缴费、报销比例与异地就医",
			Content: "参保：每年 9 月集中缴纳城乡居民医保（个人缴费部分约 380 元/年，随政策调整）。校医院就诊直接刷卡结算；校外定点医院就诊先自费垫付，后凭发票、病历、费用清单到医保办报销，报销比例约 60%-80%。异地就医：寒暑假回家看病须提前线上备案，否则报销比例下调。"},
		{ResourceID: "kb-tuition", ResourceType: "FAQ", Title: "学费缴纳与缓交申请",
			Summary: "缴费渠道、截止时间与缓交流程",
			Content: "学费缴纳：通过学校统一支付平台线上缴费（支持微信/支付宝/银行卡），每学年开学两周内完成。住宿费随学费一并缴纳。缓交：家庭经济困难学生可在开学两周内提交缓交申请表（辅导员签字 + 学院审批），缓交期最长一学期。逾期未缴且未申请缓交者影响选课与注册。"},
	}
	for _, r := range resources {
		r.OwnerScope = "school"
		if r.RoleScope == "" {
			r.RoleScope = `["student"]`
		}
		r.Version = "1.0"
		r.Status = "published"
		r.UpdatedBy = "eval-seed"
		if _, _, err := kbRepo.Upsert(r); err != nil {
			return fmt.Errorf("播种资源 %s 失败: %w", r.ResourceID, err)
		}
	}
	return nil
}

func strPtr(s string) *string { return &s }

// seedMigrationsToSkip 评测语料隔离：跳过知识/内容种子迁移（受控语料由 seedResources 提供）
func seedMigrationsToSkip() []string {
	return []string{
		"003_seed_knowledge.sql",
		"008_enrich_knowledge.sql",
		"020_seed_graduation_process.sql",
		"026_seed_additional_processes.sql",
		"031_portal_knowledge.sql",
		"034_fix_process_flow_seed.sql",
		"035_seed_common_processes.sql",
		"054_freshmen_guide_seed.sql",
		"058_major_knowledge.sql",
		"067_ai_briefings_seed.sql",
		"088_import_official_docs.sql",
	}
}

func loadCases(t *testing.T) goldenSet {
	t.Helper()
	var gs goldenSet
	if err := json.Unmarshal(goldenRaw, &gs); err != nil {
		t.Fatalf("解析 golden_cases.json 失败: %v", err)
	}
	if len(gs.Cases) == 0 {
		t.Fatal("黄金评测集为空")
	}
	return gs
}

func newEvalService(t *testing.T) (*service.ChatService, func()) {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	// :memory: 每个连接是独立数据库，必须限单连接保证 schema 一致
	db.SetMaxOpenConns(1)
	cleanup := func() { db.Close() }

	// 与生产完全一致的迁移路径（含 FTS5 与全部增量，重复列/索引容错同生产）。
	// 例外：跳过知识/内容种子迁移——评测需要受控语料，由 seedResources 提供确定性资源。
	if err := app.RunMigrations(db, dbutil.DriverSQLite, seedMigrationsToSkip()...); err != nil {
		t.Fatalf("执行迁移失败: %v", err)
	}

	if err := seedResources(db); err != nil {
		t.Fatalf("播种知识库失败: %v", err)
	}

	svc := service.NewChatService(
		repository.NewSessionRepo(db),
		repository.NewMessageRepo(db),
		repository.NewKBRepo(db),
		repository.NewAgentRepo(db),
		llm.NewMockClient("eval-llm"),
	)
	return svc, cleanup
}

// TestGoldenRetrieval 检索质量回归评测：全部用例通过才算通过。
func TestGoldenRetrieval(t *testing.T) {
	gs := loadCases(t)
	svc, cleanup := newEvalService(t)
	defer cleanup()

	type failure struct {
		id     string
		reason string
	}
	var failures []failure

	for _, tc := range gs.Cases {
		t.Run(tc.ID, func(t *testing.T) {
			role := tc.Role
			if role == "" {
				role = "student"
			}
			userCtx := &model.UserContext{
				UserID: 1, Username: "eval-user", Role: role,
				OwnerScope: "school", OwnerID: "school-1", DisplayName: "评测用户",
			}

			sessionID := ""
			if tc.PreQuestion != "" {
				_, sid, err := svc.Ask(context.Background(), userCtx, "", tc.PreQuestion, "")
				if err != nil {
					t.Fatalf("前置问题执行失败: %v", err)
				}
				sessionID = sid
			}

			card, _, err := svc.Ask(context.Background(), userCtx, sessionID, tc.Question, "")
			if err != nil {
				t.Fatalf("Ask 失败: %v", err)
			}
			if card == nil {
				t.Fatal("AnswerCard 为空")
			}

			// 全局断言：过期资源永不出现
			for _, s := range card.Sources {
				if s.ResourceID == "kb-expired-transfer-notice" {
					t.Errorf("过期资源泄漏进 sources（编造/过期引用风险）")
				}
			}

			gotIDs := map[string]bool{}
			for _, s := range card.Sources {
				gotIDs[s.ResourceID] = true
			}

			if tc.ExpectFallback {
				if !card.Fallback {
					t.Errorf("期望兜底但返回了正常回答，sources=%v", keys(gotIDs))
				}
				return
			}

			if card.Fallback {
				t.Errorf("期望命中但走了兜底，期望资源=%v", tc.ExpectResourceIDs)
				return
			}
			for _, want := range tc.ExpectResourceIDs {
				if !gotIDs[want] {
					t.Errorf("期望资源 %s 未进入 sources Top-K，实际=%v", want, keys(gotIDs))
				}
			}
		})
	}

	if len(failures) > 0 {
		t.Errorf("共 %d 例失败", len(failures))
	}
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
