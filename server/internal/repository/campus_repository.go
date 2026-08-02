package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/dll/wxx/server/internal/model"
)

// CampusRepository 校园报到步骤数据访问层
type CampusRepository struct {
	db *sql.DB
}

// NewCampusRepository 构造函数
func NewCampusRepository(db *sql.DB) *CampusRepository {
	return &CampusRepository{db: db}
}

// ListPublished 返回指定校区已发布的步骤（按 step_order 升序）
func (r *CampusRepository) ListPublished(campusID string) ([]model.CampusStep, error) {
	return r.list("WHERE campus_id=? AND status='published' ORDER BY step_order ASC", campusID)
}

// ListAll 返回指定校区全部步骤（管理端用）
func (r *CampusRepository) ListAll(campusID string) ([]model.CampusStep, error) {
	return r.list("WHERE campus_id=? ORDER BY step_order ASC", campusID)
}

// list 通用查询
func (r *CampusRepository) list(where string, args ...interface{}) ([]model.CampusStep, error) {
	q := `SELECT id,campus_id,step_order,title,location,lat,lng,duration,
	      task,materials,contact,note,icon_name,status,
	      COALESCE(created_by,''),COALESCE(reviewed_by,''),
	      COALESCE(published_at,''),created_at,updated_at
	      FROM campus_checkin_steps ` + where
	rows, err := r.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var steps []model.CampusStep
	for rows.Next() {
		var s model.CampusStep
		var cb, rb string
		if err := rows.Scan(&s.ID, &s.CampusID, &s.StepOrder, &s.Title,
			&s.Location, &s.Lat, &s.Lng, &s.Duration, &s.Task,
			&s.Materials, &s.Contact, &s.Note, &s.IconName, &s.Status,
			&cb, &rb, &s.PublishedAt, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		steps = append(steps, s)
	}
	return steps, rows.Err()
}

// GetByID 按 ID 查单条
func (r *CampusRepository) GetByID(id int64) (*model.CampusStep, error) {
	list, err := r.list("WHERE id=?", id)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, fmt.Errorf("步骤 %d 不存在", id)
	}
	return &list[0], nil
}

// Create 新建步骤（初始 draft 状态）
func (r *CampusRepository) Create(req *model.CampusStepRequest, createdBy int64) (int64, error) {
	res, err := r.db.Exec(
		`INSERT INTO campus_checkin_steps
		(campus_id,step_order,title,location,lat,lng,duration,task,materials,contact,note,icon_name,status,created_by)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,'draft',?)`,
		req.CampusID, req.StepOrder, req.Title, req.Location,
		req.Lat, req.Lng, req.Duration, req.Task,
		req.Materials, req.Contact, req.Note, req.IconName, createdBy,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// Update 更新步骤（仅 draft 可编辑）
func (r *CampusRepository) Update(id int64, req *model.CampusStepRequest) error {
	res, err := r.db.Exec(
		`UPDATE campus_checkin_steps
		SET step_order=?,title=?,location=?,lat=?,lng=?,duration=?,task=?,
		    materials=?,contact=?,note=?,icon_name=?,updated_at=?
		WHERE id=? AND status='draft'`,
		req.StepOrder, req.Title, req.Location, req.Lat, req.Lng,
		req.Duration, req.Task, req.Materials, req.Contact,
		req.Note, req.IconName, time.Now().Format(time.DateTime), id,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("步骤 %d 不存在或非草稿状态，无法编辑", id)
	}
	return nil
}

// Submit draft → pending_review
func (r *CampusRepository) Submit(id int64) error {
	return r.transition(id, "draft", "pending_review")
}

// Publish pending_review → published
func (r *CampusRepository) Publish(id int64, reviewerID int64) error {
	now := time.Now().Format(time.DateTime)
	res, err := r.db.Exec(
		`UPDATE campus_checkin_steps
		SET status='published',reviewed_by=?,published_at=?,updated_at=?
		WHERE id=? AND status='pending_review'`,
		reviewerID, now, now, id,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("步骤 %d 不存在或状态不是 pending_review", id)
	}
	return nil
}

// Delete 仅 draft 可删除
func (r *CampusRepository) Delete(id int64) error {
	res, err := r.db.Exec(
		`DELETE FROM campus_checkin_steps WHERE id=? AND status='draft'`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("步骤 %d 不存在或非草稿状态，无法删除", id)
	}
	return nil
}

// transition 通用状态流转
func (r *CampusRepository) transition(id int64, from, to string) error {
	res, err := r.db.Exec(
		`UPDATE campus_checkin_steps SET status=?,updated_at=? WHERE id=? AND status=?`,
		to, time.Now().Format(time.DateTime), id, from,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("步骤 %d 不存在或状态不是 %s", id, from)
	}
	return nil
}

// UpdateCoords 仅更新坐标（管理员拖拽校正专用，不受 draft 状态限制）
// 用于已发布步骤的实地位置微调，不改其他字段，不走审核流程。
func (r *CampusRepository) UpdateCoords(id int64, lat, lng float64) error {
	res, err := r.db.Exec(
		`UPDATE campus_checkin_steps
		SET lat=?,lng=?,updated_at=?
		WHERE id=?`,
		lat, lng, time.Now().Format(time.DateTime), id,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("步骤 %d 不存在", id)
	}
	return nil
}
