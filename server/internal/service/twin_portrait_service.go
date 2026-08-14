package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/dll/wxx/server/internal/llm"
	"github.com/dll/wxx/server/internal/model"
	"github.com/dll/wxx/server/internal/repository"
)

// PromptVersion 画像提示词版本（升级后前端可触发重新生成）
const PromptVersion = "3.0"

// PortraitPrototypeType 原型类型
const (
	PrototypePhoto    = "photo"
	PrototypeChaoXing = "chao_xing"
)

// TwinPortraitService 数字孪生画像生成服务
// 以用户照片为原型（图生图）或内置超星原型（文生图），
// 按「电影角色视觉总监 + 专业人像摄影师 + 图像质检师」标准生成
// 蔚小芯特色卡通动漫画像。
type TwinPortraitService struct {
	repo    *repository.TwinPortraitRepo
	imgGen  llm.ImageGenClient
	maxSize int // 原型图/生成图 base64 上限(字节)
}

// NewTwinPortraitService 创建画像服务
func NewTwinPortraitService(repo *repository.TwinPortraitRepo, imgGen llm.ImageGenClient) *TwinPortraitService {
	return &TwinPortraitService{
		repo:    repo,
		imgGen:  imgGen,
		maxSize: 8 << 20, // 8MB
	}
}

// ListPortraits 列出用户画像
func (s *TwinPortraitService) ListPortraits(userID int64) ([]*model.TwinPortraitView, error) {
	list, err := s.repo.ListByUser(userID)
	if err != nil {
		return nil, err
	}
	views := make([]*model.TwinPortraitView, 0, len(list))
	for _, p := range list {
		views = append(views, toPortraitView(p))
	}
	return views, nil
}

// GetPortrait 查询指定类型画像
func (s *TwinPortraitService) GetPortrait(userID int64, prototypeType string) (*model.TwinPortraitView, error) {
	p, err := s.repo.GetByUserAndType(userID, prototypeType)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, nil
	}
	return toPortraitView(p), nil
}

// Generate 生成画像。
// photoData: 用户照片(可选)；prototypeType: photo 需照片，chao_xing 用内置原型描述。
// extra: 用户画像特征（姓名/角色/五维亮点），用于个性化提示词。
func (s *TwinPortraitService) Generate(ctx context.Context, userID int64, prototypeType string, photoData []byte, mime string, extra PortraitPersonalization) (*model.TwinPortraitView, error) {
	if s.imgGen == nil {
		return nil, fmt.Errorf("文生图服务未配置")
	}
	if prototypeType != PrototypePhoto && prototypeType != PrototypeChaoXing {
		return nil, fmt.Errorf("不支持的画像类型: %s", prototypeType)
	}
	if prototypeType == PrototypePhoto && len(photoData) == 0 {
		return nil, fmt.Errorf("照片模式必须上传原型照片")
	}

	// 生成提示词：写实摄影标准(底层) + 卡通动漫蔚小芯风(上层)
	prompt := BuildPortraitPrompt(prototypeType, extra)

	var refData []byte
	var refMime string
	var srcPhotoB64 string
	if prototypeType == PrototypePhoto {
		refData = photoData
		refMime = mime
		srcPhotoB64 = base64.StdEncoding.EncodeToString(photoData)
	}

	imageData, err := s.imgGen.Generate(ctx, prompt, refData, refMime)
	if err != nil {
		return nil, fmt.Errorf("画像生成失败: %w", err)
	}
	if len(imageData) == 0 {
		return nil, fmt.Errorf("画像生成结果为空")
	}
	if len(imageData) > s.maxSize {
		return nil, fmt.Errorf("生成图片过大")
	}

	p := &model.TwinPortrait{
		UserID:            userID,
		PrototypeType:     prototypeType,
		PromptVersion:     PromptVersion,
		ImageBase64:       base64.StdEncoding.EncodeToString(imageData),
		ImageMIME:         "image/png",
		SourcePhotoBase64: srcPhotoB64,
	}
	if _, err := s.repo.Upsert(p); err != nil {
		return nil, fmt.Errorf("画像保存失败: %w", err)
	}
	return toPortraitView(p), nil
}

// PortraitPersonalization 画像个性化参数
type PortraitPersonalization struct {
	DisplayName string
	Major       string
	Role        string
	// 五维亮点描述，如 "学业优秀、社交活跃"
	Highlights string
}

// BuildPortraitPrompt 构建画像提示词。
// 策略：CogView 对过长提示词容易忽略细节，精简为 3 段式：
// 风格定义 → 人物描述 → 画面元素，总计 ≤200 字。
func BuildPortraitPrompt(prototypeType string, extra PortraitPersonalization) string {
	var sb strings.Builder

	// 第一段：风格（最关键，放最前面）
	sb.WriteString("Q版卡通可爱风格半身像，大头小身3头身比例，超星数字人风格，明亮3D渲染，圆润Q萌。\n")

	// 第二段：人物
	if prototypeType == PrototypePhoto {
		sb.WriteString("以参考照片人物为原型，保留发型、发色、脸型等主要特征，转化为Q版可爱形象。")
	} else {
		name := safe(extra.DisplayName, "同学")
		major := safe(extra.Major, "计算机科学")
		sb.WriteString(name + "，" + major + "专业学生，")
		role := roleLabel(extra.Role)
		if role == "在校学生" {
			sb.WriteString(role + "，")
		}
	}
	highlights := safe(extra.Highlights, "阳光青春")
	sb.WriteString(highlights + "，甜美微笑，大眼睛亮晶晶，腮红可爱。\n")

	// 第三段：画面元素
	sb.WriteString("学士帽蓝色装饰，背景柔和渐变，星星书本点缀，发梢微光粒子，精致3D卡通质感，高清。")

	return sb.String()
}

func brandStyle() string {
	return "「蔚小芯」校园 AI 智能体"
}

func roleLabel(role string) string {
	switch role {
	case "teacher":
		return "教师"
	case "counselor":
		return "辅导员"
	case "student_union":
		return "学生会成员"
	case "college_admin", "school_admin", "sys_admin":
		return "管理人员"
	default:
		return "在校学生"
	}
}

func safe(s, def string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return def
	}
	return s
}

func toPortraitView(p *model.TwinPortrait) *model.TwinPortraitView {
	return &model.TwinPortraitView{
		ID:            p.ID,
		UserID:        p.UserID,
		PrototypeType: p.PrototypeType,
		PromptVersion: p.PromptVersion,
		ImageBase64:   p.ImageBase64,
		ImageMIME:     p.ImageMIME,
		HasPhoto:      p.SourcePhotoBase64 != "",
		CreatedAt:     p.CreatedAt,
	}
}
