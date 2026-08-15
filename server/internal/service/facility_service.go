package service

import (
	"context"
	"fmt"
	"time"

	"github.com/dll/wxx/server/internal/repository"
)

// FacilityService 后勤服务台服务
// 覆盖实验室开门/关门、教室保洁、热水、宿舍查岗、环卫、图书馆借阅等
// 后勤保障类工作。所有记录均为操作人手动登记的真实数据（data_source=real），
// 不展示示例/编造数据。
type FacilityService struct {
	repo *repository.FacilityRepo
}

func NewFacilityService(repo *repository.FacilityRepo) *FacilityService {
	return &FacilityService{repo: repo}
}

// RoleMeta 暴露岗位类型元信息（供前端下拉/看板）
func (s *FacilityService) RoleMeta() map[string]string {
	return repository.FacilityRoleMeta
}

// CreateRecord 登记一条后勤服务记录
func (s *FacilityService) CreateRecord(ctx context.Context, rec *repository.FacilityRecord) (int64, error) {
	if rec == nil {
		return 0, fmt.Errorf("记录为空")
	}
	if rec.Role == "" {
		return 0, fmt.Errorf("岗位类型不能为空")
	}
	if rec.Title == "" {
		return 0, fmt.Errorf("事项简述不能为空")
	}
	if rec.OccurredAt == "" {
		rec.OccurredAt = time.Now().Format(time.RFC3339)
	}
	if _, ok := repository.FacilityRoleMeta[rec.Role]; !ok {
		return 0, fmt.Errorf("未知岗位类型: %s", rec.Role)
	}
	return s.repo.Create(rec)
}

// ListRecords 查询服务记录
func (s *FacilityService) ListRecords(ctx context.Context, role, operatorName string, studentID int64, from, to string, limit int) ([]repository.FacilityRecord, error) {
	return s.repo.List(role, operatorName, studentID, from, to, limit)
}

// Dashboard 后勤台看板
func (s *FacilityService) Dashboard(ctx context.Context, operatorID int64, from, to string) (map[string]interface{}, error) {
	byRole, total, stuCnt, err := s.repo.Dashboard(operatorID, from, to)
	if err != nil {
		return nil, err
	}
	// 补齐所有岗位类型（未发生服务的岗位显示 0，诚实展示）
	roleReadable := map[string]string{}
	for k, v := range repository.FacilityRoleMeta {
		roleReadable[k] = v
		if _, ok := byRole[k]; !ok {
			byRole[k] = 0
		}
	}
	return map[string]interface{}{
		"by_role":        byRole,
		"role_readable":  roleReadable,
		"total":          total,
		"student_served": stuCnt,
		"data_source":    "real",
	}, nil
}
