package service_test

import (
	"context"
	"testing"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
	"github.com/dll/wxx/server/internal/service"
	"github.com/dll/wxx/server/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestRetentionServiceDeletesExpiredData(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()

	userRepo := repository.NewUserRepo(db)
	uid, err := userRepo.Create(&model.User{
		Username: "retention_user", DisplayName: "Retention", Role: "student",
		OwnerScope: "college", OwnerID: "default", Status: "active",
	})
	require.NoError(t, err)

	sessionRepo := repository.NewSessionRepo(db)
	s := &model.Session{SessionID: "old-session", UserID: uid, Title: "old"}
	require.NoError(t, sessionRepo.Create(s))
	_, err = db.Exec(`UPDATE sessions SET updated_at = datetime('now', '-400 days') WHERE session_id = ?`, s.SessionID)
	require.NoError(t, err)

	msgRepo := repository.NewMessageRepo(db)
	require.NoError(t, msgRepo.Create(&model.Message{SessionID: s.SessionID, Role: "user", Content: "old message"}))
	_, err = db.Exec(`UPDATE messages SET created_at = datetime('now', '-400 days') WHERE session_id = ?`, s.SessionID)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO audit_logs (user_id, username, role, action, resource, detail, trace_id, ip, duration_ms, result_code, created_at)
		VALUES (?, 'retention_user', 'student', 'GET', '/test', '/test', 'trace', '127.0.0.1', 1, 200, datetime('now', '-200 days'))`, uid)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO emotion_logs (alert_id, user_id, username, session_id, message_text, score, risk_level, analysis_json, notified, status, created_at)
		VALUES ('alert-old', ?, 'retention_user', ?, 'old', 0, 'low', '{}', 0, 'pending', datetime('now', '-400 days'))`, uid, s.SessionID)
	require.NoError(t, err)

	exportRepo := repository.NewExportLogRepo(db)
	require.NoError(t, exportRepo.Insert(&model.ExportLog{UserID: uid, Role: "student", Format: "pdf", AnswerID: "a1", TraceID: "t1"}))
	_, err = db.Exec(`UPDATE export_logs SET created_at = datetime('now', '-200 days') WHERE answer_id = 'a1'`)
	require.NoError(t, err)

	retention := service.NewRetentionService(db)
	result, err := retention.RunOnce(context.Background(), 180, 365, 365, 180)
	require.NoError(t, err)
	require.GreaterOrEqual(t, result.SessionsDeleted, int64(1))
	require.GreaterOrEqual(t, result.AuditLogsDeleted, int64(1))
	require.GreaterOrEqual(t, result.EmotionLogsDeleted, int64(1))
	require.GreaterOrEqual(t, result.ExportLogsDeleted, int64(1))

	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&count))
	require.Equal(t, 0, count)
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM messages`).Scan(&count))
	require.Equal(t, 0, count)
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM audit_logs`).Scan(&count))
	require.Equal(t, 0, count)
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM emotion_logs`).Scan(&count))
	require.Equal(t, 0, count)
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM export_logs`).Scan(&count))
	require.Equal(t, 0, count)
}

func TestRetentionServiceKeepsRecentData(t *testing.T) {
	db := testutil.NewTestDB(t)
	defer db.Close()
	_, err := db.Exec(`INSERT INTO audit_logs (username, action, resource, detail, trace_id, ip, duration_ms, result_code, created_at)
		VALUES ('recent_user', 'GET', '/test', '/test', 'trace', '127.0.0.1', 1, 200, datetime('now'))`)
	require.NoError(t, err)
	retention := service.NewRetentionService(db)
	_, err = retention.RunOnce(context.Background(), 180, 365, 365, 180)
	require.NoError(t, err)
	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM audit_logs`).Scan(&count))
	require.Equal(t, 1, count)
}
