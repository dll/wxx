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
const PromptVersion = "1.0"

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
// 分两层：底层为写实人像摄影标准（解剖/光学/物理），上层为卡通动漫蔚小芯风格化。
func BuildPortraitPrompt(prototypeType string, extra PortraitPersonalization) string {
	var sb strings.Builder
	sb.WriteString("创作一幅" + brandStyle() + "数字孪生半身人像插画。\n\n")

	// 原型来源
	if prototypeType == PrototypePhoto {
		sb.WriteString("以参考照片中的人物为原型进行再创作，忠实还原其五官特征与辨识度。\n")
	} else {
		sb.WriteString("原型为一名中国高校学生（姓名" + safe(extra.DisplayName, "同学") + "），" +
			safe(extra.Major, "计算机科学与技术") + "专业，" + roleLabel(extra.Role) + "。\n")
	}

	// 人物特征
	sb.WriteString("人物特征：" + safe(extra.Highlights, "阳光积极、青春向上的在校大学生") + "。\n\n")

	// 底层：写实摄影标准（解剖/光学/物理）
	sb.WriteString("遵循真实人像摄影与人体解剖规则（底层标准）：\n")
	sb.WriteString("- 五官存在生理性轻微不对称，杜绝完全镜像；眼部结构真实，瞳仁有高光反光，非玻璃假眼。\n")
	sb.WriteString("- 皮肤保留原生肌理：可见毛孔、面部细绒毛、唇纹、眼下细纹，贴合年龄，无痕磨皮；禁瓷娃娃无暇肤。\n")
	sb.WriteString("- 姿态重心合理，关节与肌肉形变符合人体运动逻辑；手指姿态自然，无 AI 畸形手。\n")
	sb.WriteString("- 服装随动作产生力学自然褶皱，面料质感真实无塑料感；双脚落地生成准确接触阴影，身体不悬浮。\n")
	sb.WriteString("- 皮肤、毛发、服饰受统一环境光，光影协调、明暗过渡自然。\n")
	sb.WriteString("- 保留适度相机原生噪点、真实镜头景深与轻微实拍曝光偏移；禁全局过度锐化、虚假光滑肌理。\n\n")

	// 上层：蔚小芯特色卡通动漫（融合超星数字人 IP 参考）
	sb.WriteString("在此基础上，以清新柔和的卡通动漫插画风格呈现（蔚小芯特色，参考超星数字人形象气质）：\n")
	sb.WriteString("- 整体氛围明亮温暖、简洁干净、友好可爱，传达快乐与亲和力。\n")
	sb.WriteString("- 头部圆润，眼睛大而有神、目光明亮友善；表情微笑，乐观向上。\n")
	sb.WriteString("- 融入标志性学士帽元素（黑色学士帽或校徽蓝装饰），凸显「有学识的校园智能体」气质。\n")
	sb.WriteString("- 色彩基调以明快暖色为主：阳光黄、校徽蓝（#1565C0）点缀，线条流畅干净。\n")
	sb.WriteString("- 背景简洁留白，可点缀书本、小星星等校园元素，突出前景人物。\n")
	sb.WriteString("- 卡通化仅在光影与线条层面，五官比例与皮肤质感仍遵循上述写实标准，避免过度幼态化。\n")

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
