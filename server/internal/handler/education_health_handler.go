package handler

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"log"
	"math"
	"net/http"
	"strconv"
	"time"

	dbutil "github.com/dll/wxx/server/internal/db"
	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/gin-gonic/gin"
)

// ── 学生「身体健康」模块 handler ──
// 三部分：身体基本信息（每用户一行）、体检记录、病历记录。
// 全部按 user_id 归属，仅本人可读写（handler 内强制 user_id = 当前用户）。

// getCurrentUserID 提取当前登录用户 ID
func getCurrentUserID(c *gin.Context) int64 {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		return 0
	}
	return userCtx.UserID
}

// ── 身体基本信息 ──

// GetHealthBasicInfo 获取本人身体基本信息
// GET /api/v1/health/basic
func (h *EducationHandler) GetHealthBasicInfo(c *gin.Context) {
	userID := getCurrentUserID(c)
	if userID <= 0 {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未认证"})
		return
	}

	var info struct {
		ID               int64   `json:"id"`
		HeightCm         float64 `json:"height_cm"`
		WeightKg         float64 `json:"weight_kg"`
		BloodType        string  `json:"blood_type"`
		VisionLeft       string  `json:"vision_left"`
		VisionRight      string  `json:"vision_right"`
		Allergies        string  `json:"allergies"`
		PastIllness      string  `json:"past_illness"`
		FamilyHistory    string  `json:"family_history"`
		EmergencyContact string  `json:"emergency_contact"`
		EmergencyPhone   string  `json:"emergency_phone"`
		UpdatedAt        string  `json:"updated_at"`
	}

	err := h.db.QueryRow(
		`SELECT id, height_cm, weight_kg, blood_type, vision_left, vision_right,
		        allergies, past_illness, family_history, emergency_contact, emergency_phone, updated_at
		 FROM health_basic_info WHERE user_id = ?`,
		userID,
	).Scan(&info.ID, &info.HeightCm, &info.WeightKg, &info.BloodType, &info.VisionLeft,
		&info.VisionRight, &info.Allergies, &info.PastIllness, &info.FamilyHistory,
		&info.EmergencyContact, &info.EmergencyPhone, &info.UpdatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": nil})
		return
	}
	if err != nil {
		log.Printf("health GetHealthBasicInfo err: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询身体信息失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": info})
}

// UpsertHealthBasicInfo 保存本人身体基本信息（不存在则插入，存在则更新）
// PUT /api/v1/health/basic
func (h *EducationHandler) UpsertHealthBasicInfo(c *gin.Context) {
	userID := getCurrentUserID(c)
	if userID <= 0 {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未认证"})
		return
	}

	var req struct {
		HeightCm         *float64 `json:"height_cm"`
		WeightKg         *float64 `json:"weight_kg"`
		BloodType        string   `json:"blood_type"`
		VisionLeft       string   `json:"vision_left"`
		VisionRight      string   `json:"vision_right"`
		Allergies        string   `json:"allergies"`
		PastIllness      string   `json:"past_illness"`
		FamilyHistory    string   `json:"family_history"`
		EmergencyContact string   `json:"emergency_contact"`
		EmergencyPhone   string   `json:"emergency_phone"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "请求参数错误"})
		return
	}

	height := float64(0)
	weight := float64(0)
	if req.HeightCm != nil {
		height = *req.HeightCm
	}
	if req.WeightKg != nil {
		weight = *req.WeightKg
	}

	stmt := `INSERT INTO health_basic_info
		   (user_id, height_cm, weight_kg, blood_type, vision_left, vision_right,
		    allergies, past_illness, family_history, emergency_contact, emergency_phone, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now','localtime'))
		 ON CONFLICT(user_id) DO UPDATE SET
		   height_cm = excluded.height_cm, weight_kg = excluded.weight_kg,
		   blood_type = excluded.blood_type, vision_left = excluded.vision_left,
		   vision_right = excluded.vision_right, allergies = excluded.allergies,
		   past_illness = excluded.past_illness, family_history = excluded.family_history,
		   emergency_contact = excluded.emergency_contact, emergency_phone = excluded.emergency_phone,
		   updated_at = datetime('now','localtime')`
	_, err := h.db.Exec(dbutil.AdaptForDriver(stmt, dbutil.DriverOf(h.db)),
		userID, height, weight, req.BloodType, req.VisionLeft, req.VisionRight,
		req.Allergies, req.PastIllness, req.FamilyHistory, req.EmergencyContact, req.EmergencyPhone,
	)
	if err != nil {
		log.Printf("health UpsertHealthBasicInfo err: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "保存身体信息失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已保存"})
}

// ── 体检记录 ──

type healthCheckupItem struct {
	ID          int64    `json:"id"`
	CheckupDate string   `json:"checkup_date"`
	Hospital    string   `json:"hospital"`
	Conclusion  string   `json:"conclusion"`
	Details     string   `json:"details"`
	Attachments []string `json:"attachments"`
	CreatedAt   string   `json:"created_at"`
}

// ListHealthCheckups 获取本人体检记录列表
// GET /api/v1/health/checkups
func (h *EducationHandler) ListHealthCheckups(c *gin.Context) {
	userID := getCurrentUserID(c)
	if userID <= 0 {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未认证"})
		return
	}

	rows, err := h.db.Query(
		`SELECT id, checkup_date, hospital, conclusion, details, attachments, created_at
		 FROM health_checkups WHERE user_id = ? ORDER BY checkup_date DESC, id DESC`,
		userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询体检记录失败"})
		return
	}
	defer rows.Close()

	var list []*healthCheckupItem
	for rows.Next() {
		item := &healthCheckupItem{}
		var att string
		if err := rows.Scan(&item.ID, &item.CheckupDate, &item.Hospital,
			&item.Conclusion, &item.Details, &att, &item.CreatedAt); err != nil {
			continue
		}
		item.Attachments = parseStringSlice(att)
		list = append(list, item)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": list})
}

// CreateHealthCheckup 新增体检记录
// POST /api/v1/health/checkups
func (h *EducationHandler) CreateHealthCheckup(c *gin.Context) {
	userID := getCurrentUserID(c)
	if userID <= 0 {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未认证"})
		return
	}

	var req struct {
		CheckupDate string   `json:"checkup_date"`
		Hospital    string   `json:"hospital"`
		Conclusion  string   `json:"conclusion"`
		Details     string   `json:"details"`
		Attachments []string `json:"attachments"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "请求参数错误"})
		return
	}

	att, _ := jsonMarshalStringSlice(req.Attachments)
	res, err := h.db.Exec(
		`INSERT INTO health_checkups (user_id, checkup_date, hospital, conclusion, details, attachments)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		userID, req.CheckupDate, req.Hospital, req.Conclusion, req.Details, att,
	)
	if err != nil {
		log.Printf("health CreateHealthCheckup err: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "新增体检记录失败"})
		return
	}
	id, _ := res.LastInsertId()

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已新增", "data": gin.H{"id": id}})
}

// UpdateHealthCheckup 更新体检记录（仅本人）
// PUT /api/v1/health/checkups/:id
func (h *EducationHandler) UpdateHealthCheckup(c *gin.Context) {
	id := c.Param("id")
	userID := getCurrentUserID(c)
	if userID <= 0 {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未认证"})
		return
	}

	var req struct {
		CheckupDate string   `json:"checkup_date"`
		Hospital    string   `json:"hospital"`
		Conclusion  string   `json:"conclusion"`
		Details     string   `json:"details"`
		Attachments []string `json:"attachments"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "请求参数错误"})
		return
	}

	att, _ := jsonMarshalStringSlice(req.Attachments)
	res, err := h.db.Exec(
		`UPDATE health_checkups SET checkup_date = ?, hospital = ?, conclusion = ?, details = ?, attachments = ?
		 WHERE id = ? AND user_id = ?`,
		req.CheckupDate, req.Hospital, req.Conclusion, req.Details, att, id, userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "更新体检记录失败"})
		return
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Code: 404, Message: "记录不存在或无权操作"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已更新"})
}

// DeleteHealthCheckup 删除体检记录（仅本人）
// DELETE /api/v1/health/checkups/:id
func (h *EducationHandler) DeleteHealthCheckup(c *gin.Context) {
	h.deleteHealthRecord(c, "health_checkups")
}

// ── 病历记录 ──

type healthRecordItem struct {
	ID          int64    `json:"id"`
	RecordDate  string   `json:"record_date"`
	Hospital    string   `json:"hospital"`
	Department  string   `json:"department"`
	Diagnosis   string   `json:"diagnosis"`
	Treatment   string   `json:"treatment"`
	Attachments []string `json:"attachments"`
	CreatedAt   string   `json:"created_at"`
}

// ListHealthRecords 获取本人病历记录列表
// GET /api/v1/health/records
func (h *EducationHandler) ListHealthRecords(c *gin.Context) {
	userID := getCurrentUserID(c)
	if userID <= 0 {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未认证"})
		return
	}

	rows, err := h.db.Query(
		`SELECT id, record_date, hospital, department, diagnosis, treatment, attachments, created_at
		 FROM health_records WHERE user_id = ? ORDER BY record_date DESC, id DESC`,
		userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询病历记录失败"})
		return
	}
	defer rows.Close()

	var list []*healthRecordItem
	for rows.Next() {
		item := &healthRecordItem{}
		var att string
		if err := rows.Scan(&item.ID, &item.RecordDate, &item.Hospital,
			&item.Department, &item.Diagnosis, &item.Treatment, &att, &item.CreatedAt); err != nil {
			continue
		}
		item.Attachments = parseStringSlice(att)
		list = append(list, item)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": list})
}

// CreateHealthRecord 新增病历记录
// POST /api/v1/health/records
func (h *EducationHandler) CreateHealthRecord(c *gin.Context) {
	userID := getCurrentUserID(c)
	if userID <= 0 {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未认证"})
		return
	}

	var req struct {
		RecordDate  string   `json:"record_date"`
		Hospital    string   `json:"hospital"`
		Department  string   `json:"department"`
		Diagnosis   string   `json:"diagnosis"`
		Treatment   string   `json:"treatment"`
		Attachments []string `json:"attachments"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "请求参数错误"})
		return
	}

	att, _ := jsonMarshalStringSlice(req.Attachments)
	res, err := h.db.Exec(
		`INSERT INTO health_records (user_id, record_date, hospital, department, diagnosis, treatment, attachments)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		userID, req.RecordDate, req.Hospital, req.Department, req.Diagnosis, req.Treatment, att,
	)
	if err != nil {
		log.Printf("health CreateHealthRecord err: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "新增病历记录失败"})
		return
	}
	id, _ := res.LastInsertId()

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已新增", "data": gin.H{"id": id}})
}

// UpdateHealthRecord 更新病历记录（仅本人）
// PUT /api/v1/health/records/:id
func (h *EducationHandler) UpdateHealthRecord(c *gin.Context) {
	id := c.Param("id")
	userID := getCurrentUserID(c)
	if userID <= 0 {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未认证"})
		return
	}

	var req struct {
		RecordDate  string   `json:"record_date"`
		Hospital    string   `json:"hospital"`
		Department  string   `json:"department"`
		Diagnosis   string   `json:"diagnosis"`
		Treatment   string   `json:"treatment"`
		Attachments []string `json:"attachments"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "请求参数错误"})
		return
	}

	att, _ := jsonMarshalStringSlice(req.Attachments)
	res, err := h.db.Exec(
		`UPDATE health_records SET record_date = ?, hospital = ?, department = ?, diagnosis = ?, treatment = ?, attachments = ?
		 WHERE id = ? AND user_id = ?`,
		req.RecordDate, req.Hospital, req.Department, req.Diagnosis, req.Treatment, att, id, userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "更新病历记录失败"})
		return
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Code: 404, Message: "记录不存在或无权操作"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已更新"})
}

// DeleteHealthRecord 删除病历记录（仅本人）
// DELETE /api/v1/health/records/:id
func (h *EducationHandler) DeleteHealthRecord(c *gin.Context) {
	h.deleteHealthRecord(c, "health_records")
}

// ── 通用删除（校验本人归属）──

func (h *EducationHandler) deleteHealthRecord(c *gin.Context, table string) {
	id := c.Param("id")
	userID := getCurrentUserID(c)
	if userID <= 0 {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未认证"})
		return
	}

	query := "DELETE FROM " + table + " WHERE id = ? AND user_id = ?"
	res, err := h.db.Exec(query, id, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "删除失败"})
		return
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Code: 404, Message: "记录不存在或无权操作"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已删除"})
}

// ── 工具 ──

// parseStringSlice 解析 JSON 字符串数组（容错：空/非法返回空切片）
func parseStringSlice(s string) []string {
	var out []string
	if s == "" {
		return out
	}
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil
	}
	return out
}

// jsonMarshalStringSlice 序列化字符串切片为 JSON（nil → "[]"）
func jsonMarshalStringSlice(s []string) (string, error) {
	if s == nil {
		return "[]", nil
	}
	b, err := json.Marshal(s)
	if err != nil {
		return "[]", err
	}
	return string(b), nil
}

// ── 日常记录（身高/体重/血压/心率，折线图可视化）──

type healthDailyItem struct {
	ID         int64   `json:"id"`
	RecordDate string  `json:"record_date"`
	HeightCm   float64 `json:"height_cm"`
	WeightKg   float64 `json:"weight_kg"`
	Systolic   int     `json:"systolic"`
	Diastolic  int     `json:"diastolic"`
	HeartRate  int     `json:"heart_rate"`
	Note       string  `json:"note"`
	CreatedAt  string  `json:"created_at"`
}

// ListHealthDaily 获取本人日常健康记录（按日期升序，用于趋势图）
// GET /api/v1/health/daily?limit=90
func (h *EducationHandler) ListHealthDaily(c *gin.Context) {
	userID := getCurrentUserID(c)
	if userID <= 0 {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未认证"})
		return
	}

	limit := 90
	if l, err := parsePositiveInt(c.Query("limit")); err == nil && l > 0 && l <= 365 {
		limit = l
	}

	rows, err := h.db.Query(
		`SELECT id, record_date, height_cm, weight_kg, systolic, diastolic, heart_rate, note, created_at
		 FROM health_daily_records WHERE user_id = ? ORDER BY record_date ASC LIMIT ?`,
		userID, limit,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询日常记录失败"})
		return
	}
	defer rows.Close()

	var list []*healthDailyItem
	for rows.Next() {
		item := &healthDailyItem{}
		if err := rows.Scan(&item.ID, &item.RecordDate, &item.HeightCm, &item.WeightKg,
			&item.Systolic, &item.Diastolic, &item.HeartRate, &item.Note, &item.CreatedAt); err != nil {
			continue
		}
		list = append(list, item)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": list})
}

// UpsertHealthDaily 新增/更新本人某日健康记录（同用户同日唯一，upsert）
// PUT /api/v1/health/daily
func (h *EducationHandler) UpsertHealthDaily(c *gin.Context) {
	userID := getCurrentUserID(c)
	if userID <= 0 {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未认证"})
		return
	}

	var req struct {
		RecordDate string   `json:"record_date" binding:"required"`
		HeightCm   *float64 `json:"height_cm"`
		WeightKg   *float64 `json:"weight_kg"`
		Systolic   *int     `json:"systolic"`
		Diastolic  *int     `json:"diastolic"`
		HeartRate  *int     `json:"heart_rate"`
		Note       string   `json:"note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "请求参数错误"})
		return
	}

	height := float64(0)
	weight := float64(0)
	systolic := 0
	diastolic := 0
	heartRate := 0
	if req.HeightCm != nil {
		height = *req.HeightCm
	}
	if req.WeightKg != nil {
		weight = *req.WeightKg
	}
	if req.Systolic != nil {
		systolic = *req.Systolic
	}
	if req.Diastolic != nil {
		diastolic = *req.Diastolic
	}
	if req.HeartRate != nil {
		heartRate = *req.HeartRate
	}

	stmt := `INSERT INTO health_daily_records
		   (user_id, record_date, height_cm, weight_kg, systolic, diastolic, heart_rate, note, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, datetime('now','localtime'))
		 ON CONFLICT(user_id, record_date) DO UPDATE SET
		   height_cm = excluded.height_cm, weight_kg = excluded.weight_kg,
		   systolic = excluded.systolic, diastolic = excluded.diastolic,
		   heart_rate = excluded.heart_rate, note = excluded.note,
		   updated_at = datetime('now','localtime')`
	_, err := h.db.Exec(dbutil.AdaptForDriver(stmt, dbutil.DriverOf(h.db)),
		userID, req.RecordDate, height, weight, systolic, diastolic, heartRate, req.Note,
	)
	if err != nil {
		log.Printf("health UpsertHealthDaily err: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "保存日常记录失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已保存"})
}

// DeleteHealthDaily 删除某日健康记录（仅本人）
// DELETE /api/v1/health/daily/:date
func (h *EducationHandler) DeleteHealthDaily(c *gin.Context) {
	userID := getCurrentUserID(c)
	if userID <= 0 {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未认证"})
		return
	}
	date := c.Param("date")
	if date == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "缺少日期参数"})
		return
	}

	res, err := h.db.Exec(
		`DELETE FROM health_daily_records WHERE user_id = ? AND record_date = ?`,
		userID, date,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "删除失败"})
		return
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Code: 404, Message: "记录不存在或无权操作"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已删除"})
}

// parsePositiveInt 解析正整数（容错）
func parsePositiveInt(s string) (int, error) {
	if s == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return 0, err
	}
	return n, nil
}

// ── 健身活动 / 竞技比赛（与校园文化数据链通）──

type healthActivityItem struct {
	ActivityID     string `json:"activity_id"`
	Title          string `json:"title"`
	Category       string `json:"category"`
	Description    string `json:"description"`
	StartAt        string `json:"start_at"`
	EndAt          string `json:"end_at"`
	Venue          string `json:"venue"`
	Organizer      string `json:"organizer"`
	Capacity       int    `json:"capacity"`
	SignupDeadline string `json:"signup_deadline"`
	Status         string `json:"status"`
	CreatorRole    string `json:"creator_role"`
	// 关注/报名统计与当前用户状态
	FavoriteCount int  `json:"favorite_count"`
	SignupCount   int  `json:"signup_count"`
	IsFavorite    bool `json:"is_favorite"`
	IsSignup      bool `json:"is_signup"`
}

// ListHealthActivities 获取活动列表（按开始时间排序，含关注/报名状态）
// GET /api/v1/health/activities?category=sports
// student_union/管理员 可看全部；普通学生看 active
func (h *EducationHandler) ListHealthActivities(c *gin.Context) {
	userID := getCurrentUserID(c)
	if userID <= 0 {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未认证"})
		return
	}
	category := c.Query("category")

	where := "WHERE a.status = 'active'"
	var args []interface{}
	if category != "" {
		where += " AND a.category = ?"
		args = append(args, category)
	}

	rows, err := h.db.Query(
		`SELECT a.activity_id, a.title, a.category, a.description, a.start_at, a.end_at,
		        a.venue, a.organizer, a.capacity, a.signup_deadline, a.status, a.creator_role,
		        (SELECT COUNT(*) FROM health_activity_favorites f WHERE f.activity_id = a.activity_id) AS fav,
		        (SELECT COUNT(*) FROM health_activity_signups s WHERE s.activity_id = a.activity_id AND s.status='registered') AS sg,
		        EXISTS(SELECT 1 FROM health_activity_favorites f2 WHERE f2.activity_id = a.activity_id AND f2.user_id = ?) AS is_fav,
		        EXISTS(SELECT 1 FROM health_activity_signups s2 WHERE s2.activity_id = a.activity_id AND s2.user_id = ? AND s2.status='registered') AS is_sg
		 FROM health_activities a `+where+` ORDER BY a.start_at DESC LIMIT 200`,
		append(args, userID, userID)...,
	)
	if err != nil {
		log.Printf("health ListHealthActivities err: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询活动失败"})
		return
	}
	defer rows.Close()

	var list []*healthActivityItem
	for rows.Next() {
		item := &healthActivityItem{}
		var isFav, isSg int
		if err := rows.Scan(&item.ActivityID, &item.Title, &item.Category, &item.Description,
			&item.StartAt, &item.EndAt, &item.Venue, &item.Organizer, &item.Capacity,
			&item.SignupDeadline, &item.Status, &item.CreatorRole,
			&item.FavoriteCount, &item.SignupCount, &isFav, &isSg); err != nil {
			continue
		}
		item.IsFavorite = isFav == 1
		item.IsSignup = isSg == 1
		list = append(list, item)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": list})
}

// CreateHealthActivity 发起健身/竞技活动（学生会 student_union 及以上）
// POST /api/v1/health/activities
func (h *EducationHandler) CreateHealthActivity(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未认证"})
		return
	}

	var req struct {
		ActivityID     string `json:"activity_id"`
		Title          string `json:"title" binding:"required"`
		Category       string `json:"category"`
		Description    string `json:"description"`
		StartAt        string `json:"start_at"`
		EndAt          string `json:"end_at"`
		Venue          string `json:"venue"`
		Organizer      string `json:"organizer"`
		Capacity       int    `json:"capacity"`
		SignupDeadline string `json:"signup_deadline"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "请求参数错误"})
		return
	}
	if req.ActivityID == "" {
		req.ActivityID = "act-" + time.Now().Format("20060102") + "-" + randHex(6)
	}
	if req.Category == "" {
		req.Category = "sports"
	}
	if req.Organizer == "" {
		req.Organizer = userCtx.Username
	}

	_, err := h.db.Exec(
		`INSERT INTO health_activities
		   (activity_id, title, category, description, start_at, end_at, venue, organizer, capacity, signup_deadline, status, creator_id, creator_role)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'active', ?, ?)`,
		req.ActivityID, req.Title, req.Category, req.Description, req.StartAt, req.EndAt,
		req.Venue, req.Organizer, req.Capacity, req.SignupDeadline, userCtx.UserID, userCtx.Role,
	)
	if err != nil {
		log.Printf("health CreateHealthActivity err: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "发布活动失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "活动已发布", "data": gin.H{"activity_id": req.ActivityID}})
}

// ToggleActivityFavorite 关注/取消关注活动
// POST /api/v1/health/activities/:id/favorite  body: {favorite: bool}
func (h *EducationHandler) ToggleActivityFavorite(c *gin.Context) {
	userID := getCurrentUserID(c)
	if userID <= 0 {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未认证"})
		return
	}
	activityID := c.Param("id")
	var req struct {
		Favorite bool `json:"favorite"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "请求参数错误"})
		return
	}

	if req.Favorite {
		_, err := h.db.Exec(
			dbutil.InsertIgnore(dbutil.DriverOf(h.db))+` health_activity_favorites (user_id, activity_id) VALUES (?, ?)`,
			userID, activityID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "操作失败"})
			return
		}
	} else {
		_, err := h.db.Exec(
			`DELETE FROM health_activity_favorites WHERE user_id = ? AND activity_id = ?`,
			userID, activityID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "操作失败"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

// UpdateHealthActivityStatus 更新活动状态（进行中→已结束/已关闭），用于活动生命周期的"结束"阶段
// POST /api/v1/health/activities/:id/status  body: {status: "active"|"closed"|"ended"}
func (h *EducationHandler) UpdateHealthActivityStatus(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未认证"})
		return
	}
	var req struct {
		Status string `json:"status" binding:"required,oneof=active closed ended"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "status 需为 active/closed/ended"})
		return
	}
	res, err := h.db.Exec(`UPDATE health_activities SET status=? WHERE activity_id=?`, req.Status, c.Param("id"))
	if err != nil {
		log.Printf("health UpdateHealthActivityStatus err: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "更新活动状态失败"})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Code: 404, Message: "活动不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "活动状态已更新", "status": req.Status})
}

// AttendActivitySignup 活动签到：学生会将某报名者标记为到场（复盘用）
// POST /api/v1/health/activities/:id/attend/:uid  body: {attended: bool}
func (h *EducationHandler) AttendActivitySignup(c *gin.Context) {
	var req struct {
		Attended bool `json:"attended"`
	}
	_ = c.ShouldBindJSON(&req)
	attach := 0
	if req.Attended {
		attach = 1
	}
	res, err := h.db.Exec(
		`UPDATE health_activity_signups SET attended=? WHERE activity_id=? AND user_id=? AND status='registered'`,
		attach, c.Param("id"), c.Param("uid"))
	if err != nil {
		log.Printf("health AttendActivitySignup err: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "签到失败"})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Code: 404, Message: "该用户未报名或活动不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "签到状态已更新", "attended": req.Attended})
}

// ActivityReviewStats 活动复盘指标（真实统计，非 mock）
// GET /api/v1/health/activities/review-stats
// 返回：每个活动 报名/到场/到场率 + 汇总（总报名/总到场/平均到场率/分类分布/组织方排行）
func (h *EducationHandler) ActivityReviewStats(c *gin.Context) {
	type row struct {
		ActivityID  string  `json:"activity_id"`
		Title       string  `json:"title"`
		Category    string  `json:"category"`
		Venue       string  `json:"venue"`
		Organizer   string  `json:"organizer"`
		Status      string  `json:"status"`
		SignupCount int     `json:"signup_count"`
		AttendCount int     `json:"attend_count"`
		AttendRate  float64 `json:"attend_rate"`
	}
	rows, err := h.db.Query(`
		SELECT a.activity_id, a.title, a.category, a.venue, a.organizer, a.status,
		       (SELECT COUNT(*) FROM health_activity_signups s WHERE s.activity_id=a.activity_id AND s.status='registered') AS sg,
		       (SELECT COUNT(*) FROM health_activity_signups s WHERE s.activity_id=a.activity_id AND s.status='registered' AND s.attended=1) AS at
		FROM health_activities a
		ORDER BY sg DESC`)
	if err != nil {
		log.Printf("health ActivityReviewStats err: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "获取复盘数据失败"})
		return
	}
	defer rows.Close()
	var items []row
	totalSignup, totalAttend := 0, 0
	catCount := map[string]int{}
	orgStat := map[string][2]int{} // organizer -> [报名, 到场]
	for rows.Next() {
		var r row
		var sg, at int
		if err := rows.Scan(&r.ActivityID, &r.Title, &r.Category, &r.Venue, &r.Organizer, &r.Status, &sg, &at); err != nil {
			continue
		}
		r.SignupCount = sg
		r.AttendCount = at
		if sg > 0 {
			r.AttendRate = math.Round(float64(at)/float64(sg)*1000) / 10
		}
		items = append(items, r)
		totalSignup += sg
		totalAttend += at
		catCount[r.Category]++
		o := r.Organizer
		if o == "" {
			o = "未知组织方"
		}
		orgStat[o] = [2]int{orgStat[o][0] + sg, orgStat[o][1] + at}
	}
	orgRank := make([]gin.H, 0, len(orgStat))
	for k, v := range orgStat {
		rate := 0.0
		if v[0] > 0 {
			rate = math.Round(float64(v[1])/float64(v[0])*1000) / 10
		}
		orgRank = append(orgRank, gin.H{"organizer": k, "signup": v[0], "attend": v[1], "attend_rate": rate})
	}
	// 到场率降序
	for i := 0; i < len(orgRank); i++ {
		for j := i + 1; j < len(orgRank); j++ {
			if orgRank[j]["attend_rate"].(float64) > orgRank[i]["attend_rate"].(float64) {
				orgRank[i], orgRank[j] = orgRank[j], orgRank[i]
			}
		}
	}
	avgRate := 0.0
	if totalSignup > 0 {
		avgRate = math.Round(float64(totalAttend)/float64(totalSignup)*1000) / 10
	}
	c.JSON(http.StatusOK, gin.H{
		"code":         0,
		"total_signup": totalSignup,
		"total_attend": totalAttend,
		"avg_attend_rate": avgRate,
		"category_count":  catCount,
		"organizer_rank":  orgRank,
		"activities":      items,
	})
}

// ListActivitySignups 活动报名/到场名单（学生会签到用）
// GET /api/v1/health/activities/:id/signups
func (h *EducationHandler) ListActivitySignups(c *gin.Context) {
	rows, err := h.db.Query(`
		SELECT s.user_id, u.username, u.display_name, s.attended, s.created_at
		FROM health_activity_signups s
		LEFT JOIN users u ON u.id = s.user_id
		WHERE s.activity_id=? AND s.status='registered'
		ORDER BY s.attended ASC, s.created_at ASC`, c.Param("id"))
	if err != nil {
		log.Printf("health ListActivitySignups err: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "获取名单失败"})
		return
	}
	defer rows.Close()
	var list []gin.H
	for rows.Next() {
		var uid int
		var uname, disp string
		var att int
		var created string
		if err := rows.Scan(&uid, &uname, &disp, &att, &created); err != nil {
			continue
		}
		show := disp
		if show == "" {
			show = uname
		}
		list = append(list, gin.H{"user_id": uid, "name": show, "username": uname, "attended": att == 1, "created_at": created})
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "items": list})
}

// POST /api/v1/health/activities/:id/signup  body: {signup: bool}
func (h *EducationHandler) ToggleActivitySignup(c *gin.Context) {
	userID := getCurrentUserID(c)
	if userID <= 0 {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未认证"})
		return
	}
	activityID := c.Param("id")
	var req struct {
		Signup bool `json:"signup"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "请求参数错误"})
		return
	}

	if req.Signup {
		_, err := h.db.Exec(
			dbutil.InsertIgnore(dbutil.DriverOf(h.db))+` health_activity_signups (user_id, activity_id, status) VALUES (?, ?, 'registered')`,
			userID, activityID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "报名失败"})
			return
		}
	} else {
		_, err := h.db.Exec(
			`UPDATE health_activity_signups SET status = 'cancelled' WHERE user_id = ? AND activity_id = ?`,
			userID, activityID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "取消失败"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

// randHex 生成随机 hex 后缀（用于 activity_id）
func randHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "000000"
	}
	return hex.EncodeToString(b)
}
