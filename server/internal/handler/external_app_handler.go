package handler

import (
	"net/http"

	"github.com/dll/wxx/server/internal/model"
	"github.com/gin-gonic/gin"
)

// ExternalAppHandler 第三方应用接入接口
type ExternalAppHandler struct{}

func NewExternalAppHandler() *ExternalAppHandler {
	return &ExternalAppHandler{}
}

// ListApps 获取应用列表
func (h *ExternalAppHandler) ListApps(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": []model.ExternalApp{
			{
				ID: 1, AppKey: "library", Name: "智慧图书馆", Description: "馆藏查询、座位预约、借阅记录",
				IconURL: "/assets/library.png", AppURL: "https://lib.chzu.edu.cn",
				Mode: "external_link", Category: "学习", Status: "active",
			},
			{
				ID: 2, AppKey: "canteen", Name: "食堂点评", Description: "菜品推荐、排队情况、营养分析",
				IconURL: "/assets/canteen.png", AppURL: "https://canteen.chzu.edu.cn",
				Mode: "webview", Category: "生活", Status: "active",
			},
			{
				ID: 3, AppKey: "xuegong", Name: "学工系统", Description: "奖学金申请、请假审批、信息查询",
				IconURL: "/assets/xuegong.png", AppURL: "https://xuegong.chzu.edu.cn",
				Mode: "reverse_proxy", Category: "管理", Status: "active",
			},
		},
	})
}

// GetApp 获取单个应用详情
func (h *ExternalAppHandler) GetApp(c *gin.Context) {
	appKey := c.Param("key")
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": model.ExternalApp{
			ID: 1, AppKey: appKey, Name: "智慧图书馆",
			Description: "馆藏查询、座位预约、借阅记录",
			AppURL:      "https://lib.chzu.edu.cn", Mode: "external_link",
			Category: "学习", Status: "active",
		},
	})
}
