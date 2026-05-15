package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// UnionHandler 学生会角色 AI 功能接口
type UnionHandler struct{}

func NewUnionHandler() *UnionHandler {
	return &UnionHandler{}
}

// EventPlan AI 活动策划
func (h *UnionHandler) EventPlan(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"response": "活动策划方案: 信息学院编程马拉松, 时间2026年5月25日9:00-21:00, 地点信息楼多功能厅。活动目标: 提升学生编程实战能力, 促进跨年级技术交流。流程: 09:00开幕组队, 09:30-18:00编程开发, 18:00-19:00晚餐, 19:00-20:00作品展示, 20:00-21:00评审颁奖。预算: 奖品2000元, 餐饮1500元, 物料500元。宣传: 公众号推文+海报, 班级群通知, 线下展架。",
	})
}

// PosterGen AI 海报文案生成
func (h *UnionHandler) PosterGen(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"response": "海报文案: CODE MARATHON 2026 - 12小时用代码改变世界! 信息学院首届编程马拉松来袭, 组队挑战赢取万元大奖! 时间5月25日全天, 地点信息楼多功能厅。亮点: 企业导师现场指导, 免费餐饮和咖啡, 优秀作品推荐实习机会。主办: 信息学院学生会, 协办: ACM协会、创新创业中心。",
	})
}
