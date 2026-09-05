package service

import (
	"encoding/json"
	"strings"
)

// parseLessonPlanJSON converts an LLM response into a complete lesson plan.
// Missing optional fields retain deterministic fallback content.
func parseLessonPlanJSON(text, topic string) *LessonPlan {
	plan := &LessonPlan{
		Topic:        topic,
		Outline:      "教学目标：\n1. 理解" + topic + "的基本概念\n2. 掌握" + topic + "的核心算法\n3. 能够应用" + topic + "解决实际问题\n\n教学过程：\n- 导入（5分钟）：通过生活中的例子引入\n- 讲授（30分钟）：核心概念与算法讲解\n- 练习（15分钟）：课堂练习与点评\n- 总结（5分钟）：知识回顾与作业布置",
		KeyPoints:    []string{topic + "的基本概念与定义", topic + "的核心算法原理", topic + "的典型应用场景"},
		Difficulties: []string{topic + "的算法复杂度分析", topic + "的边界条件处理"},
		Strategies:   []string{"案例驱动教学：用真实案例激发学习兴趣", "分组讨论：鼓励学生自主探究", "可视化演示：用动画辅助理解抽象概念"},
		Interactions: []string{"随机提问：检测学生理解程度", "小组竞赛：提高课堂参与度", "即时答题：实时反馈学习效果"},
		Homework:     []string{"完成" + topic + "课后练习题 1-5", "实现" + topic + "相关算法代码", "撰写" + topic + "的学习笔记与思考"},
		DataSource:   "fallback",
	}
	if text == "" {
		return plan
	}
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "```") {
		idx, endIdx := strings.Index(text, "{"), strings.LastIndex(text, "}")
		if idx >= 0 && endIdx > idx {
			text = text[idx : endIdx+1]
		}
	}
	var parsed struct {
		Outline      string   `json:"outline"`
		KeyPoints    []string `json:"key_points"`
		Difficulties []string `json:"difficulties"`
		Strategies   []string `json:"strategies"`
		Interactions []string `json:"interactions"`
		Homework     []string `json:"homework"`
	}
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		return plan
	}
	if parsed.Outline != "" {
		plan.Outline = parsed.Outline
	}
	if len(parsed.KeyPoints) > 0 {
		plan.KeyPoints = parsed.KeyPoints
	}
	if len(parsed.Difficulties) > 0 {
		plan.Difficulties = parsed.Difficulties
	}
	if len(parsed.Strategies) > 0 {
		plan.Strategies = parsed.Strategies
	}
	if len(parsed.Interactions) > 0 {
		plan.Interactions = parsed.Interactions
	}
	if len(parsed.Homework) > 0 {
		plan.Homework = parsed.Homework
	}
	return plan
}

func (s *TeacherService) fallbackPlan(topic string) *LessonPlan {
	if topic == "" {
		topic = "二叉树遍历"
	}
	return parseLessonPlanJSON("", topic)
}
