package handler

import (
	"net/http"
	"strconv"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/service"
	"github.com/gin-gonic/gin"
)

// SecretaryOutcomeHandler 书记教育成果接口
// 毕业去向登记（学生自报/教辅录入）+ 审核 + 书记教育成果大屏。
type SecretaryOutcomeHandler struct {
	svc *service.SecretaryOutcomeService
}

func NewSecretaryOutcomeHandler(svc *service.SecretaryOutcomeService) *SecretaryOutcomeHandler {
	return &SecretaryOutcomeHandler{svc: svc}
}

func outcomeRole(c *gin.Context) (int64, string, string) {
	if u := middleware.GetUserContext(c); u != nil {
		return u.UserID, u.Username, u.Role
	}
	return 0, "", ""
}

// OutcomeMeta 去向类型下拉
func (h *SecretaryOutcomeHandler) OutcomeMeta(c *gin.Context) {
	if h.svc == nil {
		c.JSON(http.StatusOK, gin.H{"outcome_types": map[string]string{}, "data_source": "real"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"outcome_types": h.svc.OutcomeTypeMeta(), "data_source": "real"})
}

// SubmitOutcome 登记毕业去向（学生自报=带 student_id + role=student；教辅录入=可代填任意学生）
func (h *SecretaryOutcomeHandler) SubmitOutcome(c *gin.Context) {
	var req struct {
		StudentID    int64  `json:"student_id"`
		StudentName  string `json:"student_name"`
		College      string `json:"college"`
		Major        string `json:"major"`
		GraduateYear int    `json:"graduate_year"`
		OutcomeType  string `json:"outcome_type"`
		EmployerName string `json:"employer_name"`
		Position     string `json:"position"`
		Remark       string `json:"remark"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求参数错误"})
		return
	}
	opID, opName, opRole := outcomeRole(c)
	if opID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录或登录已过期"})
		return
	}
	// 学生自报时强制绑定本人
	isStudent := opRole == "student"
	if isStudent {
		req.StudentID = opID
		req.StudentName = opName
	}
	if req.StudentID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请选择毕业生"})
		return
	}
	o := &repository.GraduationOutcome{
		StudentID:     req.StudentID,
		StudentName:   req.StudentName,
		College:       req.College,
		Major:         req.Major,
		GraduateYear:  req.GraduateYear,
		OutcomeType:   req.OutcomeType,
		EmployerName:  req.EmployerName,
		Position:      req.Position,
		Remark:        req.Remark,
		Status:        "pending",
		SubmittedBy:   opID,
		SubmittedRole: opRole,
	}
	id, err := h.svc.SubmitOutcome(c.Request.Context(), o)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 0, "message": "登记成功，待教辅审核", "id": id,
		"status": "pending", "data_source": "real",
	})
}

// ListOutcomes 查询毕业去向
func (h *SecretaryOutcomeHandler) ListOutcomes(c *gin.Context) {
	status := c.Query("status")
	college := c.Query("college")
	year, _ := strconv.Atoi(c.Query("year"))
	limit, _ := strconv.Atoi(c.Query("limit"))
	opID, _, opRole := outcomeRole(c)
	// 学生只能查自己的去向；教辅/书记可按过滤条件查
	var studentID int64
	if opRole == "student" {
		studentID = opID
	} else {
		studentID, _ = strconv.ParseInt(c.Query("student_id"), 10, 64)
	}
	list, err := h.svc.ListOutcomes(c.Request.Context(), status, college, year, studentID, limit)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if list == nil {
		list = []repository.GraduationOutcome{}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "list": list, "data_source": "real"})
}

// ReviewOutcome 审核毕业去向（教辅）
func (h *SecretaryOutcomeHandler) ReviewOutcome(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req struct {
		Status string `json:"status"` // approved/rejected
		Note   string `json:"note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求参数错误"})
		return
	}
	opID, opName, _ := outcomeRole(c)
	if opID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录或登录已过期"})
		return
	}
	if err := h.svc.ReviewOutcome(c.Request.Context(), id, opID, opName, req.Status, req.Note); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	msg := "已通过（计入教育成果统计）"
	if req.Status == "rejected" {
		msg = "已驳回"
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": msg})
}

// CountPending 待审核角标
func (h *SecretaryOutcomeHandler) CountPending(c *gin.Context) {
	n, err := h.svc.CountPending(c.Request.Context())
	if err != nil {
		n = 0
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "pending": n, "data_source": "real"})
}

// OutcomeDashboard 书记教育成果大屏（school_admin 全校 college=''；college_admin 本院 college=角色归属）
func (h *SecretaryOutcomeHandler) OutcomeDashboard(c *gin.Context) {
	college := c.Query("college")
	// 若未显式传学院，尝试从用户上下文取角色归属（college_admin 看本院）
	if college == "" {
		if u := middleware.GetUserContext(c); u != nil && u.Role == "college_admin" && u.OwnerScope == "college" {
			college = u.OwnerID
		}
	}
	data, err := h.svc.OutcomeDashboard(c.Request.Context(), college)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": data})
}
