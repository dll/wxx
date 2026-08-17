package handler

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/gin-gonic/gin"
)

func (h *StudentHandler) QAPlaza(c *gin.Context) {
	realPosts := []map[string]interface{}{}
	if h.phase2Svc != nil {
		if posts, err := h.phase2Svc.ListRealPosts(8); err == nil {
			for _, p := range posts {
				realPosts = append(realPosts, map[string]interface{}{
					"id":        p.ID,
					"title":     p.Title,
					"author":    p.Author,
					"answers":   p.Answers,
					"views":     p.Views,
					"ai_answer": "",
					"category":  p.Category,
					"real":      true,
				})
			}
		}
	}

	// 知识库 FAQ 作为补充
	var faqPosts []map[string]interface{}
	if h.svc != nil {
		plaza := h.svc.GenerateQAPlaza(c.Request.Context())
		if plaza != nil {
			faqPosts = plaza.HotQuestions
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"hot_questions": append(realPosts, faqPosts...),
		"categories":    []string{"学业", "生活", "政策", "心理", "就业", "竞赛", "综合"},
		"my_posts":      len(realPosts),
		"data_source":   "real",
	})
}

// CreateQAPost 发布问题
func (h *StudentHandler) CreateQAPost(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录"})
		return
	}
	var req struct {
		Title    string `json:"title"`
		Content  string `json:"content"`
		Category string `json:"category"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "问题标题不能为空"})
		return
	}
	if h.phase2Svc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "问答服务未就绪"})
		return
	}
	id, err := h.phase2Svc.CreateQAPost(userCtx.UserID, req.Title, req.Content, req.Category)
	if err != nil {
		log.Printf("创建问答帖失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "发布失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "id": id, "message": "发布成功"})
}

// ListQAPosts 真实帖子列表
func (h *StudentHandler) ListQAPosts(c *gin.Context) {
	if h.phase2Svc == nil {
		c.JSON(http.StatusOK, []gin.H{})
		return
	}
	posts, err := h.phase2Svc.ListRealPosts(50)
	if err != nil {
		c.JSON(http.StatusOK, []gin.H{})
		return
	}
	list := make([]gin.H, 0, len(posts))
	for _, p := range posts {
		list = append(list, gin.H{
			"id": p.ID, "title": p.Title, "content": p.Content, "author": p.Author,
			"category": p.Category, "answers": p.Answers, "views": p.Views, "created_at": p.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, list)
}

// GetQAPostDetail 帖子详情（含回答）
func (h *StudentHandler) GetQAPostDetail(c *gin.Context) {
	var id int64
	fmt.Sscanf(c.Param("id"), "%d", &id)
	if h.phase2Svc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "问答服务未就绪"})
		return
	}
	post, err := h.phase2Svc.GetPost(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "问题不存在"})
		return
	}
	answers, _ := h.phase2Svc.ListAnswers(id)
	c.JSON(http.StatusOK, gin.H{
		"id": post.ID, "title": post.Title, "content": post.Content, "author": post.Author,
		"category": post.Category, "views": post.Views, "created_at": post.CreatedAt,
		"answers": answers,
	})
}

// AnswerQAPost 回答问题
func (h *StudentHandler) AnswerQAPost(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录"})
		return
	}
	var id int64
	fmt.Sscanf(c.Param("id"), "%d", &id)
	var req struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "回答内容不能为空"})
		return
	}
	if h.phase2Svc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "问答服务未就绪"})
		return
	}
	aid, err := h.phase2Svc.AnswerPost(id, userCtx.UserID, req.Content)
	if err != nil {
		log.Printf("回答问答帖失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "回答失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "answer_id": aid, "message": "回答成功"})
}

// HotTopics 热点关注
func (h *StudentHandler) HotTopics(c *gin.Context) {
	if h.svc != nil {
		topics := h.svc.GenerateHotTopics(c.Request.Context())
		if topics != nil {
			c.JSON(http.StatusOK, topics)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"topics": []gin.H{
			{"id": "1", "title": "期中考试安排", "heat": 95, "trend": "rising", "posts": 23, "summary": "本学期期中考试集中在第10-11周，数据结构和高数为重点关注科目"},
			{"id": "2", "title": "暑期实习招聘", "heat": 82, "trend": "rising", "posts": 15, "summary": "多家互联网公司开放暑期实习岗位，建议提前准备简历和算法"},
			{"id": "3", "title": "校园网升级", "heat": 68, "trend": "stable", "posts": 12, "summary": "校园网将于下周升级至千兆，届时可能短暂断网"},
			{"id": "4", "title": "社团招新", "heat": 55, "trend": "falling", "posts": 8, "summary": "本学期第二轮社团招新已结束，共12个社团完成纳新"},
		},
		"updated_at":  time.Now().Format("2006-01-02 15:04"),
		"data_source": "fallback",
	})
}

// QALeaderboard 问答排行榜
func (h *StudentHandler) QALeaderboard(c *gin.Context) {
	if h.svc != nil {
		lb := h.svc.GenerateQALeaderboard(c.Request.Context())
		if lb != nil {
			c.JSON(http.StatusOK, lb)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"hot_questions": []gin.H{
			{"rank": 1, "title": "ACM竞赛如何入门？", "views": 256, "answers": 8, "score": 92.5},
			{"rank": 2, "title": "转专业需要什么条件？", "views": 128, "answers": 5, "score": 85.0},
			{"rank": 3, "title": "考研还是就业？", "views": 198, "answers": 12, "score": 80.3},
		},
		"top_answerers": []gin.H{
			{"rank": 1, "name": "知识达人", "answers": 23, "adopted": 15, "score": 95.0},
			{"rank": 2, "name": "热心学长", "answers": 18, "adopted": 10, "score": 82.5},
			{"rank": 3, "name": "编程高手", "answers": 12, "adopted": 8, "score": 78.0},
		},
		"contributors": []gin.H{
			{"rank": 1, "name": "知识达人", "contributions": 15, "quality_score": 4.8},
			{"rank": 2, "name": "热心学长", "contributions": 10, "quality_score": 4.5},
			{"rank": 3, "name": "学霸笔记", "contributions": 8, "quality_score": 4.3},
		},
		"period":      "本周",
		"data_source": "fallback",
	})
}

// PrivateChat 站内私聊
func (h *StudentHandler) PrivateChat(c *gin.Context) {
	if h.svc != nil {
		chat := h.svc.GeneratePrivateChat(c.Request.Context())
		if chat != nil {
			c.JSON(http.StatusOK, chat)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"conversations": []gin.H{
			{"id": "1", "name": "李辅导员", "role": "counselor", "last_message": "明天下午来办公室聊聊", "time": "10:30", "unread": 1},
			{"id": "2", "name": "张学长", "role": "student", "last_message": "ACM训练资料已发你邮箱", "time": "昨天", "unread": 0},
			{"id": "3", "name": "AI学友-王同学", "role": "student", "last_message": "明天一起去图书馆复习吧", "time": "昨天", "unread": 0},
		},
		"recommended_contacts": []gin.H{
			{"name": "赵学姐", "reason": "同专业大三，擅长算法", "match_score": 88},
			{"name": "刘同学", "reason": "学习风格互补，可组队复习", "match_score": 82},
		},
		"data_source": "fallback",
	})
}
