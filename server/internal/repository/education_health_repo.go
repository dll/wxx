// Package repository 学生身体健康仓库（P4-d：从 education_health_handler 下沉的 12 处裸 SQL）。
package repository

import (
	"database/sql"
	"encoding/json"

	dbutil "github.com/dll/wxx/server/internal/db"
	"github.com/dll/wxx/server/internal/model"
)

// HealthRepo 身体健康数据访问层。
type HealthRepo struct {
	db *sql.DB
}

// NewHealthRepo 创建身体健康仓库。
func NewHealthRepo(db *sql.DB) *HealthRepo {
	return &HealthRepo{db: db}
}

// parseStringSlice 解析 JSON 字符串数组（容错：空/非法返回空切片）。
func parseStringSlice(s string) []string {
	var out []string
	if s == "" {
		return out
	}
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil
	}
	return out
}

// jsonMarshalStringSlice 序列化字符串切片为 JSON（nil → "[]"）。
func jsonMarshalStringSlice(s []string) (string, error) {
	if s == nil {
		return "[]", nil
	}
	b, err := json.Marshal(s)
	if err != nil {
		return "[]", err
	}
	return string(b), nil
}

// ── 身体基本信息 ──

// GetBasicInfo 本人身体基本信息（无记录返回 nil, nil）。
func (r *HealthRepo) GetBasicInfo(userID int64) (*model.HealthBasicInfo, error) {
	info := &model.HealthBasicInfo{}
	err := r.db.QueryRow(
		`SELECT id, height_cm, weight_kg, blood_type, vision_left, vision_right,
		        allergies, past_illness, family_history, emergency_contact, emergency_phone, updated_at
		 FROM health_basic_info WHERE user_id = ?`,
		userID,
	).Scan(&info.ID, &info.HeightCm, &info.WeightKg, &info.BloodType, &info.VisionLeft,
		&info.VisionRight, &info.Allergies, &info.PastIllness, &info.FamilyHistory,
		&info.EmergencyContact, &info.EmergencyPhone, &info.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return info, nil
}

// UpsertBasicInfo 保存身体基本信息（方言适配的 upsert）。
func (r *HealthRepo) UpsertBasicInfo(userID int64, height, weight float64, bloodType, visionLeft, visionRight, allergies, pastIllness, familyHistory, emergencyContact, emergencyPhone string) error {
	stmt := `INSERT INTO health_basic_info
		   (user_id, height_cm, weight_kg, blood_type, vision_left, vision_right,
		    allergies, past_illness, family_history, emergency_contact, emergency_phone, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now','localtime'))
		 ON CONFLICT(user_id) DO UPDATE SET
		   height_cm = excluded.height_cm, weight_kg = excluded.weight_kg,
		   blood_type = excluded.blood_type, vision_left = excluded.vision_left,
		   vision_right = excluded.vision_right, allergies = excluded.allergies,
		   past_illness = excluded.past_illness, family_history = excluded.family_history,
		   emergency_contact = excluded.emergency_contact, emergency_phone = excluded.emergency_phone,
		   updated_at = datetime('now','localtime')`
	_, err := r.db.Exec(dbutil.AdaptForDriver(stmt, dbutil.DriverOf(r.db)),
		userID, height, weight, bloodType, visionLeft, visionRight,
		allergies, pastIllness, familyHistory, emergencyContact, emergencyPhone,
	)
	return err
}

// ── 体检记录 ──

// ListCheckups 本人体检记录。
func (r *HealthRepo) ListCheckups(userID int64) ([]*model.HealthCheckup, error) {
	rows, err := r.db.Query(
		`SELECT id, checkup_date, hospital, conclusion, details, attachments, created_at
		 FROM health_checkups WHERE user_id = ? ORDER BY checkup_date DESC, id DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*model.HealthCheckup, 0)
	for rows.Next() {
		item := &model.HealthCheckup{}
		var att string
		if err := rows.Scan(&item.ID, &item.CheckupDate, &item.Hospital,
			&item.Conclusion, &item.Details, &att, &item.CreatedAt); err != nil {
			continue
		}
		item.Attachments = parseStringSlice(att)
		list = append(list, item)
	}
	return list, rows.Err()
}

// CreateCheckup 新增体检记录。
func (r *HealthRepo) CreateCheckup(userID int64, checkupDate, hospital, conclusion, details string, attachments []string) (int64, error) {
	att, _ := jsonMarshalStringSlice(attachments)
	res, err := r.db.Exec(
		`INSERT INTO health_checkups (user_id, checkup_date, hospital, conclusion, details, attachments)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		userID, checkupDate, hospital, conclusion, details, att,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateCheckup 更新体检记录（返回受影响行数）。
func (r *HealthRepo) UpdateCheckup(userID int64, id int64, checkupDate, hospital, conclusion, details string, attachments []string) (int64, error) {
	att, _ := jsonMarshalStringSlice(attachments)
	res, err := r.db.Exec(
		`UPDATE health_checkups SET checkup_date = ?, hospital = ?, conclusion = ?, details = ?, attachments = ?
		 WHERE id = ? AND user_id = ?`,
		checkupDate, hospital, conclusion, details, att, id, userID,
	)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// DeleteCheckup 删除体检记录（返回受影响行数）。
func (r *HealthRepo) DeleteCheckup(userID, id int64) (int64, error) {
	res, err := r.db.Exec("DELETE FROM health_checkups WHERE id = ? AND user_id = ?", id, userID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// ── 病历记录 ──

// ListRecords 本人病历记录。
func (r *HealthRepo) ListRecords(userID int64) ([]*model.HealthRecord, error) {
	rows, err := r.db.Query(
		`SELECT id, record_date, hospital, department, diagnosis, treatment, attachments, created_at
		 FROM health_records WHERE user_id = ? ORDER BY record_date DESC, id DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*model.HealthRecord, 0)
	for rows.Next() {
		item := &model.HealthRecord{}
		var att string
		if err := rows.Scan(&item.ID, &item.RecordDate, &item.Hospital,
			&item.Department, &item.Diagnosis, &item.Treatment, &att, &item.CreatedAt); err != nil {
			continue
		}
		item.Attachments = parseStringSlice(att)
		list = append(list, item)
	}
	return list, rows.Err()
}

// CreateRecord 新增病历记录。
func (r *HealthRepo) CreateRecord(userID int64, recordDate, hospital, department, diagnosis, treatment string, attachments []string) (int64, error) {
	att, _ := jsonMarshalStringSlice(attachments)
	res, err := r.db.Exec(
		`INSERT INTO health_records (user_id, record_date, hospital, department, diagnosis, treatment, attachments)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		userID, recordDate, hospital, department, diagnosis, treatment, att,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateRecord 更新病历记录（返回受影响行数）。
func (r *HealthRepo) UpdateRecord(userID, id int64, recordDate, hospital, department, diagnosis, treatment string, attachments []string) (int64, error) {
	att, _ := jsonMarshalStringSlice(attachments)
	res, err := r.db.Exec(
		`UPDATE health_records SET record_date = ?, hospital = ?, department = ?, diagnosis = ?, treatment = ?, attachments = ?
		 WHERE id = ? AND user_id = ?`,
		recordDate, hospital, department, diagnosis, treatment, att, id, userID,
	)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// DeleteRecord 删除病历记录（返回受影响行数）。
func (r *HealthRepo) DeleteRecord(userID, id int64) (int64, error) {
	res, err := r.db.Exec("DELETE FROM health_records WHERE id = ? AND user_id = ?", id, userID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// ── 日常记录 ──

// ListDaily 日常健康记录（趋势图）。
func (r *HealthRepo) ListDaily(userID int64, limit int) ([]*model.HealthDailyItem, error) {
	rows, err := r.db.Query(
		`SELECT id, record_date, height_cm, weight_kg, systolic, diastolic, heart_rate, note, created_at
		 FROM health_daily_records WHERE user_id = ? ORDER BY record_date ASC LIMIT ?`,
		userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*model.HealthDailyItem, 0)
	for rows.Next() {
		item := &model.HealthDailyItem{}
		if err := rows.Scan(&item.ID, &item.RecordDate, &item.HeightCm, &item.WeightKg,
			&item.Systolic, &item.Diastolic, &item.HeartRate, &item.Note, &item.CreatedAt); err != nil {
			continue
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

// UpsertDaily 新增/更新某日健康记录（方言适配的 upsert）。
func (r *HealthRepo) UpsertDaily(userID int64, recordDate string, height, weight float64, systolic, diastolic, heartRate int, note string) error {
	stmt := `INSERT INTO health_daily_records
		   (user_id, record_date, height_cm, weight_kg, systolic, diastolic, heart_rate, note, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, datetime('now','localtime'))
		 ON CONFLICT(user_id, record_date) DO UPDATE SET
		   height_cm = excluded.height_cm, weight_kg = excluded.weight_kg,
		   systolic = excluded.systolic, diastolic = excluded.diastolic,
		   heart_rate = excluded.heart_rate, note = excluded.note,
		   updated_at = datetime('now','localtime')`
	_, err := r.db.Exec(dbutil.AdaptForDriver(stmt, dbutil.DriverOf(r.db)),
		userID, recordDate, height, weight, systolic, diastolic, heartRate, note,
	)
	return err
}

// DeleteDaily 删除某日记录（返回受影响行数）。
func (r *HealthRepo) DeleteDaily(userID int64, date string) (int64, error) {
	res, err := r.db.Exec(`DELETE FROM health_daily_records WHERE user_id = ? AND record_date = ?`, userID, date)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
