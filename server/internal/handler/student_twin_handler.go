package handler

import (
	"net/http"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/gin-gonic/gin"
)

// GrowthPath 成长路径（S1 功能7）：优先真实五维快照分阶段路线图，失败回落通用 AI 文案
func (h *StudentHandler) GrowthPath(c *gin.Context) {
	if h.svc != nil {
		if userCtx := middleware.GetUserContext(c); userCtx != nil {
			if result, err := h.svc.GenerateGrowthPath(c.Request.Context(), userCtx.UserID); err == nil && result != nil {
				c.JSON(http.StatusOK, result)
				return
			}
		}
	}
	// 兜底：无孪生快照时走通用 LLM 文案（与原 GenericAI 一致，保证前端可用）
	h.GenericAI("growth-path")(c)
}

// isStaffRole 判断角色是否为教辅/教师（走绩效画像而非学生五维）
func isStaffRole(role string) bool {
	return role == "counselor" || role == "teacher" || role == "assistant"
}

func (h *StudentHandler) DigitalTwin(c *gin.Context) {
	// 优先走真实五维聚合服务（S1.1 数字孪生数据底座）
	if h.twinSvc != nil {
		if userCtx := middleware.GetUserContext(c); userCtx != nil {
			// 教辅/教师角色走绩效画像（帮扶/咨询等 → 教师+学生+蔚小芯三方绑定）
			if isStaffRole(userCtx.Role) {
				result, err := h.twinSvc.GetStaffTwin(c.Request.Context(), userCtx.UserID)
				if err == nil && result != nil {
					c.JSON(http.StatusOK, result)
					return
				}
			} else {
				result, err := h.twinSvc.GetDigitalTwin(c.Request.Context(), userCtx.UserID)
				if err == nil && result != nil {
					c.JSON(http.StatusOK, result)
					return
				}
			}
		}
	}

	// 兜底：未注入 twinSvc 或聚合异常时返回 mock（保证前端可用，不阻断）
	c.JSON(http.StatusOK, gin.H{
		"dimensions": []gin.H{
			{"name": "学业", "score": 78.5, "label": "良好"},
			{"name": "社交", "score": 65.0, "label": "中等"},
			{"name": "身心", "score": 82.0, "label": "良好"},
			{"name": "实践", "score": 70.0, "label": "中等"},
			{"name": "创新", "score": 55.0, "label": "待提升"},
		},
		"ideal_dimensions": []gin.H{
			{"name": "学业", "score": 90.0, "label": "优秀"},
			{"name": "社交", "score": 80.0, "label": "良好"},
			{"name": "身心", "score": 85.0, "label": "良好"},
			{"name": "实践", "score": 85.0, "label": "良好"},
			{"name": "创新", "score": 75.0, "label": "良好"},
		},
		"ai_summary":  "你的学业和身心维度表现良好，社交和实践维度有提升空间。建议多参加社团活动和实习项目。",
		"suggestions": []string{"参加下周的企业宣讲会", "加入一个技术社团", "每周运动3次以上"},
		"fallback":    true,
	})
}

// Personality 性格洞察
func (h *StudentHandler) Personality(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录"})
		return
	}

	if h.personalitySvc != nil {
		result, err := h.personalitySvc.GetPersonality(c.Request.Context(), userCtx.UserID)
		if err == nil && result != nil {
			c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
			return
		}
	}

	// 兜底 mock
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"type":               "待评估",
			"label":              "数据不足",
			"description":        "暂无足够行为数据推断性格画像，请多使用系统积累数据后再查看。",
			"strengths":          []string{"持续使用中"},
			"weaknesses":         []string{"数据样本不足"},
			"career_suggestions": []string{"继续探索中"},
			"learning_style":     "需要更多学习记录来判断",
			"data_source":        "fallback",
		},
	})
}

// Achievements 积分成就（真实积分计算）
func (h *StudentHandler) Achievements(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录"})
		return
	}

	if h.phase2Svc != nil {
		if data, err := h.phase2Svc.GetAchievements(userCtx.UserID); err == nil && data != nil {
			c.JSON(http.StatusOK, data)
			return
		}
	}

	// 兜底：服务未注入时返回空成就（不展示假数据）
	c.JSON(http.StatusOK, gin.H{
		"total_points": 0, "level": 1, "level_name": "青铜", "next_level_points": 300,
		"weekly_rank": 0, "badges": []gin.H{}, "data_source": "fallback",
	})
}

// ---- 数字孪生/性格（复用现有 service）----

// TwinProfile 个人数字孪生五维画像
func (h *StudentHandler) TwinProfile(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录"})
		return
	}
	if h.twinSvc == nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": nil})
		return
	}
	result, err := h.twinSvc.GetDigitalTwin(c.Request.Context(), userCtx.UserID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

// PersonalityProfile 个人性格洞察
func (h *StudentHandler) PersonalityProfile(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录"})
		return
	}
	if h.personalitySvc == nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": nil})
		return
	}
	result, err := h.personalitySvc.GetPersonality(c.Request.Context(), userCtx.UserID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}
