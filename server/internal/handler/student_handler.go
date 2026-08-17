package handler

import (
	"database/sql"

	"github.com/dll/wxx/server/internal/service"
)

// StudentHandler 学生角色 AI 功能接口
type StudentHandler struct {
	svc            *service.StudentService
	twinSvc        *service.TwinService        // 数字孪生五维聚合服务，可为 nil（走兜底 mock）
	checkinSvc     *service.CheckinService     // 打卡服务，可为 nil
	personalitySvc *service.PersonalityService // 性格洞察服务，可为 nil
	phase2Svc      *service.Phase2Service      // 阶段二真实数据服务（积分/问答），可为 nil
	db             *sql.DB
}

// NewStudentHandler 创建学生 handler。svc 可为 nil（兼容旧调用），此时所有 AI 功能走兜底
func NewStudentHandler(svc *service.StudentService, db *sql.DB) *StudentHandler {
	return &StudentHandler{svc: svc, db: db}
}

// SetTwinService 注入数字孪生服务（可选依赖，装配期调用）
func (h *StudentHandler) SetTwinService(twinSvc *service.TwinService) {
	h.twinSvc = twinSvc
}

// SetCheckinService 注入打卡服务（可选依赖，装配期调用）
func (h *StudentHandler) SetCheckinService(svc *service.CheckinService) {
	h.checkinSvc = svc
}

// SetPersonalityService 注入性格洞察服务（可选依赖，装配期调用）
func (h *StudentHandler) SetPersonalityService(svc *service.PersonalityService) {
	h.personalitySvc = svc
}

// SetPhase2Service 注入阶段二真实数据服务（积分/问答，可选依赖）
func (h *StudentHandler) SetPhase2Service(svc *service.Phase2Service) {
	h.phase2Svc = svc
}
