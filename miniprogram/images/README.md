# 图标资源

本目录需要放置以下图标文件（推荐 40x40 PNG，单色/线性风格）：

## Tab 图标（常态 + 选中态，共 8 个）

| 文件名 | 用途 |
|--------|------|
| tab-chat.png | 问答 Tab（常态） |
| tab-chat-active.png | 问答 Tab（选中） |
| tab-knowledge.png | 知识大厅 Tab（常态） |
| tab-knowledge-active.png | 知识大厅 Tab（选中） |
| tab-process.png | 流程指南 Tab（常态） |
| tab-process-active.png | 流程指南 Tab（选中） |
| tab-profile.png | 我的 Tab（常态） |
| tab-profile-active.png | 我的 Tab（选中） |

## 其他图标

| 文件名 | 用途 | 建议尺寸 |
|--------|------|----------|
| logo.png | 登录页品牌 Logo | 200x200 |
| avatar-default.png | 用户默认头像 | 120x120 |
| avatar-ai.png | AI 助手头像 | 120x120 |

## 图标获取方式

1. 微信官方图标库：https://developers.weixin.qq.com/miniprogram/design/
2. Iconfont：https://www.iconfont.cn/（搜索关键词：聊天、知识、流程、用户）
3. 自行设计：使用 Figma / Sketch，导出为 PNG

## 临时方案

在图标资源就位前，tabBar 图标可在 `app.json` 中临时移除 `iconPath` 和 `selectedIconPath` 字段，
小程序将仅显示文字标签。
