package service

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
)

// AIBriefingService AI 简讯业务：CRUD、汇总统计、导出数据、RSS/Atom 自动抓取
type AIBriefingService struct {
	repo        *repository.AIBriefingRepo
	httpClient  *http.Client
	fetchLimit  int // 单次抓取单源上限
	fetchMaxLen int // 摘要截断长度
}

// NewAIBriefingService 创建 AI 简讯服务
func NewAIBriefingService(repo *repository.AIBriefingRepo) *AIBriefingService {
	return &AIBriefingService{
		repo:        repo,
		httpClient:  &http.Client{Timeout: 20 * time.Second},
		fetchLimit:  30,
		fetchMaxLen: 300,
	}
}

// List 管理端分页查询
func (s *AIBriefingService) List(statusFilter, category, q string, page, pageSize int) ([]*model.AIBriefing, int64, error) {
	return s.repo.List(statusFilter, category, q, page, pageSize)
}

// ListUserVisible 用户端列表
func (s *AIBriefingService) ListUserVisible(category, q string, limit int) ([]*model.AIBriefing, error) {
	return s.repo.ListUserVisible(category, q, limit)
}

// Get 单条
func (s *AIBriefingService) Get(id int64) (*model.AIBriefing, error) {
	return s.repo.Get(id)
}

// GetByIDs 批量查询（导出用）
func (s *AIBriefingService) GetByIDs(ids []int64) ([]*model.AIBriefing, error) {
	return s.repo.GetByIDs(ids)
}

// Create 手动录入
func (s *AIBriefingService) Create(b *model.AIBriefing, operatorID int64) (int64, error) {
	if strings.TrimSpace(b.Topic) == "" {
		return 0, fmt.Errorf("主题不能为空")
	}
	if b.Category == "" {
		b.Category = "ai_teaching"
	}
	if b.PublishedAt == "" {
		b.PublishedAt = time.Now().Format("2006-01-02 15:04:05")
	}
	b.CreatedBy = operatorID
	return s.repo.Create(b)
}

// Update 更新
func (s *AIBriefingService) Update(b *model.AIBriefing) error {
	existing, err := s.repo.Get(b.ID)
	if err != nil || existing == nil {
		return fmt.Errorf("资讯不存在: id=%d", b.ID)
	}
	if strings.TrimSpace(b.Topic) == "" {
		return fmt.Errorf("主题不能为空")
	}
	if b.Category == "" {
		b.Category = existing.Category
	}
	if b.PublishedAt == "" {
		b.PublishedAt = existing.PublishedAt
	}
	return s.repo.Update(b)
}

// UpdateStatus 上下架
func (s *AIBriefingService) UpdateStatus(id int64, status int) error {
	if status != 0 && status != 1 {
		return fmt.Errorf("状态仅支持 0(下架)/1(上架)")
	}
	return s.repo.UpdateStatus(id, status)
}

// Delete 删除
func (s *AIBriefingService) Delete(id int64) error {
	return s.repo.Delete(id)
}

// DeleteMany 批量删除（删除历史记录）
func (s *AIBriefingService) DeleteMany(ids []int64) (int64, error) {
	return s.repo.DeleteMany(ids)
}

// ClearAll 清空全部
func (s *AIBriefingService) ClearAll() (int64, error) {
	return s.repo.ClearAll()
}

// Stats 汇总统计
func (s *AIBriefingService) Stats() (*model.AIBriefingStats, error) {
	return s.repo.Stats()
}

// ── 来源管理 ──

// ListSources 全部来源
func (s *AIBriefingService) ListSources() ([]*model.AIBriefingSource, error) {
	return s.repo.ListSources()
}

// CreateSource 新增来源
func (s *AIBriefingService) CreateSource(src *model.AIBriefingSource) (int64, error) {
	if strings.TrimSpace(src.Name) == "" {
		return 0, fmt.Errorf("来源名称不能为空")
	}
	if src.Category == "" {
		src.Category = "ai_teaching"
	}
	if src.FetchTime == "" {
		src.FetchTime = "08:00"
	}
	return s.repo.CreateSource(src)
}

// UpdateSource 更新来源
func (s *AIBriefingService) UpdateSource(src *model.AIBriefingSource) error {
	if strings.TrimSpace(src.Name) == "" {
		return fmt.Errorf("来源名称不能为空")
	}
	return s.repo.UpdateSource(src)
}

// DeleteSource 删除来源
func (s *AIBriefingService) DeleteSource(id int64) error {
	return s.repo.DeleteSource(id)
}

// ── 自动抓取（RSS/Atom）──

// FetchNow 立即抓取全部启用来源（返回各来源抓取条数）
func (s *AIBriefingService) FetchNow() map[string]int {
	sources, err := s.repo.ListEnabledFetchSources()
	if err != nil {
		log.Printf("AI 简讯抓取：查询来源失败 %v", err)
		return nil
	}
	result := map[string]int{}
	for _, src := range sources {
		n := s.fetchSource(src)
		result[src.Name] = n
	}
	return result
}

// RunLoop 定时抓取调度：每分钟检查是否有来源到了抓取时刻
func (s *AIBriefingService) RunLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.FetchDue()
		}
	}
}

// FetchDue 抓取所有到达抓取时刻且当天未抓取的来源
func (s *AIBriefingService) FetchDue() {
	sources, err := s.repo.ListEnabledFetchSources()
	if err != nil {
		return
	}
	now := time.Now()
	today := now.Format("2006-01-02")
	for _, src := range sources {
		// 跳过：非当天、来源尚未启用、今天已抓取过
		fetchTime, terr := time.Parse("15:04", src.FetchTime)
		if terr != nil {
			continue
		}
		due := time.Date(now.Year(), now.Month(), now.Day(), fetchTime.Hour(), fetchTime.Minute(), 0, 0, now.Location())
		if now.Before(due) {
			continue
		}
		if strings.HasPrefix(src.LastFetchAt, today) {
			continue
		}
		s.fetchSource(src)
	}
}

// fetchSource 抓取单个来源的 RSS/Atom 并入库
func (s *AIBriefingService) fetchSource(src *model.AIBriefingSource) int {
	items := s.parseFeed(src.URL)
	if len(items) == 0 {
		log.Printf("AI 简讯抓取：%s 未解析到条目 url=%s", src.Name, src.URL)
		return 0
	}
	briefings := make([]*model.AIBriefing, 0, len(items))
	nowStr := time.Now().Format("2006-01-02 15:04:05")
	for _, it := range items {
		if len(briefings) >= s.fetchLimit {
			break
		}
		if strings.TrimSpace(it.title) == "" {
			continue
		}
		pub := it.published
		if pub == "" {
			pub = nowStr
		}
		briefings = append(briefings, &model.AIBriefing{
			Source:      src.Name,
			Category:    src.Category,
			Topic:       strings.TrimSpace(it.title),
			Summary:     truncateRunes(strings.TrimSpace(it.description), s.fetchMaxLen),
			Link:        strings.TrimSpace(it.link),
			PublishedAt: pub,
			FetchedAt:   nowStr,
		})
	}
	written, err := s.repo.CreateMany(briefings)
	if err != nil {
		log.Printf("AI 简讯抓取：%s 入库失败 %v", src.Name, err)
		return 0
	}
	_ = s.repo.SetSourceLastFetch(src.ID, nowStr)
	log.Printf("AI 简讯抓取：%s 新增 %d 条", src.Name, written)
	return written
}

// rssItem 解析出的条目
type rssItem struct {
	title       string
	link        string
	description string
	published   string
}

// parseFeed 解析 RSS 2.0 / Atom 1.0（轻量实现，不引入第三方库）
func (s *AIBriefingService) parseFeed(url string) []rssItem {
	resp, err := s.httpClient.Get(url)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20)) // 2MB 上限
	if err != nil {
		return nil
	}
	text := string(body)

	// 尝试 RSS 2.0
	var rss struct {
		Channel struct {
			Items []struct {
				Title       string `xml:"title"`
				Link        string `xml:"link"`
				Description string `xml:"description"`
				PubDate     string `xml:"pubDate"`
			} `xml:"item"`
		} `xml:"channel"`
	}
	if err := xml.Unmarshal(body, &rss); err == nil && len(rss.Channel.Items) > 0 {
		items := make([]rssItem, 0, len(rss.Channel.Items))
		for _, it := range rss.Channel.Items {
			items = append(items, rssItem{
				title:       it.Title,
				link:        it.Link,
				description: it.Description,
				published:   normalizeRFC2822(it.PubDate),
			})
		}
		return items
	}

	// 尝试 Atom 1.0
	var atom struct {
		Entries []struct {
			Title   string `xml:"title"`
			Link    struct {
				Href string `xml:"href,attr"`
			} `xml:"link"`
			Summary   string `xml:"summary"`
			Content   string `xml:"content"`
			Published string `xml:"published"`
			Updated   string `xml:"updated"`
		} `xml:"entry"`
	}
	if err := xml.Unmarshal(body, &atom); err == nil && len(atom.Entries) > 0 {
		items := make([]rssItem, 0, len(atom.Entries))
		for _, e := range atom.Entries {
			desc := strings.TrimSpace(e.Summary)
			if desc == "" {
				desc = strings.TrimSpace(e.Content)
			}
			pub := e.Published
			if pub == "" {
				pub = e.Updated
			}
			items = append(items, rssItem{
				title:       strings.TrimSpace(e.Title),
				link:        strings.TrimSpace(e.Link.Href),
				description: desc,
				published:   normalizeRFC3339(pub),
			})
		}
		return items
	}

	_ = text
	return nil
}

// normalizeRFC2822 将 RFC2822 时间转成 2006-01-02 15:04:05
func normalizeRFC2822(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	for _, layout := range []string{
		time.RFC1123, time.RFC1123Z, time.RFC822, time.RFC822Z,
		"Mon, 02 Jan 2006 15:04:05 -0700 MST",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.Format("2006-01-02 15:04:05")
		}
	}
	return ""
}

// normalizeRFC3339 将 RFC3339 时间转成 2006-01-02 15:04:05
func normalizeRFC3339(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.Format("2006-01-02 15:04:05")
	}
	return ""
}

// truncateRunes 按字符数截断并追加省略号
func truncateRunes(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "…"
}
