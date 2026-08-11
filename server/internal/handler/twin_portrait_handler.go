package handler

import (
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/dll/wxx/server/internal/middleware"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/service"
	"github.com/gin-gonic/gin"
)

// TwinPortraitHandler 数字孪生画像 HTTP handler
type TwinPortraitHandler struct {
	svc *service.TwinPortraitService
}

// NewTwinPortraitHandler 创建画像 handler
func NewTwinPortraitHandler(svc *service.TwinPortraitService) *TwinPortraitHandler {
	return &TwinPortraitHandler{svc: svc}
}

// List 查询用户全部画像 GET /api/v1/twin-portraits
func (h *TwinPortraitHandler) List(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	views, err := h.svc.ListPortraits(userCtx.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询画像失败"})
		return
	}
	if views == nil {
		views = []*model.TwinPortraitView{}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": views})
}

// Get 查询指定类型画像 GET /api/v1/twin-portraits/:type
func (h *TwinPortraitHandler) Get(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	prototypeType := c.Param("type")
	view, err := h.svc.GetPortrait(userCtx.UserID, prototypeType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Code: 500, Message: "查询画像失败"})
		return
	}
	if view == nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": view})
}

// Generate 生成画像 POST /api/v1/twin-portraits/generate
// body: {"type":"photo|chao_xing","photo_base64":"...","photo_mime":"image/jpeg","highlights":"..."}
func (h *TwinPortraitHandler) Generate(c *gin.Context) {
	userCtx := middleware.GetUserContext(c)
	if userCtx == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Code: 401, Message: "未获取到用户信息"})
		return
	}
	var req struct {
		Type        string `json:"type"`
		PhotoBase64 string `json:"photo_base64"`
		PhotoMIME   string `json:"photo_mime"`
		Highlights  string `json:"highlights"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "参数错误"})
		return
	}

	var photoData []byte
	mime := req.PhotoMIME
	if req.PhotoBase64 != "" {
		data, err := decodeBase64(req.PhotoBase64)
		if err != nil {
			c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: "照片 base64 无效"})
			return
		}
		photoData = data
		if mime == "" {
			mime = "image/jpeg"
		}
	}

	extra := service.PortraitPersonalization{
		DisplayName: userCtx.DisplayName,
		Major:       "",
		Role:        userCtx.Role,
		Highlights:  req.Highlights,
	}
	view, err := h.svc.Generate(c.Request.Context(), userCtx.UserID, req.Type, photoData, mime, extra)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Code: 400, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "生成成功", "data": view})
}

// decodeBase64 解码 base64（兼容 data URL 前缀）
func decodeBase64(s string) ([]byte, error) {
	if idx := strings.Index(s, ";base64,"); idx >= 0 {
		s = s[idx+len(";base64,"):]
	}
	return base64.StdEncoding.DecodeString(strings.TrimSpace(s))
}
