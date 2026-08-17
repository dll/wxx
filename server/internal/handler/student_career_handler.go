package handler

import (
	"fmt"
	"net/http"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/gin-gonic/gin"
)

// MockInterview AI 模拟面试
func (h *StudentHandler) MockInterview(c *gin.Context) {
	position := c.Query("position")
	if h.svc != nil {
		result := h.svc.GenerateMockInterview(c.Request.Context(), position)
		if result != nil {
			c.JSON(http.StatusOK, result)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"position":    position,
		"questions":   []gin.H{{"q": "请做一下自我介绍", "tip": "突出技术栈和项目经验"}, {"q": "你如何看待失败？", "tip": "展示成长心态"}},
		"score":       78,
		"suggestions": []string{"技术表达可以更结构化", "建议多准备行为面试问题"},
		"data_source": "fallback",
	})
}

// Resume AI 智能简历生成
func (h *StudentHandler) Resume(c *gin.Context) {
	position := c.Query("position")
	if h.svc != nil {
		userCtx := middleware.GetUserContext(c)
		if userCtx != nil {
			result := h.svc.GenerateResume(c.Request.Context(), userCtx.UserID, position)
			if result != nil {
				c.JSON(http.StatusOK, result)
				return
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"template":    "modern",
		"sections":    []string{"个人信息", "教育背景", "项目经历", "技能特长", "荣誉奖项"},
		"data_source": "fallback",
	})
}

// CareerSimulation 职业模拟器
func (h *StudentHandler) CareerSimulation(c *gin.Context) {
	careerPath := c.Query("career")
	if h.svc != nil {
		result := h.svc.GenerateCareerSimulation(c.Request.Context(), careerPath)
		if result != nil {
			c.JSON(http.StatusOK, result)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"career":      careerPath,
		"stages":      []string{"应届生", "3年经验", "5年+", "资深/管理"},
		"data_source": "fallback",
	})
}

// StudyBuddyMatch AI 学友匹配
func (h *StudentHandler) StudyBuddyMatch(c *gin.Context) {
	if h.svc != nil {
		userCtx := middleware.GetUserContext(c)
		if userCtx != nil {
			result := h.svc.GenerateStudyBuddyMatches(c.Request.Context(), userCtx.UserID)
			if result != nil {
				c.JSON(http.StatusOK, result)
				return
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"matches":     []gin.H{{"name": "张同学", "reason": "数据结构互补", "match_score": 88}},
		"data_source": "fallback",
	})
}

// MentalHealthReport AI 心理健康评估报告
func (h *StudentHandler) MentalHealthReport(c *gin.Context) {
	if h.svc != nil {
		userCtx := middleware.GetUserContext(c)
		if userCtx != nil {
			result := h.svc.GenerateMentalHealthReport(c.Request.Context(), userCtx.UserID)
			if result != nil {
				c.JSON(http.StatusOK, result)
				return
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"dimensions":  gin.H{"压力": "中等", "情绪": "良好", "社交": "活跃", "韧性": "较强"},
		"suggestions": []string{"保持规律作息", "适当运动放松"},
		"data_source": "fallback",
	})
}

// NoteAssistant AI 笔记助手
func (h *StudentHandler) NoteAssistant(c *gin.Context) {
	content := c.Query("content")
	if h.svc != nil && content != "" {
		result := h.svc.GenerateNoteAssistant(c.Request.Context(), content)
		if result != nil {
			c.JSON(http.StatusOK, result)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"key_points":  []string{"知识点1", "知识点2"},
		"mind_map":    "中心主题 → 分支1 → 细节",
		"quiz":        []string{"自测题1", "自测题2"},
		"data_source": "fallback",
	})
}

// AlumniMatch AI 前辈连线
func (h *StudentHandler) AlumniMatch(c *gin.Context) {
	if h.svc != nil {
		userCtx := middleware.GetUserContext(c)
		if userCtx != nil {
			result := h.svc.GenerateAlumniMatch(c.Request.Context(), userCtx.UserID)
			if result != nil {
				c.JSON(http.StatusOK, result)
				return
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"matches":     []gin.H{{"name": "陈学长", "company": "字节跳动", "match_score": 92}},
		"data_source": "fallback",
	})
}

// ======================== P3 生态扩展 ========================

// DynamicMentor 数字人导师（动态形象版）
func (h *StudentHandler) DynamicMentor(c *gin.Context) {
	style := c.Query("style")
	if h.svc != nil {
		userCtx := middleware.GetUserContext(c)
		if userCtx != nil {
			result := h.svc.GenerateDynamicMentor(c.Request.Context(), userCtx.UserID, style)
			if result != nil {
				c.JSON(http.StatusOK, result)
				return
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"name":         "蔚小芯",
		"avatar_style": style,
		"greeting":     fmt.Sprintf("你好！我是你的%s风格AI导师。今天我们一起努力吧！", style),
		"data_source":  "fallback",
	})
}

// EnhancedCareerSim 职业模拟器增强版
func (h *StudentHandler) EnhancedCareerSim(c *gin.Context) {
	careerPath := c.Query("career")
	if h.svc != nil {
		result := h.svc.GenerateEnhancedCareerSimulation(c.Request.Context(), careerPath)
		if result != nil {
			c.JSON(http.StatusOK, result)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"career_path": careerPath,
		"stages":      []string{"在校期", "应届生", "3年经验", "5年+"},
		"data_source": "fallback",
	})
}
