package handler

import (
	"github.com/dll/wxx/server/internal/service"
)

// StudentFeaturesHandler 学生功能 HTTP handler（竞赛+规划+入党+社团）
type StudentFeaturesHandler struct {
	svc *service.StudentFeaturesService
}

// NewStudentFeaturesHandler 创建学生功能 handler
func NewStudentFeaturesHandler(svc *service.StudentFeaturesService) *StudentFeaturesHandler {
	return &StudentFeaturesHandler{svc: svc}
}
