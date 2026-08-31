package app

// deps 汇总 setupRouter 所需的全部依赖（配置、数据库句柄、Repository 与各 Handler）。
//
// 原 setupRouter 以 约 47+ 个扁平参数透传依赖（app.go 装配行 ↔ routes.go 签名需两处同步），
// 可维护性差。本结构体把依赖收敛为单一对象：app.go 装配层负责构造，routes.go 只接收一个
// deps 指针即可访问全部依赖，消除「新增 handler 需同步两处」的维护隐患。
// 行为与拆分前完全一致：字段只是扁平参数的容器，不改变任何 handler 实例或构造方式。
import (
	"database/sql"

	"github.com/dll/wxx/server/internal/config"
	"github.com/dll/wxx/server/internal/handler"
	"github.com/dll/wxx/server/internal/repository"
)

type deps struct {
	cfg      *config.Config
	db       *sql.DB
	userRepo *repository.UserRepo

	authH               *handler.AuthHandler
	sessionH            *handler.SessionHandler
	chatH               *handler.ChatHandler
	kbH                 *handler.KBHandler
	kgH                 *handler.KnowledgeGovernanceHandler
	voiceH              *handler.VoiceHandler
	emotionH            *handler.EmotionHandler
	agentH              *handler.AgentHandler
	exportH             *handler.ExportHandler
	integrationH        *handler.IntegrationHandler
	recH                *handler.RecommendationHandler
	adminH              *handler.AdminHandler
	feedbackH           *handler.FeedbackHandler
	modelConfigH        *handler.ModelConfigHandler
	tokenStatsH         *handler.TokenStatsHandler
	userActivityStatsH  *handler.UserActivityStatsHandler
	studentH            *handler.StudentHandler
	counselorH          *handler.CounselorHandler
	teacherH            *handler.TeacherHandler
	assistantH          *handler.AssistantHandler
	facilityH           *handler.FacilityHandler
	secretaryH          *handler.SecretaryOutcomeHandler
	unionH              *handler.UnionHandler
	collegeH            *handler.CollegeHandler
	cultureH            *handler.CultureHandler
	schoolAdminH        *handler.SchoolAdminHandler
	sysAdminH           *handler.SysAdminHandler
	govTicketH          *handler.GovTicketHandler
	teacherCourseH      *handler.TeacherCourseHandler
	homeworkH           *handler.HomeworkHandler
	processRecordH      *handler.ProcessRecordHandler
	processH            *handler.ProcessHandler
	forecastH           *handler.ForecastHandler
	graduationH         *handler.GraduationHandler
	studentFeaturesH    *handler.StudentFeaturesHandler
	notificationH       *handler.NotificationHandler
	uploadH             *handler.UploadHandler
	documentH           *handler.DocumentHandler
	educationH          *handler.EducationHandler
	studyPlanH          *handler.StudyPlanHandler
	statsH              *handler.StatsHandler
	userNotificationH   *handler.UserNotificationHandler
	appVersionH         *handler.AppVersionHandler
	campusH             *handler.CampusHandler
	dataImportH         *handler.DataImportHandler
	externalAppH        *handler.ExternalAppHandler
	aiBriefingH         *handler.AIBriefingHandler
	twinPortraitH       *handler.TwinPortraitHandler
	portalCredH         *handler.PortalCredentialHandler
	portalProxyH        *handler.PortalProxyHandler
	vopcH               *handler.VOPCHandler
	feedbackRepairTaskH *handler.FeedbackRepairTaskHandler
}
