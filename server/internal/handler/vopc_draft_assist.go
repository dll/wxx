package handler

import (
	"net/http"
	"strings"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/gin-gonic/gin"
)

// AssistProjectDraft turns a student's short idea into a reviewable G0 draft.
// This is deliberately deterministic and transparent: it provides a safe local
// example when no model is available, never pretends generated text is factual,
// and lets the student edit every field before saving or submitting.
func (h *VOPCHandler) AssistProjectDraft(c *gin.Context) {
	var in struct {
		Idea        string `json:"idea"`
		ProjectType string `json:"project_type"`
		TargetUsers string `json:"target_users"`
	}
	if c.ShouldBindJSON(&in) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求 JSON 格式错误"})
		return
	}
	in.Idea = strings.TrimSpace(in.Idea)
	in.ProjectType = strings.TrimSpace(in.ProjectType)
	in.TargetUsers = strings.TrimSpace(in.TargetUsers)
	if in.Idea == "" {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"code": 422, "message": "请先输入一句项目想法"})
		return
	}
	if len([]rune(in.Idea)) > 500 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"code": 422, "message": "项目想法不超过 500 字"})
		return
	}
	if in.ProjectType == "" {
		in.ProjectType = "自由探索项目"
	}
	if in.TargetUsers == "" {
		in.TargetUsers = "有该需求的在校学生"
	}

	// Keep generated copy grounded in the student's own idea. These are
	// suggestions, not claims about school policy or user research results.
	fields := map[string]string{
		"name":                shortProjectName(in.Idea),
		"summary":             "围绕“" + in.Idea + "”制作一个可在校园场景中验证的最小成果。",
		"problem_statement":   "目前“" + in.Idea + "”对应的需求缺少清晰、低成本的解决路径，学生需要更容易开始和反馈的方案。",
		"target_users":        in.TargetUsers,
		"expected_outcome":    "完成一个可演示的最小版本，并通过至少 5 名目标用户的访谈或试用记录验证核心假设。",
		"validation_plan":     "先访谈 5 名目标用户，再制作低保真原型；根据反馈迭代 1 次，记录问题、建议和是否满足验收标准。",
		"product_form":        "可交互原型或 Web 应用",
		"project_cycle":       "4 周",
		"acceptance_criteria": "核心流程可完整演示；至少 5 份真实反馈记录；列出已知限制与下一步计划。",
		"mentor_needs":        "产品范围与用户验证方法指导",
		"resource_needs":      "原型工具、测试用户和基础开发环境",
	}

	u := middleware.GetUserContext(c)
	role := "student"
	if u != nil && strings.TrimSpace(u.Role) != "" {
		role = u.Role
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{
		"fields":             fields,
		"project_type":       in.ProjectType,
		"data_source":        "template",
		"provider":           "local-rule-assist",
		"review_required":    true,
		"rationale":          "内容仅根据你输入的想法生成结构化示例，不代表真实调研或学校政策；请逐项核对后再保存。",
		"generated_for_role": role,
	}})
}

func shortProjectName(idea string) string {
	idea = strings.TrimSpace(idea)
	if len([]rune(idea)) > 18 {
		idea = string([]rune(idea)[:18]) + "…"
	}
	return idea + "校园实践"
}
