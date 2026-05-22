package repository

import (
	"database/sql"
)

// FeedbackScreenshotRepo 反馈截图二进制数据（base64 存 SQLite，跨实例可读）
type FeedbackScreenshotRepo struct {
	db *sql.DB
}

func NewFeedbackScreenshotRepo(db *sql.DB) *FeedbackScreenshotRepo {
	return &FeedbackScreenshotRepo{db: db}
}

// Save 保存截图数据
func (r *FeedbackScreenshotRepo) Save(filename, mime, dataBase64, uploader string, sizeBytes int64) error {
	_, err := r.db.Exec(
		`INSERT INTO feedback_screenshots (filename, mime_type, size_bytes, data_base64, uploaded_by)
		 VALUES (?, ?, ?, ?, ?)`,
		filename, mime, sizeBytes, dataBase64, uploader,
	)
	return err
}

// GetByFilename 按文件名取出（用于路由直接返回字节）
func (r *FeedbackScreenshotRepo) GetByFilename(filename string) (dataBase64, mime string, err error) {
	err = r.db.QueryRow(
		`SELECT data_base64, mime_type FROM feedback_screenshots WHERE filename = ?`,
		filename,
	).Scan(&dataBase64, &mime)
	if err == sql.ErrNoRows {
		return "", "", nil
	}
	return dataBase64, mime, err
}
