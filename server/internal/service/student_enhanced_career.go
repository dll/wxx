package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/dll/wxx/server/internal/llm"
)

func (s *StudentService) GenerateEnhancedCareerSimulation(ctx context.Context, careerPath string) *EnhancedCareerSimulation {
	if careerPath == "" {
		careerPath = "软件开发工程师"
	}
	data := &EnhancedCareerSimulation{CareerPath: careerPath, CurrentStage: "在校学生（大二）", Stages: []map[string]interface{}{{"stage": "在校期(当前)", "duration": "大二-大四", "actions": []string{"夯实数据结构与算法", "参与开源项目", "获得至少1次实习经历"}, "success_rate": 0.85}, {"stage": "应届生", "duration": "毕业1年", "title": "初级开发工程师", "salary": "8-15万", "actions": []string{"快速熟悉公司技术栈", "建立技术博客", "考取相关认证"}, "success_rate": 0.78}, {"stage": "3年经验", "duration": "工作3-5年", "title": "高级开发工程师", "salary": "20-35万", "actions": []string{"深入某一技术领域", "带新人/做技术分享", "参与架构设计"}, "success_rate": 0.65}, {"stage": "5年+", "duration": "工作5-10年", "title": "技术专家/架构师", "salary": "40-80万", "actions": []string{"技术决策与架构规划", "跨团队协作", "行业影响力建设"}, "success_rate": 0.35}}, SkillsGap: []map[string]interface{}{{"skill": "分布式系统设计", "current": 30, "target": 75, "importance": "high"}, {"skill": "系统架构能力", "current": 20, "target": 70, "importance": "high"}, {"skill": "技术管理", "current": 10, "target": 60, "importance": "medium"}}, SalaryProjection: []map[string]interface{}{{"year": "2026(应届)", "range": "8-15万", "percentile_50": 10, "percentile_90": 18}, {"year": "2029(3年)", "range": "20-35万", "percentile_50": 25, "percentile_90": 40}, {"year": "2031(5年)", "range": "35-60万", "percentile_50": 42, "percentile_90": 70}}, MarketTrends: []string{"AI/大模型相关岗位需求年增长45%", "全栈工程师薪资溢价约20%", "远程办公成为常态，国际化机会增多"}, AlternativePathways: []string{"技术管理路径: 开发→Tech Lead→技术经理→技术总监", "创业路径: 积累3-5年经验→加入初创公司→自主创业", "自由职业路径: 建立技术品牌→接外包→远程工作→数字游民"}, DataSource: "reference"}
	if s.llmClient != nil {
		if resp, err := s.llmClient.Chat(ctx, &llm.ChatRequest{Messages: []llm.ChatMessage{{Role: "user", Content: fmt.Sprintf("你是职业规划师。请为一名大二计算机专业学生规划「%s」职业路径，包括各阶段关键行动和技能差距。约100字。", careerPath)}}, Temperature: 0.5, MaxTokens: 400}); err == nil && resp != nil && resp.Content != "" {
			data.AIAdvice = strings.TrimSpace(resp.Content)
			data.DataSource = "reference+ai"
		}
	}
	return data
}
