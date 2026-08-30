package handler

import (
	"log"
	"net/http"
	"strconv"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/gin-gonic/gin"
)

// ── 学生「身体健康」模块 handler ──
// 三部分：身体基本信息（每用户一行）、体检记录、病历记录、日常记录。
// 全部按 user_id 归属，仅本人可读写；SQL 已下沉 HealthRepo（P4-d）。

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

	info, err := h.healthRepo.GetBasicInfo(userID)
	if err != nil {
		log.Printf("health GetHealthBasicInfo err: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询身体信息失败"})
		return
	}
	if info == nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": info})
}

// UpsertHealthBasicInfo 保存本人身体基本信息
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

	if err := h.healthRepo.UpsertBasicInfo(userID, height, weight, req.BloodType, req.VisionLeft, req.VisionRight,
		req.Allergies, req.PastIllness, req.FamilyHistory, req.EmergencyContact, req.EmergencyPhone); err != nil {
		log.Printf("health UpsertHealthBasicInfo err: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "保存身体信息失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已保存"})
}

// ── 体检记录 ──

// ListHealthCheckups 获取本人体检记录列表
// GET /api/v1/health/checkups
func (h *EducationHandler) ListHealthCheckups(c *gin.Context) {
	userID := getCurrentUserID(c)
	if userID <= 0 {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未认证"})
		return
	}

	list, err := h.healthRepo.ListCheckups(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询体检记录失败"})
		return
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

	id, err := h.healthRepo.CreateCheckup(userID, req.CheckupDate, req.Hospital, req.Conclusion, req.Details, req.Attachments)
	if err != nil {
		log.Printf("health CreateHealthCheckup err: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "新增体检记录失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已新增", "data": gin.H{"id": id}})
}

// UpdateHealthCheckup 更新体检记录（仅本人）
// PUT /api/v1/health/checkups/:id
func (h *EducationHandler) UpdateHealthCheckup(c *gin.Context) {
	userID := getCurrentUserID(c)
	if userID <= 0 {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未认证"})
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "记录ID无效"})
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

	affected, err := h.healthRepo.UpdateCheckup(userID, id, req.CheckupDate, req.Hospital, req.Conclusion, req.Details, req.Attachments)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "更新体检记录失败"})
		return
	}
	if affected == 0 {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Code: 404, Message: "记录不存在或无权操作"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已更新"})
}

// DeleteHealthCheckup 删除体检记录（仅本人）
// DELETE /api/v1/health/checkups/:id
func (h *EducationHandler) DeleteHealthCheckup(c *gin.Context) {
	h.deleteHealthByID(c, h.healthRepo.DeleteCheckup)
}

// ── 病历记录 ──

// ListHealthRecords 获取本人病历记录列表
// GET /api/v1/health/records
func (h *EducationHandler) ListHealthRecords(c *gin.Context) {
	userID := getCurrentUserID(c)
	if userID <= 0 {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未认证"})
		return
	}

	list, err := h.healthRepo.ListRecords(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询病历记录失败"})
		return
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

	id, err := h.healthRepo.CreateRecord(userID, req.RecordDate, req.Hospital, req.Department, req.Diagnosis, req.Treatment, req.Attachments)
	if err != nil {
		log.Printf("health CreateHealthRecord err: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "新增病历记录失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已新增", "data": gin.H{"id": id}})
}

// UpdateHealthRecord 更新病历记录（仅本人）
// PUT /api/v1/health/records/:id
func (h *EducationHandler) UpdateHealthRecord(c *gin.Context) {
	userID := getCurrentUserID(c)
	if userID <= 0 {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未认证"})
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "记录ID无效"})
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

	affected, err := h.healthRepo.UpdateRecord(userID, id, req.RecordDate, req.Hospital, req.Department, req.Diagnosis, req.Treatment, req.Attachments)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "更新病历记录失败"})
		return
	}
	if affected == 0 {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Code: 404, Message: "记录不存在或无权操作"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已更新"})
}

// DeleteHealthRecord 删除病历记录（仅本人）
// DELETE /api/v1/health/records/:id
func (h *EducationHandler) DeleteHealthRecord(c *gin.Context) {
	h.deleteHealthByID(c, h.healthRepo.DeleteRecord)
}

// ── 通用删除（校验本人归属；表名收敛为仓库方法，杜绝动态表名拼接）──

func (h *EducationHandler) deleteHealthByID(c *gin.Context, deleteFn func(userID, id int64) (int64, error)) {
	userID := getCurrentUserID(c)
	if userID <= 0 {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未认证"})
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "记录ID无效"})
		return
	}

	affected, err := deleteFn(userID, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "删除失败"})
		return
	}
	if affected == 0 {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Code: 404, Message: "记录不存在或无权操作"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已删除"})
}

// ── 日常记录（身高/体重/血压/心率，折线图可视化）──

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

	list, err := h.healthRepo.ListDaily(userID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询日常记录失败"})
		return
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

	if err := h.healthRepo.UpsertDaily(userID, req.RecordDate, height, weight, systolic, diastolic, heartRate, req.Note); err != nil {
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

	affected, err := h.healthRepo.DeleteDaily(userID, date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "删除失败"})
		return
	}
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
