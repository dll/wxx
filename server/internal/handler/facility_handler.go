package handler

import (
	"net/http"
	"strconv"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/service"
	"github.com/gin-gonic/gin"
)

// FacilityHandler 后勤服务台接口（并入教辅角色）
type FacilityHandler struct {
	svc *service.FacilityService
}

func NewFacilityHandler(svc *service.FacilityService) *FacilityHandler {
	return &FacilityHandler{svc: svc}
}

func getOperator(c *gin.Context) (int64, string) {
	if u := middleware.GetUserContext(c); u != nil {
		return u.UserID, u.Username
	}
	return 0, ""
}

// RoleMeta 岗位类型下拉
func (h *FacilityHandler) RoleMeta(c *gin.Context) {
	if h.svc == nil {
		c.JSON(http.StatusOK, gin.H{"roles": map[string]string{}, "data_source": "real"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"roles": h.svc.RoleMeta(), "data_source": "real"})
}

// CreateRecord 登记一条后勤服务记录（真实数据）
func (h *FacilityHandler) CreateRecord(c *gin.Context) {
	var req struct {
		Role        string `json:"role"`
		Title       string `json:"title"`
		Location    string `json:"location"`
		Detail      string `json:"detail"`
		StudentID   int64  `json:"student_id"`
		StudentName string `json:"student_name"`
		OccurredAt  string `json:"occurred_at"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求参数错误"})
		return
	}
	opID, opName := getOperator(c)
	if opID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录或登录已过期"})
		return
	}
	rec := &repository.FacilityRecord{
		Role:         req.Role,
		Title:        req.Title,
		Location:     req.Location,
		Detail:       req.Detail,
		OperatorID:   opID,
		OperatorName: opName,
		StudentID:    req.StudentID,
		StudentName:  req.StudentName,
		OccurredAt:   req.OccurredAt,
	}
	id, err := h.svc.CreateRecord(c.Request.Context(), rec)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 0, "message": "登记成功", "id": id,
		"data_source": "real",
	})
}

// ListRecords 查询后勤服务记录
func (h *FacilityHandler) ListRecords(c *gin.Context) {
	role := c.Query("role")
	operatorName := c.Query("operator_name")
	studentID, _ := strconv.ParseInt(c.Query("student_id"), 10, 64)
	from := c.Query("from")
	to := c.Query("to")
	limit, _ := strconv.Atoi(c.Query("limit"))

	recs, err := h.svc.ListRecords(c.Request.Context(), role, operatorName, studentID, from, to, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "查询失败"})
		return
	}
	if recs == nil {
		recs = []repository.FacilityRecord{}
	}
	c.JSON(http.StatusOK, gin.H{
		"records": recs, "count": len(recs), "data_source": "real",
	})
}

// Dashboard 后勤台看板
func (h *FacilityHandler) Dashboard(c *gin.Context) {
	opID, _ := getOperator(c)
	from := c.Query("from")
	to := c.Query("to")
	data, err := h.svc.Dashboard(c.Request.Context(), opID, from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "看板加载失败"})
		return
	}
	c.JSON(http.StatusOK, data)
}
