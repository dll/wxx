package service

import (
	"time"

	"github.com/dll/wxx/server/internal/repository"
)

// CheckinService 学生打卡业务服务
type CheckinService struct {
	repo *repository.CheckinRepo
}

// NewCheckinService 创建打卡服务
func NewCheckinService(repo *repository.CheckinRepo) *CheckinService {
	return &CheckinService{repo: repo}
}

// CheckinResult 打卡结果
type CheckinResult struct {
	Success       bool     `json:"success"`
	Message       string   `json:"message"`
	Date          string   `json:"date"`
	Streak        int      `json:"streak"`         // 当前连续天数
	TotalDays     int      `json:"total_days"`     // 累计打卡天数
	LongestStreak int      `json:"longest_streak"` // 历史最长连续
	TodayChecked  bool     `json:"today_checked"`  // 今日是否已打卡
	MakeupLeft    int      `json:"makeup_left"`    // 当月剩余补签次数（上限 2 次）
	Milestones    []string `json:"milestones"`     // 达成的里程碑
}

// makeupMonthlyLimit 每月补签上限（断签保护，docs/蔚小芯角色功能.md P1）
const makeupMonthlyLimit = 2

// DoCheckin 执行每日打卡
func (s *CheckinService) DoCheckin(userID int64, mood, note string) *CheckinResult {
	today := time.Now().Format("2006-01-02")

	_, err := s.repo.DoCheckin(userID, today, mood, note)
	if err != nil {
		return &CheckinResult{
			Success: false,
			Message: "打卡失败，请稍后重试",
			Date:    today,
		}
	}

	streak := s.repo.CalcStreak(userID)
	total := s.repo.CountTotal(userID)
	longest := s.repo.CalcLongestStreak(userID)

	// 里程碑判定
	milestones := checkMilestones(streak, total)

	return &CheckinResult{
		Success:       true,
		Message:       checkinMessage(streak),
		Date:          today,
		Streak:        streak,
		TotalDays:     total,
		LongestStreak: longest,
		TodayChecked:  true,
		MakeupLeft:    s.MakeupLeft(userID),
		Milestones:    milestones,
	}
}

// GetHistory 获取打卡历史统计
func (s *CheckinService) GetHistory(userID int64) *CheckinResult {
	today := time.Now().Format("2006-01-02")
	streak := s.repo.CalcStreak(userID)
	total := s.repo.CountTotal(userID)
	longest := s.repo.CalcLongestStreak(userID)
	todayChecked := s.repo.IsCheckedToday(userID)

	return &CheckinResult{
		Success:       true,
		Date:          today,
		Streak:        streak,
		TotalDays:     total,
		LongestStreak: longest,
		TodayChecked:  todayChecked,
		MakeupLeft:    s.MakeupLeft(userID),
		Milestones:    checkMilestones(streak, total),
	}
}

// MakeupLeft 当月剩余补签次数
func (s *CheckinService) MakeupLeft(userID int64) int {
	month := time.Now().Format("2006-01")
	used := s.repo.CountMakeupInMonth(userID, month)
	left := makeupMonthlyLimit - used
	if left < 0 {
		left = 0
	}
	return left
}

// MakeupCheckin 补签：为当月已错过的日期补一次打卡
// 规则：仅限当月且为过去日期；当日已打卡/未来日期拒绝；每月最多 2 次。
func (s *CheckinService) MakeupCheckin(userID int64, date, mood, note string) *CheckinResult {
	now := time.Now()
	today := now.Format("2006-01-02")
	month := now.Format("2006-01")

	target, err := time.Parse("2006-01-02", date)
	if err != nil {
		return &CheckinResult{Success: false, Message: "日期格式不正确", Date: today}
	}
	// 仅限当月且为过去日期（不允许补未来，也不允许跨月补）
	if target.Format("2006-01") != month || date >= today {
		return &CheckinResult{Success: false, Message: "仅可补签本月已错过的日期", Date: today}
	}
	// 当日不可补签（当日走正常打卡）
	if date == today {
		return &CheckinResult{Success: false, Message: "今天请直接打卡", Date: today}
	}
	// 已打卡日期无需补签
	if s.repo.HasChecked(userID, date) {
		return &CheckinResult{Success: false, Message: "该日期已打卡，无需补签", Date: today}
	}
	// 月度次数校验
	if s.MakeupLeft(userID) <= 0 {
		return &CheckinResult{Success: false, Message: "本月补签次数已用完（每月限 2 次）", Date: today}
	}

	if _, err := s.repo.MakeupCheckin(userID, date, month, mood, note); err != nil {
		return &CheckinResult{Success: false, Message: "补签失败，请稍后重试", Date: today}
	}

	streak := s.repo.CalcStreak(userID)
	total := s.repo.CountTotal(userID)
	longest := s.repo.CalcLongestStreak(userID)

	return &CheckinResult{
		Success:       true,
		Message:       "补签成功！继续保持好习惯 🌱",
		Date:          today,
		Streak:        streak,
		TotalDays:     total,
		LongestStreak: longest,
		TodayChecked:  s.repo.IsCheckedToday(userID),
		MakeupLeft:    s.MakeupLeft(userID),
		Milestones:    checkMilestones(streak, total),
	}
}

// GetRecentDates 获取最近的打卡日期（供日历展示）
func (s *CheckinService) GetRecentDates(userID int64, limit int) ([]string, error) {
	return s.repo.GetRecentDates(userID, limit)
}

// checkinMessage 根据连续天数生成鼓励语
func checkinMessage(streak int) string {
	switch {
	case streak >= 100:
		return "百日连续打卡！你是坚持的典范 🏆"
	case streak >= 30:
		return "一个月了！坚持的力量正在积累 💪"
	case streak >= 14:
		return "两周连续打卡，习惯正在形成 🌟"
	case streak >= 7:
		return "连续一周！好习惯已经养成 ✨"
	case streak >= 3:
		return "三天连续打卡，继续保持 🔥"
	default:
		return "打卡成功！每一天都是新的开始 ☀️"
	}
}

// checkMilestones 检查达成的里程碑
func checkMilestones(streak, total int) []string {
	var ms []string
	// 连续天数里程碑
	streakMilestones := []int{3, 7, 14, 30, 60, 100, 365}
	for _, m := range streakMilestones {
		if streak == m {
			ms = append(ms, streakMilestoneName(m))
		}
	}
	// 累计天数里程碑
	totalMilestones := []int{10, 30, 50, 100, 200, 365}
	for _, m := range totalMilestones {
		if total == m {
			ms = append(ms, totalMilestoneName(m))
		}
	}
	return ms
}

func streakMilestoneName(days int) string {
	switch days {
	case 3:
		return "初露锋芒（连续3天）"
	case 7:
		return "一周达人（连续7天）"
	case 14:
		return "两周坚守（连续14天）"
	case 30:
		return "月度之星（连续30天）"
	case 60:
		return "双月传奇（连续60天）"
	case 100:
		return "百日大师（连续100天）"
	case 365:
		return "年度王者（连续365天）"
	default:
		return ""
	}
}

func totalMilestoneName(days int) string {
	switch days {
	case 10:
		return "累计10天"
	case 30:
		return "累计30天"
	case 50:
		return "半百里程碑"
	case 100:
		return "累计百日"
	case 200:
		return "两百日纪念"
	case 365:
		return "全年打卡"
	default:
		return ""
	}
}
