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
const PromptVersion = "2.0"

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
// 风格定位：Q 版可爱精灵数字人（参考超星数字人），明亮清新、亲和讨喜，
// 学生普遍喜爱；舍弃写实逼真路线。
func BuildPortraitPrompt(prototypeType string, extra PortraitPersonalization) string {
	var sb strings.Builder
	sb.WriteString("创作一幅" + brandStyle() + "Q 版可爱数字人半身像插画，风格参考超星数字人（卡通 3D、Q 萌可爱）。\n\n")

	// 原型来源
	if prototypeType == PrototypePhoto {
		sb.WriteString("以参考照片中的人物为原型进行二次创作，保留其发型、发色、脸型轮廓等主要辨识特征，但整体化为 Q 版可爱形象。\n")
	} else {
		sb.WriteString("原型为一名中国高校学生（姓名" + safe(extra.DisplayName, "同学") + "），" +
			safe(extra.Major, "计算机科学与技术") + "专业，" + roleLabel(extra.Role) + "。\n")
	}

	// 人物特征
	sb.WriteString("人物气质：" + safe(extra.Highlights, "阳光积极、青春向上的在校大学生") + "。\n\n")

	// 核心：Q 版可爱精灵造型
	sb.WriteString("造型要求（Q 版可爱精灵风）：\n")
	sb.WriteString("- 大头小身比例（Q 版约 3~4 头身），整体圆润饱满，萌态十足。\n")
	sb.WriteString("- 大而圆的眼睛，亮晶晶有神，眼中有高光星点；脸颊自带可爱腮红；表情甜美微笑。\n")
	sb.WriteString("- 圆润的娃娃脸、小巧的鼻子与嘴巴，整体Q萌可爱，亲和力强，学生一看就喜欢。\n")
	sb.WriteString("- 保留标志性学士帽（黑色或校徽蓝装饰），帽檐下露出俏皮发丝，凸显「聪明小博士」气质。\n")
	sb.WriteString("- 纯净通透的皮肤（卡通质感，不追求写实毛孔），线条圆滑干净，无 AI 畸形手指。\n\n")

	// 上色与背景
	sb.WriteString("上色与背景（明亮清新）：\n")
	sb.WriteString("- 色彩明快鲜亮：主色校徽蓝 #1565C0 + 暖阳黄点缀，饱和度适中，干净不刺眼。\n")
	sb.WriteString("- 背景简洁梦幻：柔和渐变 + 漂浮的小星星、书本、光点等校园元素，不喧宾夺主。\n")
	sb.WriteString("- 整体呈精致 3D 卡通渲染质感（参考超星数字人），光影柔和、边缘干净，如高品质游戏立绘。\n\n")

	// 精灵感点缀
	sb.WriteString("精灵感点缀：\n")
	sb.WriteString("- 发梢与衣角点缀微光粒子与小星光，仿佛自带魔法光效，灵动梦幻。\n")
	sb.WriteString("- 整体元气满满、积极向上的校园精灵气质，温暖治愈。\n")

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
