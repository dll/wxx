// Package repository 健康活动仓库（P4-d：从 education_health_activity_handler 下沉的 10 处裸 SQL）。
package repository

import (
	"database/sql"

	dbutil "github.com/dll/wxx/server/internal/db"
	"github.com/dll/wxx/server/internal/model"
)

// HealthActivityRepo 健康活动数据访问层。
type HealthActivityRepo struct {
	db *sql.DB
}

// NewHealthActivityRepo 创建健康活动仓库。
func NewHealthActivityRepo(db *sql.DB) *HealthActivityRepo {
	return &HealthActivityRepo{db: db}
}

// ListActivities 活动列表（含关注/报名统计与当前用户状态）。
func (r *HealthActivityRepo) ListActivities(userID int64, category string) ([]*model.HealthActivityItem, error) {
	where := "WHERE a.status = 'active'"
	var args []interface{}
	if category != "" {
		where += " AND a.category = ?"
		args = append(args, category)
	}

	rows, err := r.db.Query(
		`SELECT a.activity_id, a.title, a.category, a.description, a.start_at, a.end_at,
		        a.venue, a.organizer, a.capacity, a.signup_deadline, a.status, a.creator_role,
		        (SELECT COUNT(*) FROM health_activity_favorites f WHERE f.activity_id = a.activity_id) AS fav,
		        (SELECT COUNT(*) FROM health_activity_signups s WHERE s.activity_id = a.activity_id AND s.status='registered') AS sg,
		        EXISTS(SELECT 1 FROM health_activity_favorites f2 WHERE f2.activity_id = a.activity_id AND f2.user_id = ?) AS is_fav,
		        EXISTS(SELECT 1 FROM health_activity_signups s2 WHERE s2.activity_id = a.activity_id AND s2.user_id = ? AND s2.status='registered') AS is_sg
		 FROM health_activities a `+where+` ORDER BY a.start_at DESC LIMIT 200`,
		append(args, userID, userID)...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*model.HealthActivityItem, 0)
	for rows.Next() {
		item := &model.HealthActivityItem{}
		var isFav, isSg int
		if err := rows.Scan(&item.ActivityID, &item.Title, &item.Category, &item.Description,
			&item.StartAt, &item.EndAt, &item.Venue, &item.Organizer, &item.Capacity,
			&item.SignupDeadline, &item.Status, &item.CreatorRole,
			&item.FavoriteCount, &item.SignupCount, &isFav, &isSg); err != nil {
			continue
		}
		item.IsFavorite = isFav == 1
		item.IsSignup = isSg == 1
		list = append(list, item)
	}
	return list, rows.Err()
}

// CreateActivity 发布活动。
func (r *HealthActivityRepo) CreateActivity(activityID, title, category, description, startAt, endAt, venue, organizer string, capacity int, signupDeadline string, creatorID int64, creatorRole string) error {
	_, err := r.db.Exec(
		`INSERT INTO health_activities
		   (activity_id, title, category, description, start_at, end_at, venue, organizer, capacity, signup_deadline, status, creator_id, creator_role)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'active', ?, ?)`,
		activityID, title, category, description, startAt, endAt,
		venue, organizer, capacity, signupDeadline, creatorID, creatorRole,
	)
	return err
}

// AddFavorite 关注活动（幂等，方言适配）。
func (r *HealthActivityRepo) AddFavorite(userID int64, activityID string) error {
	_, err := r.db.Exec(
		dbutil.InsertIgnore(dbutil.DriverOf(r.db))+` health_activity_favorites (user_id, activity_id) VALUES (?, ?)`,
		userID, activityID,
	)
	return err
}

// RemoveFavorite 取消关注。
func (r *HealthActivityRepo) RemoveFavorite(userID int64, activityID string) error {
	_, err := r.db.Exec(`DELETE FROM health_activity_favorites WHERE user_id = ? AND activity_id = ?`, userID, activityID)
	return err
}

// UpdateStatus 更新活动状态，返回受影响行数。
func (r *HealthActivityRepo) UpdateStatus(activityID, status string) (int64, error) {
	res, err := r.db.Exec(`UPDATE health_activities SET status=? WHERE activity_id=?`, status, activityID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// MarkAttended 活动签到（标记到场），返回受影响行数。
func (r *HealthActivityRepo) MarkAttended(activityID string, uid string, attended int) (int64, error) {
	res, err := r.db.Exec(
		`UPDATE health_activity_signups SET attended=? WHERE activity_id=? AND user_id=? AND status='registered'`,
		attended, activityID, uid)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// ListReviewRows 复盘原始行（按报名数降序）。
func (r *HealthActivityRepo) ListReviewRows() ([]*model.ActivityReviewRow, error) {
	rows, err := r.db.Query(`
		SELECT a.activity_id, a.title, a.category, a.venue, a.organizer, a.status,
		       (SELECT COUNT(*) FROM health_activity_signups s WHERE s.activity_id=a.activity_id AND s.status='registered') AS sg,
		       (SELECT COUNT(*) FROM health_activity_signups s WHERE s.activity_id=a.activity_id AND s.status='registered' AND s.attended=1) AS at
		FROM health_activities a
		ORDER BY sg DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]*model.ActivityReviewRow, 0)
	for rows.Next() {
		row := &model.ActivityReviewRow{}
		var sg, at int
		if err := rows.Scan(&row.ActivityID, &row.Title, &row.Category, &row.Venue, &row.Organizer, &row.Status, &sg, &at); err != nil {
			continue
		}
		row.SignupCount = sg
		row.AttendCount = at
		items = append(items, row)
	}
	return items, rows.Err()
}

// ListSignups 活动报名/到场名单。
func (r *HealthActivityRepo) ListSignups(activityID string) ([]*model.ActivitySignup, error) {
	rows, err := r.db.Query(`
		SELECT s.user_id, u.username, u.display_name, s.attended, s.created_at
		FROM health_activity_signups s
		LEFT JOIN users u ON u.id = s.user_id
		WHERE s.activity_id=? AND s.status='registered'
		ORDER BY s.attended ASC, s.created_at ASC`, activityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*model.ActivitySignup, 0)
	for rows.Next() {
		item := &model.ActivitySignup{}
		var uid int
		var uname, disp string
		var att int
		if err := rows.Scan(&uid, &uname, &disp, &att, &item.CreatedAt); err != nil {
			continue
		}
		item.UserID = uid
		item.Username = uname
		item.Name = disp
		if item.Name == "" {
			item.Name = uname
		}
		item.Attended = att == 1
		list = append(list, item)
	}
	return list, rows.Err()
}

// AddSignup 报名（幂等，方言适配）。
func (r *HealthActivityRepo) AddSignup(userID int64, activityID string) error {
	_, err := r.db.Exec(
		dbutil.InsertIgnore(dbutil.DriverOf(r.db))+` health_activity_signups (user_id, activity_id, status) VALUES (?, ?, 'registered')`,
		userID, activityID,
	)
	return err
}

// CancelSignup 取消报名。
func (r *HealthActivityRepo) CancelSignup(userID int64, activityID string) error {
	_, err := r.db.Exec(`UPDATE health_activity_signups SET status = 'cancelled' WHERE user_id = ? AND activity_id = ?`, userID, activityID)
	return err
}
