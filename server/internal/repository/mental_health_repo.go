// Package repository 心理健康教育仓库（P4-d：从 education_mental_health_handler 下沉的 14 处裸 SQL）。
package repository

import (
	"database/sql"
	"errors"
	"log"

	"github.com/dll/wxx/server/internal/model"
)

// ErrPsychNotFound 心理域资源不存在（量表/文章）。
var ErrPsychNotFound = errors.New("心理资源不存在")

// MentalHealthRepo 心理健康数据访问层。
type MentalHealthRepo struct {
	db *sql.DB
}

// NewMentalHealthRepo 创建心理健康仓库。
func NewMentalHealthRepo(db *sql.DB) *MentalHealthRepo {
	return &MentalHealthRepo{db: db}
}

// ── 量表 ──

// ListPsychScales 启用状态量表列表（可按分类过滤）。
func (r *MentalHealthRepo) ListPsychScales(category string) ([]*model.PsychScale, error) {
	where := []string{"status = 'active'"}
	args := []interface{}{}
	if category != "" {
		where = append(where, "category = ?")
		args = append(args, category)
	}

	rows, err := r.db.Query(
		"SELECT id, scale_id, name, abbreviation, category, description, question_count, "+
			"estimated_minutes, is_crisis, created_at "+
			"FROM psych_scales "+buildWhereClause(where)+" ORDER BY is_crisis DESC, id ASC",
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*model.PsychScale, 0)
	for rows.Next() {
		item := &model.PsychScale{}
		if err := rows.Scan(&item.ID, &item.ScaleID, &item.Name, &item.Abbreviation,
			&item.Category, &item.Description, &item.QuestionCount, &item.EstimatedMinutes,
			&item.IsCrisis, &item.CreatedAt); err != nil {
			continue
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

// GetPsychScaleDetail 活动状态量表详情（无记录返回 ErrPsychNotFound）。
func (r *MentalHealthRepo) GetPsychScaleDetail(scaleID string) (*model.PsychScaleDetail, error) {
	detail := &model.PsychScaleDetail{}
	err := r.db.QueryRow(
		"SELECT id, scale_id, name, abbreviation, category, description, question_count, "+
			"estimated_minutes, scoring_method, interpretation, questions_json, is_crisis, status, created_at "+
			"FROM psych_scales WHERE scale_id = ? AND status = 'active'", scaleID,
	).Scan(&detail.ID, &detail.ScaleID, &detail.Name, &detail.Abbreviation,
		&detail.Category, &detail.Description, &detail.QuestionCount, &detail.EstimatedMinutes,
		&detail.ScoringMethod, &detail.Interpretation, &detail.QuestionsJSON,
		&detail.IsCrisis, &detail.Status, &detail.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrPsychNotFound
	}
	if err != nil {
		return nil, err
	}
	return detail, nil
}

// GetScaleForScoring 取量表计分所需字段（name/questions/interpretation/scoring_method）。
func (r *MentalHealthRepo) GetScaleForScoring(scaleID string) (name, questionsJSON, interpretationJSON, scoringMethod string, err error) {
	err = r.db.QueryRow(
		"SELECT name, questions_json, interpretation, scoring_method FROM psych_scales WHERE scale_id = ? AND status = 'active'",
		scaleID,
	).Scan(&name, &questionsJSON, &interpretationJSON, &scoringMethod)
	if err == sql.ErrNoRows {
		return "", "", "", "", ErrPsychNotFound
	}
	return
}

// SaveAssessmentRecord 保存测评记录，返回完成时间戳（保持原 API 契约）。
func (r *MentalHealthRepo) SaveAssessmentRecord(userID int64, recordID, scaleID, scaleName, answersJSON, scoresJSON string, totalScore float64, level, resultSummary, suggestion string) (string, error) {
	completedAt := nowString()
	_, err := r.db.Exec(
		"INSERT INTO psych_assessment_records (user_id, record_id, scale_id, scale_name, answers_json, scores_json, total_score, level, result_summary, suggestion, completed_at, created_at) "+
			"VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		userID, recordID, scaleID, scaleName, answersJSON, scoresJSON,
		totalScore, level, resultSummary, suggestion, completedAt, completedAt,
	)
	if err != nil {
		return "", err
	}
	return completedAt, nil
}

// ListMyAssessments 用户测评记录（按完成时间倒序）。
func (r *MentalHealthRepo) ListMyAssessments(userID int64) ([]*model.AssessmentRecord, error) {
	rows, err := r.db.Query(
		"SELECT id, record_id, scale_id, scale_name, total_score, level, result_summary, suggestion, completed_at, created_at "+
			"FROM psych_assessment_records WHERE user_id = ? ORDER BY completed_at DESC, id DESC",
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*model.AssessmentRecord, 0)
	for rows.Next() {
		item := &model.AssessmentRecord{}
		if err := rows.Scan(&item.ID, &item.RecordID, &item.ScaleID, &item.ScaleName,
			&item.TotalScore, &item.Level, &item.ResultSummary, &item.Suggestion,
			&item.CompletedAt, &item.CreatedAt); err != nil {
			continue
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

// ── 科普文章 ──

// ListPsychArticles 分页文章列表（含总数）。
func (r *MentalHealthRepo) ListPsychArticles(category string, page, pageSize int) ([]*model.PsychArticle, int, error) {
	where := []string{"status = 'active'"}
	args := []interface{}{}
	if category != "" {
		where = append(where, "category = ?")
		args = append(args, category)
	}
	whereSQL := buildWhereClause(where)

	var total int
	if err := r.db.QueryRow("SELECT COUNT(*) FROM psych_articles "+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(
		"SELECT id, article_id, title, category, summary, cover_image, author, read_count, is_crisis, tags, created_at "+
			"FROM psych_articles "+whereSQL+" ORDER BY is_crisis DESC, created_at DESC, id DESC LIMIT ? OFFSET ?",
		append(args, pageSize, pageOffset(page, pageSize))...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	list := make([]*model.PsychArticle, 0)
	for rows.Next() {
		item := &model.PsychArticle{}
		if err := rows.Scan(&item.ID, &item.ArticleID, &item.Title, &item.Category,
			&item.Summary, &item.CoverImage, &item.Author, &item.ReadCount,
			&item.IsCrisis, &item.Tags, &item.CreatedAt); err != nil {
			continue
		}
		list = append(list, item)
	}
	return list, total, rows.Err()
}

// GetPsychArticleDetail 活动状态文章详情（无记录返回 ErrPsychNotFound）。
func (r *MentalHealthRepo) GetPsychArticleDetail(articleID string) (*model.PsychArticleDetail, error) {
	detail := &model.PsychArticleDetail{}
	err := r.db.QueryRow(
		"SELECT id, article_id, title, category, summary, content, cover_image, author, "+
			"read_count, is_crisis, tags, status, created_at, updated_at "+
			"FROM psych_articles WHERE article_id = ? AND status = 'active'", articleID,
	).Scan(&detail.ID, &detail.ArticleID, &detail.Title, &detail.Category,
		&detail.Summary, &detail.Content, &detail.CoverImage, &detail.Author,
		&detail.ReadCount, &detail.IsCrisis, &detail.Tags, &detail.Status,
		&detail.CreatedAt, &detail.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrPsychNotFound
	}
	if err != nil {
		return nil, err
	}
	return detail, nil
}

// IncrementArticleReadCount 阅读数自增（best-effort，失败仅日志）。
func (r *MentalHealthRepo) IncrementArticleReadCount(id int64) {
	if _, err := r.db.Exec("UPDATE psych_articles SET read_count = read_count + 1 WHERE id = ?", id); err != nil {
		log.Printf("[WARN] 文章阅读数自增失败 id=%d: %v", id, err)
	}
}

// ── 危机热线 ──

// ListCrisisHotlines 启用状态热线（按等级升序）。
func (r *MentalHealthRepo) ListCrisisHotlines() ([]*model.CrisisHotline, error) {
	rows, err := r.db.Query(
		"SELECT id, hotline_id, name, phone, service_time, description, level, created_at " +
			"FROM crisis_hotlines WHERE status = 'active' ORDER BY level ASC, id ASC",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*model.CrisisHotline, 0)
	for rows.Next() {
		item := &model.CrisisHotline{}
		if err := rows.Scan(&item.ID, &item.HotlineID, &item.Name, &item.Phone,
			&item.ServiceTime, &item.Description, &item.Level, &item.CreatedAt); err != nil {
			continue
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

// ── 情绪日记 ──

// ListMoodDiary 日期范围查询用户情绪日记（起止均可空）。
func (r *MentalHealthRepo) ListMoodDiary(userID int64, startDate, endDate string) ([]*model.MoodDiaryItem, error) {
	where := []string{"user_id = ?"}
	args := []interface{}{userID}
	if startDate != "" {
		where = append(where, "date >= ?")
		args = append(args, startDate)
	}
	if endDate != "" {
		where = append(where, "date <= ?")
		args = append(args, endDate)
	}

	rows, err := r.db.Query(
		"SELECT id, diary_id, date, mood_score, mood_tags, events, diary_content, sleep_hours, exercise_minutes, social_level, created_at, updated_at "+
			"FROM mood_diary "+buildWhereClause(where)+" ORDER BY date DESC, id DESC",
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*model.MoodDiaryItem, 0)
	for rows.Next() {
		item := &model.MoodDiaryItem{}
		if err := rows.Scan(&item.ID, &item.DiaryID, &item.Date, &item.MoodScore,
			&item.MoodTags, &item.Events, &item.DiaryContent, &item.SleepHours,
			&item.ExerciseMinutes, &item.SocialLevel, &item.CreatedAt, &item.UpdatedAt); err != nil {
			continue
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

// UpsertMoodDiary 按用户+日期幂等写情绪日记（存在则更新，返回 updated 标记）。
func (r *MentalHealthRepo) UpsertMoodDiary(userID int64, diaryID, date string, moodScore int, moodTags, events, diaryContent string, sleepHours float64, exerciseMinutes, socialLevel int) (updated bool, err error) {
	now := nowString()
	var existingID int64
	err = r.db.QueryRow("SELECT id FROM mood_diary WHERE user_id = ? AND date = ?", userID, date).Scan(&existingID)
	if err == nil {
		if _, err = r.db.Exec(
			"UPDATE mood_diary SET mood_score = ?, mood_tags = ?, events = ?, diary_content = ?, sleep_hours = ?, exercise_minutes = ?, social_level = ?, updated_at = ? WHERE id = ?",
			moodScore, moodTags, events, diaryContent, sleepHours, exerciseMinutes, socialLevel, now, existingID,
		); err != nil {
			return true, err
		}
		return true, nil
	}
	if err != sql.ErrNoRows {
		return false, err
	}

	_, err = r.db.Exec(
		"INSERT INTO mood_diary (user_id, diary_id, date, mood_score, mood_tags, events, diary_content, sleep_hours, exercise_minutes, social_level, created_at, updated_at) "+
			"VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		userID, diaryID, date, moodScore, moodTags, events, diaryContent, sleepHours, exerciseMinutes, socialLevel, now, now,
	)
	return false, err
}
