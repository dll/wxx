package handler

import (
	"encoding/json"
	"log"
	"math"
	"net/http"
	"strconv"

	"github.com/dll/wxx/server/internal/auth"
	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/gin-gonic/gin"
)

// 心理健康相关 handler（从 education_handler.go 按业务域拆分；SQL 已下沉 MentalHealthRepo）
func (h *EducationHandler) ListPsychScales(c *gin.Context) {
	list, err := h.mentalRepo.ListPsychScales(c.Query("category"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询量表列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    list,
		"total":   len(list),
	})
}

// GetPsychScale 量表详情（含题目）
// GET /api/v1/mental/scales/:id
func (h *EducationHandler) GetPsychScale(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "量表ID不能为空"})
		return
	}

	detail, err := h.mentalRepo.GetPsychScaleDetail(id)
	if err == repository.ErrPsychNotFound {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Code: 404, Message: "量表不存在"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询量表详情失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    detail,
	})
}

// scaleQuestion 量表题目结构
type scaleQuestion struct {
	ID      int    `json:"id"`
	Text    string `json:"text"`
	Reverse bool   `json:"reverse"`
}

// scaleInterpretation 量表解释结构
type scaleInterpretation struct {
	Levels []struct {
		Name       string  `json:"name"`
		Min        float64 `json:"min"`
		Max        float64 `json:"max"`
		Color      string  `json:"color"`
		Suggestion string  `json:"suggestion"`
	} `json:"levels"`
}

// SubmitAssessment 提交测评结果（自动计算分数）
// POST /api/v1/mental/assessments
func (h *EducationHandler) SubmitAssessment(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未认证"})
		return
	}

	var req struct {
		ScaleID string         `json:"scale_id" binding:"required"`
		Answers map[string]int `json:"answers" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("mental_health bind err: %v", err)
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "参数校验失败"})
		return
	}

	scaleName, questionsJSON, interpretationJSON, _, err := h.mentalRepo.GetScaleForScoring(req.ScaleID)
	if err == repository.ErrPsychNotFound {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Code: 404, Message: "量表不存在"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询量表失败"})
		return
	}

	var questions []scaleQuestion
	if err := json.Unmarshal([]byte(questionsJSON), &questions); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "量表题目格式错误"})
		return
	}

	var interpretation scaleInterpretation
	_ = json.Unmarshal([]byte(interpretationJSON), &interpretation)

	totalScore := 0.0
	scores := make(map[string]int)
	for _, q := range questions {
		qID := strconv.Itoa(q.ID)
		score, ok := req.Answers[qID]
		if !ok {
			continue
		}
		if q.Reverse {
			score = 4 - score
		}
		scores[qID] = score
		totalScore += float64(score)
	}

	standardScore := math.Round(totalScore*1.25*100) / 100

	level := "normal"
	resultSummary := ""
	suggestion := ""
	for _, l := range interpretation.Levels {
		if standardScore >= l.Min && standardScore <= l.Max {
			level = l.Name
			suggestion = l.Suggestion
			resultSummary = scaleName + "测评结果：" + level
			break
		}
	}

	scoresJSON, _ := json.Marshal(scores)
	answersJSON, _ := json.Marshal(req.Answers)
	recordID := generateID("assess")

	completedAt, err := h.mentalRepo.SaveAssessmentRecord(userCtx.UserID, recordID, req.ScaleID, scaleName,
		string(answersJSON), string(scoresJSON), standardScore, level, resultSummary, suggestion)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "保存测评结果失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "测评提交成功",
		"data": gin.H{
			"record_id":      recordID,
			"scale_id":       req.ScaleID,
			"scale_name":     scaleName,
			"total_score":    standardScore,
			"level":          level,
			"result_summary": resultSummary,
			"suggestion":     suggestion,
			"completed_at":   completedAt,
		},
	})
}

// ListMyAssessments 我的测评记录
// GET /api/v1/mental/assessments
func (h *EducationHandler) ListMyAssessments(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未认证"})
		return
	}

	list, err := h.mentalRepo.ListMyAssessments(userCtx.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询测评记录失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    list,
		"total":   len(list),
	})
}

// ListPsychArticles 心理科普文章列表
// GET /api/v1/mental/articles?category=&page=1&page_size=20
func (h *EducationHandler) ListPsychArticles(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	list, total, err := h.mentalRepo.ListPsychArticles(c.Query("category"), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询文章列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":      0,
		"message":   "success",
		"data":      list,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetPsychArticle 文章详情
// GET /api/v1/mental/articles/:id
func (h *EducationHandler) GetPsychArticle(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "文章ID不能为空"})
		return
	}

	detail, err := h.mentalRepo.GetPsychArticleDetail(id)
	if err == repository.ErrPsychNotFound {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Code: 404, Message: "文章不存在"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询文章详情失败"})
		return
	}

	h.mentalRepo.IncrementArticleReadCount(detail.ID)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    detail,
	})
}

// ListCrisisHotlines 危机热线列表
// GET /api/v1/mental/hotlines
func (h *EducationHandler) ListCrisisHotlines(c *gin.Context) {
	list, err := h.mentalRepo.ListCrisisHotlines()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询危机热线失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    list,
		"total":   len(list),
	})
}

// ListMyMoodDiary 我的情绪日记列表
// GET /api/v1/mental/mood?start_date=&end_date=
func (h *EducationHandler) ListMyMoodDiary(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未认证"})
		return
	}

	list, err := h.mentalRepo.ListMoodDiary(userCtx.UserID, c.Query("start_date"), c.Query("end_date"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询情绪日记失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    list,
		"total":   len(list),
	})
}

// CreateMoodDiary 记录情绪日记
// POST /api/v1/mental/mood
func (h *EducationHandler) CreateMoodDiary(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未认证"})
		return
	}

	var req struct {
		Date            string  `json:"date" binding:"required"`
		MoodScore       int     `json:"mood_score" binding:"required,min=1,max=10"`
		MoodTags        string  `json:"mood_tags"`
		Events          string  `json:"events"`
		DiaryContent    string  `json:"diary_content"`
		SleepHours      float64 `json:"sleep_hours"`
		ExerciseMinutes int     `json:"exercise_minutes"`
		SocialLevel     int     `json:"social_level"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("mental_health bind err: %v", err)
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "参数校验失败"})
		return
	}

	moodTags := req.MoodTags
	if moodTags == "" {
		moodTags = "[]"
	}

	diaryID := generateID("mood")
	updated, err := h.mentalRepo.UpsertMoodDiary(userCtx.UserID, diaryID, req.Date,
		req.MoodScore, moodTags, req.Events, req.DiaryContent, req.SleepHours, req.ExerciseMinutes, req.SocialLevel)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "保存情绪日记失败"})
		return
	}

	message := "情绪日记记录成功"
	if updated {
		message = "情绪日记已更新"
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": message,
		"data": gin.H{
			"diary_id":   diaryID,
			"date":       req.Date,
			"mood_score": req.MoodScore,
		},
	})
}

// RequireEducationAuth 是一个辅助函数，用于包装 auth.RequireCapability
// 保持与现有代码风格一致
var _ = auth.RequireCapability
