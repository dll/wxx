package handler

import (
	"database/sql"
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dll/wxx/server/internal/auth"
	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/gin-gonic/gin"
)

// 心理健康相关 handler（从 education_handler.go 按业务域拆分）
func (h *EducationHandler) ListPsychScales(c *gin.Context) {
	category := c.Query("category")

	var where []string
	var args []interface{}
	where = append(where, "status = 'active'")
	if category != "" {
		where = append(where, "category = ?")
		args = append(args, category)
	}
	whereSQL := strings.Join(where, " AND ")

	rows, err := h.db.Query(
		"SELECT id, scale_id, name, abbreviation, category, description, question_count, "+
			"estimated_minutes, is_crisis, created_at "+
			"FROM psych_scales WHERE "+whereSQL+" ORDER BY is_crisis DESC, id ASC",
		args...,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询量表列表失败"})
		return
	}
	defer rows.Close()

	type ScaleItem struct {
		ID               int64  `json:"id"`
		ScaleID          string `json:"scale_id"`
		Name             string `json:"name"`
		Abbreviation     string `json:"abbreviation"`
		Category         string `json:"category"`
		Description      string `json:"description"`
		QuestionCount    int    `json:"question_count"`
		EstimatedMinutes int    `json:"estimated_minutes"`
		IsCrisis         int    `json:"is_crisis"`
		CreatedAt        string `json:"created_at"`
	}

	var list []*ScaleItem
	for rows.Next() {
		item := &ScaleItem{}
		if err := rows.Scan(&item.ID, &item.ScaleID, &item.Name, &item.Abbreviation,
			&item.Category, &item.Description, &item.QuestionCount, &item.EstimatedMinutes,
			&item.IsCrisis, &item.CreatedAt); err != nil {
			continue
		}
		list = append(list, item)
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

	type ScaleDetail struct {
		ID               int64  `json:"id"`
		ScaleID          string `json:"scale_id"`
		Name             string `json:"name"`
		Abbreviation     string `json:"abbreviation"`
		Category         string `json:"category"`
		Description      string `json:"description"`
		QuestionCount    int    `json:"question_count"`
		EstimatedMinutes int    `json:"estimated_minutes"`
		ScoringMethod    string `json:"scoring_method"`
		Interpretation   string `json:"interpretation"`
		QuestionsJSON    string `json:"questions_json"`
		IsCrisis         int    `json:"is_crisis"`
		Status           string `json:"status"`
		CreatedAt        string `json:"created_at"`
	}

	detail := &ScaleDetail{}
	err := h.db.QueryRow(
		"SELECT id, scale_id, name, abbreviation, category, description, question_count, "+
			"estimated_minutes, scoring_method, interpretation, questions_json, is_crisis, status, created_at "+
			"FROM psych_scales WHERE scale_id = ? AND status = 'active'", id,
	).Scan(&detail.ID, &detail.ScaleID, &detail.Name, &detail.Abbreviation,
		&detail.Category, &detail.Description, &detail.QuestionCount, &detail.EstimatedMinutes,
		&detail.ScoringMethod, &detail.Interpretation, &detail.QuestionsJSON,
		&detail.IsCrisis, &detail.Status, &detail.CreatedAt)
	if err == sql.ErrNoRows {
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
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}

	var req struct {
		ScaleID string            `json:"scale_id" binding:"required"`
		Answers map[string]int    `json:"answers" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "参数校验失败: " + err.Error()})
		return
	}

	var questionsJSON, interpretationJSON, scaleName, scoringMethod string
	err := h.db.QueryRow(
		"SELECT name, questions_json, interpretation, scoring_method FROM psych_scales WHERE scale_id = ? AND status = 'active'",
		req.ScaleID,
	).Scan(&scaleName, &questionsJSON, &interpretationJSON, &scoringMethod)
	if err == sql.ErrNoRows {
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

	standardScore := math.Round(totalScore * 1.25 * 100) / 100

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
	now := time.Now().Format("2006-01-02 15:04:05")

	_, err = h.db.Exec(
		"INSERT INTO psych_assessment_records (user_id, record_id, scale_id, scale_name, answers_json, scores_json, total_score, level, result_summary, suggestion, completed_at, created_at) "+
			"VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		userCtx.UserID, recordID, req.ScaleID, scaleName, string(answersJSON), string(scoresJSON),
		standardScore, level, resultSummary, suggestion, now, now,
	)
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
			"completed_at":   now,
		},
	})
}

// ListMyAssessments 我的测评记录
// GET /api/v1/mental/assessments
func (h *EducationHandler) ListMyAssessments(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}

	rows, err := h.db.Query(
		"SELECT id, record_id, scale_id, scale_name, total_score, level, result_summary, suggestion, completed_at, created_at "+
			"FROM psych_assessment_records WHERE user_id = ? ORDER BY completed_at DESC, id DESC",
		userCtx.UserID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询测评记录失败"})
		return
	}
	defer rows.Close()

	type AssessmentItem struct {
		ID            int64   `json:"id"`
		RecordID      string  `json:"record_id"`
		ScaleID       string  `json:"scale_id"`
		ScaleName     string  `json:"scale_name"`
		TotalScore    float64 `json:"total_score"`
		Level         string  `json:"level"`
		ResultSummary string  `json:"result_summary"`
		Suggestion    string  `json:"suggestion"`
		CompletedAt   string  `json:"completed_at"`
		CreatedAt     string  `json:"created_at"`
	}

	var list []*AssessmentItem
	for rows.Next() {
		item := &AssessmentItem{}
		if err := rows.Scan(&item.ID, &item.RecordID, &item.ScaleID, &item.ScaleName,
			&item.TotalScore, &item.Level, &item.ResultSummary, &item.Suggestion,
			&item.CompletedAt, &item.CreatedAt); err != nil {
			continue
		}
		list = append(list, item)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    list,
		"total":   len(list),
	})
}

// ListCounselors 咨询师列表
// GET /api/v1/mental/counselors
func (h *EducationHandler) ListCounselors(c *gin.Context) {
	rows, err := h.db.Query(
		"SELECT id, counselor_id, name, title, avatar, gender, department, specialties, bio, working_days, available, created_at "+
			"FROM counselors WHERE status = 'active' ORDER BY available DESC, id ASC",
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询咨询师列表失败"})
		return
	}
	defer rows.Close()

	type CounselorItem struct {
		ID          int64  `json:"id"`
		CounselorID string `json:"counselor_id"`
		Name        string `json:"name"`
		Title       string `json:"title"`
		Avatar      string `json:"avatar"`
		Gender      string `json:"gender"`
		Department  string `json:"department"`
		Specialties string `json:"specialties"`
		Bio         string `json:"bio"`
		WorkingDays string `json:"working_days"`
		Available   int    `json:"available"`
		CreatedAt   string `json:"created_at"`
	}

	var list []*CounselorItem
	for rows.Next() {
		item := &CounselorItem{}
		if err := rows.Scan(&item.ID, &item.CounselorID, &item.Name, &item.Title,
			&item.Avatar, &item.Gender, &item.Department, &item.Specialties,
			&item.Bio, &item.WorkingDays, &item.Available, &item.CreatedAt); err != nil {
			continue
		}
		list = append(list, item)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    list,
		"total":   len(list),
	})
}

// ListMyAppointments 我的预约
// GET /api/v1/mental/appointments
func (h *EducationHandler) ListMyAppointments(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}

	rows, err := h.db.Query(
		"SELECT id, appointment_id, counselor_id, counselor_name, appointment_date, time_slot, "+
			"appointment_type, reason, status, cancel_reason, notes, created_at, updated_at "+
			"FROM counseling_appointments WHERE user_id = ? ORDER BY appointment_date DESC, time_slot DESC, id DESC",
		userCtx.UserID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询预约记录失败"})
		return
	}
	defer rows.Close()

	type AppointmentItem struct {
		ID              int64  `json:"id"`
		AppointmentID   string `json:"appointment_id"`
		CounselorID     string `json:"counselor_id"`
		CounselorName   string `json:"counselor_name"`
		AppointmentDate string `json:"appointment_date"`
		TimeSlot        string `json:"time_slot"`
		AppointmentType string `json:"appointment_type"`
		Reason          string `json:"reason"`
		Status          string `json:"status"`
		CancelReason    string `json:"cancel_reason"`
		Notes           string `json:"notes"`
		CreatedAt       string `json:"created_at"`
		UpdatedAt       string `json:"updated_at"`
	}

	var list []*AppointmentItem
	for rows.Next() {
		item := &AppointmentItem{}
		if err := rows.Scan(&item.ID, &item.AppointmentID, &item.CounselorID, &item.CounselorName,
			&item.AppointmentDate, &item.TimeSlot, &item.AppointmentType, &item.Reason,
			&item.Status, &item.CancelReason, &item.Notes, &item.CreatedAt, &item.UpdatedAt); err != nil {
			continue
		}
		list = append(list, item)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    list,
		"total":   len(list),
	})
}

// CreateAppointment 提交预约
// POST /api/v1/mental/appointments
func (h *EducationHandler) CreateAppointment(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}

	var req struct {
		CounselorID     string `json:"counselor_id" binding:"required"`
		AppointmentDate string `json:"appointment_date" binding:"required"`
		TimeSlot        string `json:"time_slot" binding:"required"`
		AppointmentType string `json:"appointment_type"`
		Reason          string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "参数校验失败: " + err.Error()})
		return
	}

	var counselorName string
	err := h.db.QueryRow("SELECT name FROM counselors WHERE counselor_id = ? AND status = 'active'", req.CounselorID).Scan(&counselorName)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Code: 404, Message: "咨询师不存在"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询咨询师失败"})
		return
	}

	appointmentType := req.AppointmentType
	if appointmentType == "" {
		appointmentType = "face_to_face"
	}

	appointmentID := generateID("appt")
	now := time.Now().Format("2006-01-02 15:04:05")

	_, err = h.db.Exec(
		"INSERT INTO counseling_appointments (user_id, appointment_id, counselor_id, counselor_name, appointment_date, time_slot, appointment_type, reason, status, created_at, updated_at) "+
			"VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'pending', ?, ?)",
		userCtx.UserID, appointmentID, req.CounselorID, counselorName,
		req.AppointmentDate, req.TimeSlot, appointmentType, req.Reason, now, now,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "提交预约失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "预约提交成功",
		"data": gin.H{
			"appointment_id":   appointmentID,
			"counselor_id":     req.CounselorID,
			"counselor_name":   counselorName,
			"appointment_date": req.AppointmentDate,
			"time_slot":        req.TimeSlot,
			"appointment_type": appointmentType,
			"status":           "pending",
			"created_at":       now,
		},
	})
}

// ListPsychArticles 心理科普文章列表
// GET /api/v1/mental/articles?category=&page=1&page_size=20
func (h *EducationHandler) ListPsychArticles(c *gin.Context) {
	category := c.Query("category")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var where []string
	var args []interface{}
	where = append(where, "status = 'active'")
	if category != "" {
		where = append(where, "category = ?")
		args = append(args, category)
	}
	whereSQL := strings.Join(where, " AND ")

	var total int
	err := h.db.QueryRow("SELECT COUNT(*) FROM psych_articles WHERE "+whereSQL, args...).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询文章列表失败"})
		return
	}

	rows, err := h.db.Query(
		"SELECT id, article_id, title, category, summary, cover_image, author, read_count, is_crisis, tags, created_at "+
			"FROM psych_articles WHERE "+whereSQL+" ORDER BY is_crisis DESC, created_at DESC, id DESC LIMIT ? OFFSET ?",
		append(args, pageSize, offset)...,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询文章列表失败"})
		return
	}
	defer rows.Close()

	type ArticleItem struct {
		ID          int64  `json:"id"`
		ArticleID   string `json:"article_id"`
		Title       string `json:"title"`
		Category    string `json:"category"`
		Summary     string `json:"summary"`
		CoverImage  string `json:"cover_image"`
		Author      string `json:"author"`
		ReadCount   int    `json:"read_count"`
		IsCrisis    int    `json:"is_crisis"`
		Tags        string `json:"tags"`
		CreatedAt   string `json:"created_at"`
	}

	var list []*ArticleItem
	for rows.Next() {
		item := &ArticleItem{}
		if err := rows.Scan(&item.ID, &item.ArticleID, &item.Title, &item.Category,
			&item.Summary, &item.CoverImage, &item.Author, &item.ReadCount,
			&item.IsCrisis, &item.Tags, &item.CreatedAt); err != nil {
			continue
		}
		list = append(list, item)
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

	type ArticleDetail struct {
		ID         int64  `json:"id"`
		ArticleID  string `json:"article_id"`
		Title      string `json:"title"`
		Category   string `json:"category"`
		Summary    string `json:"summary"`
		Content    string `json:"content"`
		CoverImage string `json:"cover_image"`
		Author     string `json:"author"`
		ReadCount  int    `json:"read_count"`
		IsCrisis   int    `json:"is_crisis"`
		Tags       string `json:"tags"`
		Status     string `json:"status"`
		CreatedAt  string `json:"created_at"`
		UpdatedAt  string `json:"updated_at"`
	}

	detail := &ArticleDetail{}
	err := h.db.QueryRow(
		"SELECT id, article_id, title, category, summary, content, cover_image, author, "+
			"read_count, is_crisis, tags, status, created_at, updated_at "+
			"FROM psych_articles WHERE article_id = ? AND status = 'active'", id,
	).Scan(&detail.ID, &detail.ArticleID, &detail.Title, &detail.Category,
		&detail.Summary, &detail.Content, &detail.CoverImage, &detail.Author,
		&detail.ReadCount, &detail.IsCrisis, &detail.Tags, &detail.Status,
		&detail.CreatedAt, &detail.UpdatedAt)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Code: 404, Message: "文章不存在"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询文章详情失败"})
		return
	}

	_, _ = h.db.Exec("UPDATE psych_articles SET read_count = read_count + 1 WHERE id = ?", detail.ID)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    detail,
	})
}

// ListCrisisHotlines 危机热线列表
// GET /api/v1/mental/hotlines
func (h *EducationHandler) ListCrisisHotlines(c *gin.Context) {
	rows, err := h.db.Query(
		"SELECT id, hotline_id, name, phone, service_time, description, level, created_at "+
			"FROM crisis_hotlines WHERE status = 'active' ORDER BY level ASC, id ASC",
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询危机热线失败"})
		return
	}
	defer rows.Close()

	type HotlineItem struct {
		ID          int64  `json:"id"`
		HotlineID   string `json:"hotline_id"`
		Name        string `json:"name"`
		Phone       string `json:"phone"`
		ServiceTime string `json:"service_time"`
		Description string `json:"description"`
		Level       int    `json:"level"`
		CreatedAt   string `json:"created_at"`
	}

	var list []*HotlineItem
	for rows.Next() {
		item := &HotlineItem{}
		if err := rows.Scan(&item.ID, &item.HotlineID, &item.Name, &item.Phone,
			&item.ServiceTime, &item.Description, &item.Level, &item.CreatedAt); err != nil {
			continue
		}
		list = append(list, item)
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
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}

	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	var where []string
	var args []interface{}
	where = append(where, "user_id = ?")
	args = append(args, userCtx.UserID)
	if startDate != "" {
		where = append(where, "date >= ?")
		args = append(args, startDate)
	}
	if endDate != "" {
		where = append(where, "date <= ?")
		args = append(args, endDate)
	}
	whereSQL := strings.Join(where, " AND ")

	rows, err := h.db.Query(
		"SELECT id, diary_id, date, mood_score, mood_tags, events, diary_content, sleep_hours, exercise_minutes, social_level, created_at, updated_at "+
			"FROM mood_diary WHERE "+whereSQL+" ORDER BY date DESC, id DESC",
		args...,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询情绪日记失败"})
		return
	}
	defer rows.Close()

	type MoodItem struct {
		ID              int64   `json:"id"`
		DiaryID         string  `json:"diary_id"`
		Date            string  `json:"date"`
		MoodScore       int     `json:"mood_score"`
		MoodTags        string  `json:"mood_tags"`
		Events          string  `json:"events"`
		DiaryContent    string  `json:"diary_content"`
		SleepHours      float64 `json:"sleep_hours"`
		ExerciseMinutes int     `json:"exercise_minutes"`
		SocialLevel     int     `json:"social_level"`
		CreatedAt       string  `json:"created_at"`
		UpdatedAt       string  `json:"updated_at"`
	}

	var list []*MoodItem
	for rows.Next() {
		item := &MoodItem{}
		if err := rows.Scan(&item.ID, &item.DiaryID, &item.Date, &item.MoodScore,
			&item.MoodTags, &item.Events, &item.DiaryContent, &item.SleepHours,
			&item.ExerciseMinutes, &item.SocialLevel, &item.CreatedAt, &item.UpdatedAt); err != nil {
			continue
		}
		list = append(list, item)
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
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
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
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "参数校验失败: " + err.Error()})
		return
	}

	moodTags := req.MoodTags
	if moodTags == "" {
		moodTags = "[]"
	}

	diaryID := generateID("mood")
	now := time.Now().Format("2006-01-02 15:04:05")

	var existingID int64
	err := h.db.QueryRow("SELECT id FROM mood_diary WHERE user_id = ? AND date = ?", userCtx.UserID, req.Date).Scan(&existingID)
	if err == nil {
		_, err = h.db.Exec(
			"UPDATE mood_diary SET mood_score = ?, mood_tags = ?, events = ?, diary_content = ?, sleep_hours = ?, exercise_minutes = ?, social_level = ?, updated_at = ? WHERE id = ?",
			req.MoodScore, moodTags, req.Events, req.DiaryContent, req.SleepHours, req.ExerciseMinutes, req.SocialLevel, now, existingID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "更新情绪日记失败"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "情绪日记已更新",
			"data": gin.H{
				"diary_id":   diaryID,
				"date":       req.Date,
				"mood_score": req.MoodScore,
				"updated_at": now,
			},
		})
		return
	}

	if err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询情绪日记失败"})
		return
	}

	_, err = h.db.Exec(
		"INSERT INTO mood_diary (user_id, diary_id, date, mood_score, mood_tags, events, diary_content, sleep_hours, exercise_minutes, social_level, created_at, updated_at) "+
			"VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		userCtx.UserID, diaryID, req.Date, req.MoodScore, moodTags, req.Events,
		req.DiaryContent, req.SleepHours, req.ExerciseMinutes, req.SocialLevel, now, now,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "保存情绪日记失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "情绪日记记录成功",
		"data": gin.H{
			"diary_id":   diaryID,
			"date":       req.Date,
			"mood_score": req.MoodScore,
			"created_at": now,
		},
	})
}

// RequireEducationAuth 是一个辅助函数，用于包装 auth.RequireCapability
// 保持与现有代码风格一致
var _ = auth.RequireCapability
