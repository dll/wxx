package repository

import (
	"database/sql"

	"github.com/dll/wxx/server/internal/model"
)

// ExportLogRepo 导出审计记录数据访问。
type ExportLogRepo struct {
	db *sql.DB
}

func NewExportLogRepo(db *sql.DB) *ExportLogRepo {
	return &ExportLogRepo{db: db}
}

func (r *ExportLogRepo) Insert(log *model.ExportLog) error {
	_, err := r.db.Exec(
		`INSERT INTO export_logs (user_id, role, format, answer_id, has_sensitive, trace_id)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		log.UserID, log.Role, log.Format, log.AnswerID, log.HasSensitive, log.TraceID,
	)
	return err
}
