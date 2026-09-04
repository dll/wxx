package app

import (
	"github.com/dll/wxx/server/internal/auth"
	"github.com/gin-gonic/gin"
)

// registerGraduationRoutes 注册毕设选题、里程碑和学院管理路由。
func registerGraduationRoutes(secured *gin.RouterGroup, d *deps) {
	graduation := secured.Group("/graduation")
	graduation.GET("/advisors", auth.RequireCapability(auth.SelfGraduationRead), d.graduationH.ListAdvisors)
	graduation.GET("/topics", auth.RequireCapability(auth.SelfGraduationRead), d.graduationH.ListTopics)
	graduation.GET("/topics/:id", auth.RequireCapability(auth.SelfGraduationRead), d.graduationH.GetTopic)
	graduation.POST("/select", auth.RequireCapability(auth.SelfGraduationWrite), d.graduationH.SelectTopic)
	graduation.GET("/my-selection", auth.RequireCapability(auth.SelfGraduationRead), d.graduationH.GetMySelection)
	graduation.GET("/milestones", auth.RequireCapability(auth.SelfGraduationRead), d.graduationH.ListMilestones)
	graduation.GET("/stats", auth.RequireCapability(auth.SelfGraduationRead), d.graduationH.GetStats)
	graduation.GET("/selections", auth.RequireCapability(auth.CollegeGraduationRead), d.graduationH.ListSelections)
	graduation.PUT("/selections/:id/confirm", auth.RequireCapability(auth.CollegeGraduationWrite), d.graduationH.ConfirmSelection)
	graduation.POST("/admin/topics", auth.RequireCapability(auth.CollegeGraduationWrite), d.graduationH.CreateTopic)
	graduation.PUT("/admin/topics/:id", auth.RequireCapability(auth.CollegeGraduationWrite), d.graduationH.UpdateTopic)
	graduation.DELETE("/admin/topics/:id", auth.RequireCapability(auth.CollegeGraduationWrite), d.graduationH.DeleteTopic)
}
