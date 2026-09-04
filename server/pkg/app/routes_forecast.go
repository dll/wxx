package app

import (
	"github.com/dll/wxx/server/internal/auth"
	"github.com/gin-gonic/gin"
)

// registerForecastRoutes 注册问题预案分析、列表和统计路由。
func registerForecastRoutes(secured *gin.RouterGroup, d *deps) {
	forecast := secured.Group("/forecast")
	forecast.POST("/analysis", auth.RequireCapability(auth.CollegeForecast), d.forecastH.Analyze)
	forecast.GET("/issues", auth.RequireCapability(auth.CollegeForecast), d.forecastH.ListForecasts)
	forecast.GET("/issues/:id", auth.RequireCapability(auth.CollegeForecast), d.forecastH.GetForecast)
	forecast.PUT("/issues/:id/status", auth.RequireCapability(auth.CollegeForecast), d.forecastH.UpdateStatus)
	forecast.GET("/statistics", auth.RequireCapability(auth.CollegeForecast), d.forecastH.GetStatistics)
}
