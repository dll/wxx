package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CultureHandler 校园文化智能体接口（骨架版：返回种子数据，后续接入资源管理）
type CultureHandler struct{}

func NewCultureHandler() *CultureHandler {
	return &CultureHandler{}
}

// Anthems 校歌曲库（学校自有资源）
func (h *CultureHandler) Anthems(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"items": []gin.H{
			{
				"id":         "anthem-school",
				"title":      "滁州学院校歌",
				"category":   "school",
				"duration":   198,
				"lyric":      "求是 求真 求美 求新，琅琊西涧畔我们奋发前行……",
				"audio_url":  "https://example.edu.cn/assets/audio/school-anthem.mp3",
				"cover_url":  "https://example.edu.cn/assets/img/anthem-school.jpg",
				"updated_at": "2026-04-01",
			},
			{
				"id":         "anthem-info",
				"title":      "计算机学院院歌",
				"category":   "college",
				"duration":   156,
				"lyric":      "代码行行点亮夜空，比特跳跃指尖之上……",
				"audio_url":  "https://example.edu.cn/assets/audio/info-anthem.mp3",
				"cover_url":  "https://example.edu.cn/assets/img/anthem-info.jpg",
				"updated_at": "2026-03-15",
			},
			{
				"id":         "anthem-cuit",
				"title":      "信息时代之歌",
				"category":   "classic",
				"duration":   224,
				"lyric":      "我们这一代，遇见了人工智能，也遇见了星辰大海……",
				"audio_url":  "https://example.edu.cn/assets/audio/info-classic.mp3",
				"cover_url":  "https://example.edu.cn/assets/img/anthem-classic.jpg",
				"updated_at": "2026-02-28",
			},
		},
	})
}

// Radio 校园广播（节目单 + 当前直播）
func (h *CultureHandler) Radio(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"now_playing": gin.H{
			"id":         "radio-morning",
			"title":      "晨间新闻播报",
			"host":       "信院广播站",
			"start_at":   "2026-05-16 07:30",
			"end_at":     "2026-05-16 08:00",
			"stream_url": "https://example.edu.cn/stream/morning.m3u8",
			"is_live":    true,
		},
		"schedule": []gin.H{
			{"time": "07:30-08:00", "title": "晨间新闻播报", "category": "新闻"},
			{"time": "12:00-12:30", "title": "午间心理树洞", "category": "心理"},
			{"time": "17:30-18:00", "title": "校园声纳·学术访谈", "category": "学术"},
			{"time": "21:00-21:30", "title": "夜读·诗与远方", "category": "文学"},
		},
		"recent_episodes": []gin.H{
			{"id": "ep-2026-05-15-night", "title": "夜读·琅琊山的春", "duration": 1820, "audio_url": "https://example.edu.cn/audio/ep1.mp3", "published_at": "2026-05-15"},
			{"id": "ep-2026-05-14-talk", "title": "校园声纳·智能时代的青年责任", "duration": 1620, "audio_url": "https://example.edu.cn/audio/ep2.mp3", "published_at": "2026-05-14"},
		},
	})
}

// Lectures 学术讲座（外链联动：B 站/企微直播/腾讯会议等）
func (h *CultureHandler) Lectures(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"upcoming": []gin.H{
			{
				"id":       "lec-2026-05-20",
				"title":    "大模型时代的工程范式变革",
				"speaker":  "张三 · 中国科学技术大学",
				"start_at": "2026-05-20 19:00",
				"duration": 90,
				"venue":    "线上 + 计算机学院 312 报告厅",
				"link":     "https://meeting.tencent.com/dm/example",
				"platform": "腾讯会议",
				"tags":     []string{"AI", "工程化"},
			},
			{
				"id":       "lec-2026-05-25",
				"title":    "信息安全前沿：从供应链攻击到主动防御",
				"speaker":  "李四 · 浙江大学",
				"start_at": "2026-05-25 14:30",
				"duration": 120,
				"venue":    "图书馆学术报告厅",
				"link":     "https://live.bilibili.com/example",
				"platform": "B 站直播",
				"tags":     []string{"安全", "前沿"},
			},
		},
		"replay": []gin.H{
			{
				"id":       "lec-2026-05-08",
				"title":    "智慧校园建设的实践与思考",
				"speaker":  "王五 · 清华大学",
				"link":     "https://www.bilibili.com/video/BVexample",
				"duration": 95,
				"played":   1820,
				"tags":     []string{"智慧校园"},
			},
		},
	})
}

// Events 校园活动（报名 + 推送）
func (h *CultureHandler) Events(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"upcoming": []gin.H{
			{
				"id":           "event-tech-festival",
				"title":        "第十二届信息技术文化节",
				"category":     "学术",
				"start_at":     "2026-05-28 09:00",
				"end_at":       "2026-06-05 17:00",
				"venue":        "计算机学院全院",
				"organizer":    "计算机学院团委",
				"capacity":     500,
				"registered":   217,
				"register_url": "/api/v1/culture/events/event-tech-festival/register",
				"tags":         []string{"编程大赛", "技术沙龙", "答辩展示"},
			},
			{
				"id":           "event-volunteer-spring",
				"title":        "春季校园志愿服务月",
				"category":     "志愿",
				"start_at":     "2026-05-18 08:00",
				"end_at":       "2026-06-18 18:00",
				"venue":        "全校",
				"organizer":    "学生会·青年志愿者协会",
				"capacity":     300,
				"registered":   142,
				"register_url": "/api/v1/culture/events/event-volunteer-spring/register",
				"tags":         []string{"志愿服务", "社区共建", "敬老助残"},
			},
			{
				"id":           "event-anthem-night",
				"title":        "校歌之夜·原创音乐节",
				"category":     "文化",
				"start_at":     "2026-05-22 19:30",
				"end_at":       "2026-05-22 22:00",
				"venue":        "大学生活动中心",
				"organizer":    "学生会文艺部",
				"capacity":     800,
				"registered":   356,
				"register_url": "/api/v1/culture/events/event-anthem-night/register",
				"tags":         []string{"原创音乐", "校歌"},
			},
		},
		"recommended": []gin.H{
			{"id": "event-tech-festival", "reason": "与你的专业相关"},
			{"id": "event-volunteer-spring", "reason": "本月志愿时长仍差 12h"},
		},
	})
}

// Volunteer 志愿服务（个人时长 + 项目推荐）
func (h *CultureHandler) Volunteer(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"my_summary": gin.H{
			"total_hours":     38.5,
			"verified_hours":  32.0,
			"pending_hours":   6.5,
			"projects_joined": 7,
			"badges":          []string{"敬老助残金牌", "社区共建银牌"},
		},
		"projects": []gin.H{
			{
				"id":           "vol-elderly",
				"title":        "敬老院常态化陪伴",
				"organizer":    "计算机学院团委",
				"location":     "琅琊区第一敬老院",
				"frequency":    "每周六上午",
				"hours":        3,
				"participants": 28,
				"capacity":     40,
				"tags":         []string{"敬老", "陪伴"},
			},
			{
				"id":           "vol-coding-class",
				"title":        "乡村青少年编程启蒙课",
				"organizer":    "青年志愿者协会",
				"location":     "凤阳县乡村小学（线上）",
				"frequency":    "周末下午",
				"hours":        2,
				"participants": 45,
				"capacity":     60,
				"tags":         []string{"科技支教", "编程"},
			},
			{
				"id":           "vol-blood",
				"title":        "无偿献血校园开放日",
				"organizer":    "校红十字会",
				"location":     "图书馆门前广场",
				"frequency":    "5/30 单次",
				"hours":        4,
				"participants": 16,
				"capacity":     80,
				"tags":         []string{"公益", "健康"},
			},
		},
	})
}
