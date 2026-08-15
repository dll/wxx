package handler

import (
	"net/http"

	"github.com/dll/wxx/server/internal/service"
	"github.com/gin-gonic/gin"
)

// UnionHandler 学生会角色 AI 功能接口
type UnionHandler struct {
	svc *service.UnionService
}

func NewUnionHandler(svc *service.UnionService) *UnionHandler {
	return &UnionHandler{svc: svc}
}

// EventPlan AI 活动策划
func (h *UnionHandler) EventPlan(c *gin.Context) {
	var req struct {
		Theme string `json:"theme"`
		Name  string `json:"name"`
		Type  string `json:"type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req = struct {
			Theme string `json:"theme"`
			Name  string `json:"name"`
			Type  string `json:"type"`
		}{}
	}
	eventName := req.Name
	if eventName == "" {
		eventName = req.Theme
	}
	eventType := req.Type

	if h.svc != nil {
		plan := h.svc.GenerateEventPlan(c.Request.Context(), eventType, eventName)
		if plan != nil {
			c.JSON(http.StatusOK, plan)
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"title":  eventName,
		"goal":   "丰富校园文化生活，提升学生综合素质",
		"budget": "预估经费：3000元",
		"timeline": []gin.H{
			{"phase": "策划期", "tasks": "确定方案、申请场地"},
			{"phase": "筹备期", "tasks": "宣传制作、物资采购"},
			{"phase": "执行期", "tasks": "活动执行、现场协调"},
		},
		"promotion":   "微信公众号推文 + 校园海报 + 班级群通知",
		"data_source": "fallback",
	})
}

// PosterGen AI 海报文案生成
func (h *UnionHandler) PosterGen(c *gin.Context) {
	var req struct {
		Title string `json:"title"`
		Style string `json:"style"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req = struct {
			Title string `json:"title"`
			Style string `json:"style"`
		}{}
	}
	title := req.Title
	style := req.Style

	if h.svc != nil {
		poster := h.svc.GeneratePoster(c.Request.Context(), title, style)
		if poster != nil {
			c.JSON(http.StatusOK, poster)
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"style":        style,
		"title":        title,
		"subtitle":     "滁州学院计算机学院",
		"copy":         "诚邀您的参与！",
		"color_scheme": "蓝色系",
		"layout":       "科技风格排版",
		"data_source":  "fallback",
	})
}

// ======================== P2 深度分析功能（统一 code/data 包装，与全站一致） ==========

func okData(c *gin.Context, v any) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": v})
}

// Recruitment 招新助手
func (h *UnionHandler) Recruitment(c *gin.Context) {
	dept := c.Query("dept")
	if h.svc != nil {
		data := h.svc.GenerateRecruitment(c.Request.Context(), dept)
		if data != nil {
			okData(c, data)
			return
		}
	}
	okData(c, gin.H{
		"plan":        dept + "招新方案",
		"stages":      []string{"宣传期", "报名期", "面试期"},
		"data_source": "reference",
	})
}

// MemberManage 成员管理（真实数据）
func (h *UnionHandler) MemberManage(c *gin.Context) {
	if h.svc != nil {
		data := h.svc.ManageMembers(c.Request.Context())
		if data != nil {
			okData(c, data)
			return
		}
	}
	okData(c, gin.H{"members": []gin.H{}, "stats": gin.H{}, "data_source": "reference"})
}

// Questionnaire AI 问卷生成
func (h *UnionHandler) Questionnaire(c *gin.Context) {
	topic := c.Query("topic")
	if h.svc != nil {
		data := h.svc.GenerateQuestionnaire(c.Request.Context(), topic)
		if data != nil {
			okData(c, data)
			return
		}
	}
	okData(c, gin.H{
		"title":       topic + "调查问卷",
		"questions":   []gin.H{{"type": "single_choice", "q": "满意度如何？"}},
		"data_source": "reference",
	})
}

// HotTopicTrack 热点追踪
func (h *UnionHandler) HotTopicTrack(c *gin.Context) {
	if h.svc != nil {
		data := h.svc.TrackHotTopics(c.Request.Context())
		if data != nil {
			okData(c, data)
			return
		}
	}
	okData(c, gin.H{"topics": []gin.H{}, "suggestions": []string{}, "data_source": "reference"})
}

// ActivityAnalysis 活动数据分析（真实数据）
func (h *UnionHandler) ActivityAnalysis(c *gin.Context) {
	eventName := c.Query("event")
	if h.svc != nil {
		data := h.svc.AnalyzeActivity(c.Request.Context(), eventName)
		if data != nil {
			okData(c, data)
			return
		}
	}
	okData(c, gin.H{
		"event_name":  eventName,
		"reg_rate":    0,
		"attend_rate": 0,
		"data_source": "reference",
	})
}
