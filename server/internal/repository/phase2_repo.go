package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

// Phase2Repo 阶段二真实数据访问层（积分 / 问答广场 / 谈心记录）
type Phase2Repo struct {
	db *sql.DB
}

// NewPhase2Repo 创建阶段二仓库
func NewPhase2Repo(db *sql.DB) *Phase2Repo {
	return &Phase2Repo{db: db}
}

// ── 积分 ──

// PointRecord 积分明细
type PointRecord struct {
	ID        int64  `json:"id"`
	Points    int    `json:"points"`
	Reason    string `json:"reason"`
	Source    string `json:"source"`
	CreatedAt string `json:"created_at"`
}

// AddPoints 增加积分记录（不返回错误时积分已落库）
func (r *Phase2Repo) AddPoints(userID int64, points int, reason, source string) error {
	_, err := r.db.Exec(
		`INSERT INTO student_points (user_id, points, reason, source) VALUES (?, ?, ?, ?)`,
		userID, points, reason, source)
	if err != nil {
		return fmt.Errorf("积分写入失败: %w", err)
	}
	return nil
}

// GetPointsSummary 查询积分汇总（总分 + 明细条数 + 最近明细）
func (r *Phase2Repo) GetPointsSummary(userID int64, limit int) (total int, count int, recent []*PointRecord, err error) {
	if err = r.db.QueryRow(`SELECT IFNULL(SUM(points),0), COUNT(*) FROM student_points WHERE user_id = ?`, userID).Scan(&total, &count); err != nil {
		return 0, 0, nil, err
	}
	rows, err := r.db.Query(
		`SELECT id, points, reason, source, created_at FROM student_points
		 WHERE user_id = ? ORDER BY id DESC LIMIT ?`, userID, limit)
	if err != nil {
		return total, count, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var p PointRecord
		if err := rows.Scan(&p.ID, &p.Points, &p.Reason, &p.Source, &p.CreatedAt); err != nil {
			return total, count, recent, err
		}
		recent = append(recent, &p)
	}
	return total, count, recent, rows.Err()
}

// CountSource 统计某来源的积分记录条数（如打卡天数）
func (r *Phase2Repo) CountSource(userID int64, source string) (int, error) {
	var n int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM student_points WHERE user_id = ? AND source = ?`, userID, source).Scan(&n)
	return n, err
}

// ── 问答广场 ──

// QAPost 帖子
type QAPost struct {
	ID        int64  `json:"id"`
	UserID    int64  `json:"user_id"`
	Author    string `json:"author"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	Category  string `json:"category"`
	Answers   int    `json:"answers"`
	Views     int    `json:"views"`
	CreatedAt string `json:"created_at"`
}

// CreateQAPost 创建帖子
func (r *Phase2Repo) CreateQAPost(userID int64, title, content, category string) (int64, error) {
	res, err := r.db.Exec(
		`INSERT INTO qa_posts (user_id, title, content, category) VALUES (?, ?, ?, ?)`,
		userID, title, content, category)
	if err != nil {
		return 0, fmt.Errorf("发布问题失败: %w", err)
	}
	return res.LastInsertId()
}

// ListQAPosts 列出最近帖子（含作者名）
func (r *Phase2Repo) ListQAPosts(limit int) ([]*QAPost, error) {
	rows, err := r.db.Query(
		`SELECT p.id, p.user_id, COALESCE(u.display_name,'同学'), p.title, p.content, p.category, p.answers, p.views, p.created_at
		 FROM qa_posts p LEFT JOIN users u ON p.user_id = u.id
		 WHERE p.status = 'published'
		 ORDER BY p.created_at DESC, p.id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var posts []*QAPost
	for rows.Next() {
		var p QAPost
		if err := rows.Scan(&p.ID, &p.UserID, &p.Author, &p.Title, &p.Content, &p.Category, &p.Answers, &p.Views, &p.CreatedAt); err != nil {
			return nil, err
		}
		posts = append(posts, &p)
	}
	return posts, rows.Err()
}

// ListQAPostsByUser 列出某用户的帖子（含未发布/隐藏，供"我的提问"）
func (r *Phase2Repo) ListQAPostsByUser(userID int64, limit int) ([]*QAPost, error) {
	rows, err := r.db.Query(
		`SELECT p.id, p.user_id, COALESCE(u.display_name,'同学'), p.title, p.content, p.category, p.answers, p.views, p.created_at
		 FROM qa_posts p LEFT JOIN users u ON p.user_id = u.id
		 WHERE p.user_id = ?
		 ORDER BY p.created_at DESC, p.id DESC LIMIT ?`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var posts []*QAPost
	for rows.Next() {
		var p QAPost
		if err := rows.Scan(&p.ID, &p.UserID, &p.Author, &p.Title, &p.Content, &p.Category, &p.Answers, &p.Views, &p.CreatedAt); err != nil {
			return nil, err
		}
		posts = append(posts, &p)
	}
	return posts, rows.Err()
}

// GetQAPost 查询单个帖子
func (r *Phase2Repo) GetQAPost(postID int64) (*QAPost, error) {
	p := &QAPost{}
	err := r.db.QueryRow(
		`SELECT p.id, p.user_id, COALESCE(u.display_name,'同学'), p.title, p.content, p.category, p.answers, p.views, p.created_at
		 FROM qa_posts p LEFT JOIN users u ON p.user_id = u.id WHERE p.id = ?`, postID).
		Scan(&p.ID, &p.UserID, &p.Author, &p.Title, &p.Content, &p.Category, &p.Answers, &p.Views, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// IncrementPostViews 增加浏览数
func (r *Phase2Repo) IncrementPostViews(postID int64) error {
	_, err := r.db.Exec(`UPDATE qa_posts SET views = views + 1 WHERE id = ?`, postID)
	return err
}

// CreateQAAnswer 提交回答并自增帖子回答数
func (r *Phase2Repo) CreateQAAnswer(postID, userID int64, content string) (int64, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`INSERT INTO qa_answers (post_id, user_id, content) VALUES (?, ?, ?)`, postID, userID, content)
	if err != nil {
		return 0, fmt.Errorf("提交回答失败: %w", err)
	}
	if _, err := tx.Exec(`UPDATE qa_posts SET answers = answers + 1 WHERE id = ?`, postID); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// ListQAAnswers 列出某帖子的回答（含作者名）
func (r *Phase2Repo) ListQAAnswers(postID int64) ([]map[string]interface{}, error) {
	rows, err := r.db.Query(
		`SELECT a.id, a.content, COALESCE(u.display_name,'同学'), a.adopted, a.created_at
		 FROM qa_answers a LEFT JOIN users u ON a.user_id = u.id
		 WHERE a.post_id = ? ORDER BY a.id ASC`, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []map[string]interface{}
	for rows.Next() {
		var id int64
		var content, author, createdAt string
		var adopted int
		if err := rows.Scan(&id, &content, &author, &adopted, &createdAt); err != nil {
			return nil, err
		}
		list = append(list, map[string]interface{}{
			"id": id, "content": content, "author": author, "adopted": adopted, "created_at": createdAt,
		})
	}
	return list, rows.Err()
}

// ── 谈心谈话 ──

// TalkRecord 谈心记录
type TalkRecord struct {
	ID          int64    `json:"id"`
	StudentName string   `json:"student_name"`
	Topic       string   `json:"topic"`
	Emotion     string   `json:"emotion"`
	Summary     string   `json:"summary"`
	FollowUps   []string `json:"follow_ups"`
	Status      string   `json:"status"`
	CreatedAt   string   `json:"created_at"`
}

// CreateTalkRecord 保存谈心记录
func (r *Phase2Repo) CreateTalkRecord(counselorID, studentID int64, studentName, topic, emotion, content, summary string, followUpsJSON string) (int64, error) {
	res, err := r.db.Exec(
		`INSERT INTO talk_records (counselor_id, student_id, student_name, topic, emotion, content, summary, follow_ups, status)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'following')`,
		counselorID, studentID, studentName, topic, emotion, content, summary, followUpsJSON)
	if err != nil {
		return 0, fmt.Errorf("谈心记录保存失败: %w", err)
	}
	return res.LastInsertId()
}

// ListTalkRecords 列出某辅导员的谈心记录
func (r *Phase2Repo) ListTalkRecords(counselorID int64, limit int) ([]map[string]interface{}, error) {
	rows, err := r.db.Query(
		`SELECT id, student_name, topic, emotion, summary, follow_ups, status, created_at
		 FROM talk_records WHERE counselor_id = ? ORDER BY id DESC LIMIT ?`, counselorID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []map[string]interface{}
	for rows.Next() {
		var id int64
		var studentName, topic, emotion, summary, followUps, status, createdAt string
		if err := rows.Scan(&id, &studentName, &topic, &emotion, &summary, &followUps, &status, &createdAt); err != nil {
			return nil, err
		}
		var fu []string
		_ = json.Unmarshal([]byte(followUps), &fu)
		list = append(list, map[string]interface{}{
			"id": id, "student_name": studentName, "topic": topic, "emotion": emotion,
			"summary": summary, "follow_ups": fu, "status": status, "created_at": createdAt,
		})
	}
	return list, rows.Err()
}
