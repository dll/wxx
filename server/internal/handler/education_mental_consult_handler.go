package handler

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/gin-gonic/gin"
)

// ListCounselors 咨询师列表
// GET /api/v1/mental/counselors
func (h *EducationHandler) ListCounselors(c *gin.Context) {
	rows, err := h.db.Query(
		"SELECT id, counselor_id, name, title, avatar, gender, department, specialties, bio, working_days, available, created_at " +
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
		log.Printf("mental_health bind err: %v", err)
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "参数校验失败"})
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
