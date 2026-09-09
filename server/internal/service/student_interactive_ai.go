package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/dll/wxx/server/internal/llm"
)

var studentAssistDomains = map[string]string{
	"general":     "校园学习与生活",
	"study":       "学习规划、课程复习与笔记整理",
	"career":      "职业探索、简历与面试准备",
	"mental":      "非诊断性的心理支持与自我照护",
	"competition": "竞赛准备与项目训练",
	"campus":      "校园生活与办事准备",
	"process":     "办事流程材料梳理",
	"profile":     "个人成长档案与目标复盘",
	"vopc":        "vOPC 项目构思、表单草稿与验证计划",
}

// GenerateInteractiveAssist provides one AI entry point shared by every
// authenticated student-facing page. It never writes business data; callers
// must explicitly review and submit generated drafts through the normal API.
func (s *StudentService) GenerateInteractiveAssist(ctx context.Context, feature string, userID int64, input string) map[string]interface{} {
	domain, ok := studentAssistDomains[feature]
	if !ok {
		feature, domain = "general", studentAssistDomains["general"]
	}
	name := "同学"
	if s != nil && s.userRepo != nil {
		if user, err := s.userRepo.GetByID(userID); err == nil && user != nil && strings.TrimSpace(user.DisplayName) != "" {
			name = user.DisplayName
		}
	}
	prompt := fmt.Sprintf(`你是蔚小芯学生 AI 助手，当前功能域是「%s」。请针对%s的输入给出可执行帮助：%s
要求：先给一句结论，再给3至5条具体建议，最后列出下一步；不得编造学校政策、日期、名额、联系方式、个人成绩或活动状态；信息不足时明确说明需补充什么；涉及心理健康时不得诊断，危机情况应建议联系专业人员。`, domain, name, input)
	if s != nil && s.llmClient != nil {
		resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{
			Messages:    []llm.ChatMessage{{Role: "user", Content: prompt}},
			Temperature: 0.4,
			MaxTokens:   700,
		})
		if err == nil && resp != nil && strings.TrimSpace(resp.Content) != "" {
			return map[string]interface{}{
				"feature": feature, "title": domain + "助手", "response": strings.TrimSpace(resp.Content),
				"data_source": "ai", "review_required": true,
				"sources": []interface{}{}, "next_action": "核对内容后再应用到当前功能",
			}
		}
	}
	return map[string]interface{}{
		"feature": feature, "title": domain + "助手", "response": fallbackInteractiveAssist(feature, input),
		"data_source": "rule", "review_required": true,
		"sources": []interface{}{}, "next_action": "补充目标、期限和已有材料后可获得更具体的建议",
	}
}

func fallbackInteractiveAssist(feature, input string) string {
	actions := map[string]string{
		"study":       "先明确本周要掌握的知识点，再把任务拆为阅读、练习、复盘三类，并为每项设置可检查的完成标准。",
		"career":      "先明确目标岗位和时间范围，再整理已有技能、项目证据与差距，优先补齐一个能在简历中验证的成果。",
		"mental":      "可以先记录当前感受、持续时间和主要压力源，再选择休息、运动或联系可信任的人。这里的建议不能替代专业评估。",
		"competition": "先核对竞赛官方通知与报名条件，再按基础知识、作品原型、模拟答辩拆解准备任务，不使用未经核实的日期或名额。",
		"campus":      "先确认事项名称、办理对象和截止时间，再从学校正式通知核对入口、材料、地点和联系人。",
		"process":     "把当前事项拆为资格确认、材料准备、线上或线下提交、结果查询四步；所有政策字段以正式来源为准。",
		"profile":     "从目标、已完成证据、当前差距和下阶段行动四部分复盘，避免用没有数据支撑的评分替代真实记录。",
		"vopc":        "把想法写成“为谁解决什么问题”，再补最小成果、4周计划、5名用户验证和可检查的验收标准。",
		"general":     "先写清目标、期限和已有材料，再把任务拆成可以立即执行、可以检查结果的下一步。",
	}
	base := actions[feature]
	if base == "" {
		base = actions["general"]
	}
	return "基于你输入的“" + strings.TrimSpace(input) + "”，建议：\n1. " + base + "\n2. 标记仍需从正式来源核实的信息。\n3. 完成后记录结果，再让 AI 帮你复盘和调整。"
}
