package app

import (
	"github.com/dll/wxx/server/internal/auth"
	"github.com/gin-gonic/gin"
)

// registerStudentFeatureRoutes 注册竞赛、大学规划、入党教育和社团生活路由。
func registerStudentFeatureRoutes(secured *gin.RouterGroup, d *deps) {
	competition := secured.Group("/competition")
	competition.GET("/list", auth.RequireCapability(auth.SelfCompetitionRead), d.studentFeaturesH.ListCompetitions)
	competition.GET("/match", auth.RequireCapability(auth.SelfCompetitionRead), d.studentFeaturesH.CompetitionMatch)
	competition.GET("/:id", auth.RequireCapability(auth.SelfCompetitionRead), d.studentFeaturesH.GetCompetition)
	competition.POST("/register", auth.RequireCapability(auth.SelfCompetitionWrite), d.studentFeaturesH.RegisterCompetition)
	competition.GET("/my-registrations", auth.RequireCapability(auth.SelfCompetitionRead), d.studentFeaturesH.GetMyCompetitionRegistrations)
	competition.POST("/submit-work", auth.RequireCapability(auth.SelfCompetitionWrite), d.studentFeaturesH.SubmitWork)
	competition.GET("/stats", auth.RequireCapability(auth.SelfCompetitionRead), d.studentFeaturesH.GetCompetitionStats)
	competition.GET("/admin/list", auth.RequireCapability(auth.SystemSettingsWrite), d.studentFeaturesH.AdminListCompetitions)
	competition.POST("/admin", auth.RequireCapability(auth.SystemSettingsWrite), d.studentFeaturesH.AdminCreateCompetition)
	competition.PUT("/admin/:id", auth.RequireCapability(auth.SystemSettingsWrite), d.studentFeaturesH.AdminUpdateCompetition)
	competition.DELETE("/admin/:id", auth.RequireCapability(auth.SystemSettingsWrite), d.studentFeaturesH.AdminDeleteCompetition)

	plan := secured.Group("/plan")
	plan.GET("/templates", auth.RequireCapability(auth.SelfPlanRead), d.studentFeaturesH.ListPlanTemplates)
	plan.GET("/my-plans", auth.RequireCapability(auth.SelfPlanRead), d.studentFeaturesH.ListMyPlans)
	plan.POST("/create", auth.RequireCapability(auth.SelfPlanWrite), d.studentFeaturesH.CreatePlan)
	plan.PUT("/:id/submit", auth.RequireCapability(auth.SelfPlanWrite), d.studentFeaturesH.SubmitPlan)
	plan.PUT("/:id/review", auth.RequireCapability(auth.CounselorKBWrite), d.studentFeaturesH.ReviewPlan)

	party := secured.Group("/party")
	party.GET("/stages", auth.RequireCapability(auth.SelfPartyRead), d.studentFeaturesH.ListPartyStages)
	party.GET("/my-progress", auth.RequireCapability(auth.SelfPartyRead), d.studentFeaturesH.GetMyPartyProgress)
	party.PUT("/my-progress", auth.RequireCapability(auth.SelfPartyWrite), d.studentFeaturesH.UpdatePartyProgress)
	party.GET("/my-study-records", auth.RequireCapability(auth.SelfPartyRead), d.studentFeaturesH.ListMyStudyRecords)
	party.POST("/study-record", auth.RequireCapability(auth.SelfPartyWrite), d.studentFeaturesH.AddStudyRecord)
	party.GET("/stats", auth.RequireCapability(auth.SelfPartyRead), d.studentFeaturesH.GetPartyStats)

	club := secured.Group("/club")
	club.GET("/list", auth.RequireCapability(auth.SelfClubRead), d.studentFeaturesH.ListClubs)
	club.GET("/:id", auth.RequireCapability(auth.SelfClubRead), d.studentFeaturesH.GetClub)
	club.POST("/join", auth.RequireCapability(auth.SelfClubWrite), d.studentFeaturesH.JoinClub)
	club.GET("/my-clubs", auth.RequireCapability(auth.SelfClubRead), d.studentFeaturesH.GetMyClubs)
	club.GET("/activities", auth.RequireCapability(auth.SelfClubRead), d.studentFeaturesH.ListClubActivities)
	club.POST("/activity/register", auth.RequireCapability(auth.SelfClubWrite), d.studentFeaturesH.RegisterClubActivity)
}
