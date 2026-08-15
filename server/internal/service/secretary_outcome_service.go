package service

import (
	"context"
	"fmt"

	"github.com/dll/wxx/server/internal/repository"
)

// SecretaryOutcomeService 书记教育成果服务
// 覆盖：毕业去向登记（学生自报/教辅录入）+ 教辅审核 + 书记教育成果大屏聚合。
// 数据全部来自真实表，未审核的去向不进入统计，就业率/考研率在无真实数据时诚实返回 not_available。
type SecretaryOutcomeService struct {
	repo *repository.SecretaryOutcomeRepo
}

func NewSecretaryOutcomeService(repo *repository.SecretaryOutcomeRepo) *SecretaryOutcomeService {
	return &SecretaryOutcomeService{repo: repo}
}

// OutcomeTypeMeta 去向类型元信息
func (s *SecretaryOutcomeService) OutcomeTypeMeta() map[string]string {
	return repository.OutcomeTypeMeta
}

// SubmitOutcome 登记毕业去向。
// role="student"（学生自报）→ status=pending，需教辅审核；
// role 为教辅（counselor/teacher/assistant/college_admin 等）→ 可直录为 pending，由另一教辅审核。
func (s *SecretaryOutcomeService) SubmitOutcome(ctx context.Context, o *repository.GraduationOutcome) (int64, error) {
	if o == nil {
		return 0, fmt.Errorf("去向数据为空")
	}
	if o.StudentID <= 0 {
		return 0, fmt.Errorf("学生不能为空")
	}
	if _, ok := repository.OutcomeTypeMeta[o.OutcomeType]; !ok {
		return 0, fmt.Errorf("未知去向类型: %s", o.OutcomeType)
	}
	if o.OutcomeType == "employment" || o.OutcomeType == "postgrad" || o.OutcomeType == "study_abroad" || o.OutcomeType == "entrepreneurship" {
		if o.EmployerName == "" {
			return 0, fmt.Errorf("%s须填写去向单位/院校", repository.OutcomeTypeMeta[o.OutcomeType])
		}
	}
	if o.Status == "" {
		o.Status = "pending"
	}
	return s.repo.CreateOutcome(o)
}

// ListOutcomes 查询去向（教辅/书记可用）
func (s *SecretaryOutcomeService) ListOutcomes(ctx context.Context, status, college string, year int, studentID int64, limit int) ([]repository.GraduationOutcome, error) {
	return s.repo.ListOutcomes(status, college, year, studentID, limit)
}

// ReviewOutcome 审核去向（教辅）
func (s *SecretaryOutcomeService) ReviewOutcome(ctx context.Context, id, reviewerID int64, reviewerName, status, note string) error {
	return s.repo.ReviewOutcome(id, reviewerID, reviewerName, status, note)
}

// CountPending 待审核条数
func (s *SecretaryOutcomeService) CountPending(ctx context.Context) (int, error) {
	return s.repo.CountPendingOutcomes()
}

// OutcomeDashboard 书记教育成果大屏（college 为空=全校，非空=本院）
func (s *SecretaryOutcomeService) OutcomeDashboard(ctx context.Context, college string) (map[string]interface{}, error) {
	return s.repo.EducationOutcomeDashboard(college)
}

// PartyDashboard 党建育人聚合（ownerID 为空=全校，非空=本院（按 users.owner_id 精确匹配））
func (s *SecretaryOutcomeService) PartyDashboard(ctx context.Context, ownerID string) (map[string]interface{}, error) {
	return s.repo.PartyDashboard(ownerID)
}

// CreatePartyRecord 党课/活动登记（教师/教辅，蓝图第3块）
func (s *SecretaryOutcomeService) CreatePartyRecord(ctx context.Context, title, studyType, content string, duration int, studyDate string, createdBy int64, createdByRole string, studentIDs []int64) (int64, error) {
	ids, err := s.repo.CreatePartyRecord(title, studyType, content, duration, studyDate, createdBy, createdByRole, studentIDs)
	if err != nil {
		return 0, err
	}
	if len(ids) == 0 {
		return 0, fmt.Errorf("登记失败")
	}
	return ids[0], nil
}

// ListPartyRecords 查看党课/活动登记（本人登记的）
func (s *SecretaryOutcomeService) ListPartyRecords(ctx context.Context, opID int64) ([]map[string]interface{}, error) {
	return s.repo.ListPartyRecords(opID)
}

// DeletePartyRecord 删除本人登记的党课/活动
func (s *SecretaryOutcomeService) DeletePartyRecord(ctx context.Context, id, opID int64) error {
	return s.repo.DeletePartyRecord(id, opID)
}

// CollabDashboard 协同育人总览（书记视角，蓝图第2块）
func (s *SecretaryOutcomeService) CollabDashboard(ctx context.Context, ownerID string) (map[string]interface{}, error) {
	return s.repo.CollabDashboard(ownerID)
}
