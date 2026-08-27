package handler

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dll/wxx/server/internal/config"
	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/service"
	"github.com/dll/wxx/server/internal/testutil"
	"github.com/gin-gonic/gin"
)

// setupAccessRouter 构造一个包含 Get / GetLogs / ServeScreenshot 的最小受保护路由。
func setupAccessRouter(t *testing.T) (*gin.Engine, *config.Config, *repository.UserRepo, *repository.FeedbackRepo, *repository.FeedbackScreenshotRepo) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db := testutil.NewTestDB(t)
	t.Cleanup(func() { db.Close() })

	cfg := &config.Config{JWTSecret: "test-access-secret", JWTExpireHours: 2}
	userRepo := repository.NewUserRepo(db)
	feedbackRepo := repository.NewFeedbackRepo(db)
	screenshotRepo := repository.NewFeedbackScreenshotRepo(db)
	feedbackSvc := service.NewFeedbackService(feedbackRepo, userRepo, screenshotRepo)
	feedbackH := NewFeedbackHandler(feedbackSvc)

	r := gin.New()
	secured := r.Group("/api/v1")
	secured.Use(middleware.JWTAuth(cfg))
	secured.GET("/feedback/:id", feedbackH.Get)
	secured.GET("/feedback/:id/logs", feedbackH.GetLogs)
	secured.GET("/uploads/feedback/:filename", feedbackH.ServeScreenshot)

	return r, cfg, userRepo, feedbackRepo, screenshotRepo
}

// makeUser 创建用户并返回实际数据库分配的 ID（id 列为自增，忽略入参 id）。
func makeUser(t *testing.T, userRepo *repository.UserRepo, username, role string) int64 {
	t.Helper()
	id, err := userRepo.Create(&model.User{Username: username, Role: role, OwnerScope: "school", Status: "active"})
	if err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	return id
}

func createFeedback(t *testing.T, fbRepo *repository.FeedbackRepo, feedbackID string, userID int64, username, screenshotURL string) {
	t.Helper()
	if _, err := fbRepo.Create(&model.Feedback{
		FeedbackID: feedbackID, UserID: userID, Username: username,
		Category: "answer_error", Module: "反馈系统", Content: "隐私测试内容",
		ScreenshotURL: screenshotURL, Status: "pending",
	}); err != nil {
		t.Fatalf("创建反馈失败: %v", err)
	}
}

func mkToken(cfg *config.Config, id int64, username, role string) string {
	tok, _ := middleware.GenerateToken(cfg, &model.User{ID: id, Username: username, Role: role})
	return tok
}

func doGet(t *testing.T, r *gin.Engine, token, path string) int {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Code
}

// TestFeedbackAccess_OwnerCanReadOwn 普通用户可读自己的反馈详情与日志。
func TestFeedbackAccess_OwnerCanReadOwn(t *testing.T) {
	r, cfg, userRepo, fbRepo, _ := setupAccessRouter(t)
	aliceID := makeUser(t, userRepo, "alice", "student")
	createFeedback(t, fbRepo, "fb-own", aliceID, "alice", "")
	token := mkToken(cfg, aliceID, "alice", "student")

	if code := doGet(t, r, token, "/api/v1/feedback/fb-own"); code != http.StatusOK {
		t.Errorf("本人读取自己反馈应 200，得到 %d", code)
	}
	if code := doGet(t, r, token, "/api/v1/feedback/fb-own/logs"); code != http.StatusOK {
		t.Errorf("本人读取自己反馈日志应 200，得到 %d", code)
	}
}

// TestFeedbackAccess_ThirdPartyCannotRead 普通用户不能读他人反馈（应 404 且不泄露存在性）。
func TestFeedbackAccess_ThirdPartyCannotRead(t *testing.T) {
	r, cfg, userRepo, fbRepo, _ := setupAccessRouter(t)
	aliceID := makeUser(t, userRepo, "alice", "student")
	bobID := makeUser(t, userRepo, "bob", "student")
	createFeedback(t, fbRepo, "fb-alice", aliceID, "alice", "")
	token := mkToken(cfg, bobID, "bob", "student")

	if code := doGet(t, r, token, "/api/v1/feedback/fb-alice"); code != http.StatusNotFound {
		t.Errorf("越权第三方读他人反馈应 404，得到 %d", code)
	}
	if code := doGet(t, r, token, "/api/v1/feedback/fb-alice/logs"); code != http.StatusNotFound {
		t.Errorf("越权第三方读他人反馈日志应 404，得到 %d", code)
	}
}

// TestFeedbackAccess_AdminCanReadAll 反馈管理员（student_union，持 union.feedback.list）可读全部。
func TestFeedbackAccess_AdminCanReadAll(t *testing.T) {
	r, cfg, userRepo, fbRepo, _ := setupAccessRouter(t)
	aliceID := makeUser(t, userRepo, "alice", "student")
	unionID := makeUser(t, userRepo, "union1", "student_union")
	createFeedback(t, fbRepo, "fb-alice", aliceID, "alice", "")
	token := mkToken(cfg, unionID, "union1", "student_union")

	if code := doGet(t, r, token, "/api/v1/feedback/fb-alice"); code != http.StatusOK {
		t.Errorf("反馈管理员读他人反馈应 200，得到 %d", code)
	}
	if code := doGet(t, r, token, "/api/v1/feedback/fb-alice/logs"); code != http.StatusOK {
		t.Errorf("反馈管理员读他人反馈日志应 200，得到 %d", code)
	}
}

// TestFeedbackAccess_GetNonexistent_404 不存在反馈应 404。
func TestFeedbackAccess_GetNonexistent_404(t *testing.T) {
	r, cfg, userRepo, _, _ := setupAccessRouter(t)
	aliceID := makeUser(t, userRepo, "alice", "student")
	token := mkToken(cfg, aliceID, "alice", "student")
	if code := doGet(t, r, token, "/api/v1/feedback/fb-nonexistent"); code != http.StatusNotFound {
		t.Errorf("读不存在反馈应 404，得到 %d", code)
	}
}

// TestScreenshotAccess_ThirdPartyForbidden 第三方不能读他人截图（应 403）。
func TestScreenshotAccess_ThirdPartyForbidden(t *testing.T) {
	r, cfg, userRepo, _, ssRepo := setupAccessRouter(t)
	_ = makeUser(t, userRepo, "alice", "student")
	bobID := makeUser(t, userRepo, "bob", "student")

	if err := ssRepo.Save("fb-shot-abc.png", "image/png", base64.StdEncoding.EncodeToString([]byte("png")), "alice", 3); err != nil {
		t.Fatalf("保存截图失败: %v", err)
	}

	token := mkToken(cfg, bobID, "bob", "student")
	if code := doGet(t, r, token, "/api/v1/uploads/feedback/fb-shot-abc.png"); code != http.StatusForbidden {
		t.Errorf("第三方读他人截图应 403，得到 %d", code)
	}
}

// TestScreenshotAccess_UploaderCanRead 截图上传者本人可读。
func TestScreenshotAccess_UploaderCanRead(t *testing.T) {
	r, cfg, userRepo, _, ssRepo := setupAccessRouter(t)
	aliceID := makeUser(t, userRepo, "alice", "student")

	if err := ssRepo.Save("fb-shot-own.png", "image/png", base64.StdEncoding.EncodeToString([]byte("png")), "alice", 3); err != nil {
		t.Fatalf("保存截图失败: %v", err)
	}

	token := mkToken(cfg, aliceID, "alice", "student")
	if code := doGet(t, r, token, "/api/v1/uploads/feedback/fb-shot-own.png"); code != http.StatusOK {
		t.Errorf("截图上传者本人应 200，得到 %d", code)
	}
}

// TestScreenshotAccess_ReferencerCanRead 引用该截图的反馈提交者本人可读（即使非上传者）。
func TestScreenshotAccess_ReferencerCanRead(t *testing.T) {
	r, cfg, userRepo, fbRepo, ssRepo := setupAccessRouter(t)
	aliceID := makeUser(t, userRepo, "alice", "student")

	if err := ssRepo.Save("fb-shot-ref.png", "image/png", base64.StdEncoding.EncodeToString([]byte("png")), "admin", 3); err != nil {
		t.Fatalf("保存截图失败: %v", err)
	}
	createFeedback(t, fbRepo, "fb-ref", aliceID, "alice", "/uploads/feedback/fb-shot-ref.png")

	token := mkToken(cfg, aliceID, "alice", "student")
	if code := doGet(t, r, token, "/api/v1/uploads/feedback/fb-shot-ref.png"); code != http.StatusOK {
		t.Errorf("引用截图的反馈提交者应 200，得到 %d", code)
	}
}

// TestScreenshotAccess_AdminCanRead 反馈管理员可读任意截图。
func TestScreenshotAccess_AdminCanRead(t *testing.T) {
	r, cfg, userRepo, _, ssRepo := setupAccessRouter(t)
	unionID := makeUser(t, userRepo, "union1", "student_union")

	if err := ssRepo.Save("fb-shot-x.png", "image/png", base64.StdEncoding.EncodeToString([]byte("png")), "alice", 3); err != nil {
		t.Fatalf("保存截图失败: %v", err)
	}

	token := mkToken(cfg, unionID, "union1", "student_union")
	if code := doGet(t, r, token, "/api/v1/uploads/feedback/fb-shot-x.png"); code != http.StatusOK {
		t.Errorf("反馈管理员读任意截图应 200，得到 %d", code)
	}
}

// TestFeedbackAccess_Unauthenticated_401 未认证访问应 401。
func TestFeedbackAccess_Unauthenticated_401(t *testing.T) {
	r, _, _, _, _ := setupAccessRouter(t)
	if code := doGet(t, r, "", "/api/v1/feedback/fb-own"); code != http.StatusUnauthorized {
		t.Errorf("未认证访问应 401，得到 %d", code)
	}
}
