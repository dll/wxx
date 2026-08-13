# 反馈管理与 AI 在线修复

## 反馈提交（用户端）

- 入口：对话纠错（chat_page）与全局反馈对话框（feedback_dialog）。
- 字段：类型（回答有误/功能建议/其他）、**所属模块**（下拉，前端 `feedbackModules` 维护）、内容、截图（自动截屏）。
- 提交接口：`POST /api/v1/feedback`（能力 `self.feedback.submit`）。
- **截图佐证（可选）**：仅当截图像素真实可解码时才显示「已截屏」标记（`isDecodableImage` 校验，见 `widgets/feedback_screenshot.dart`），避免"显示已截屏但实际无图"；支持「重新截屏」重试，截图为可选佐证。
- **一键复制反馈数据（用户端）**：两个提交对话框均内置「复制 JSON / 复制报告(提交AI修复)」按钮，将 类型/模块/内容/截图 base64/消息ID 拼装为结构化数据复制到剪贴板，可直接粘贴给 Claude / GLM 等 AI 工具进行修复。实现复用 `utils/feedback_report.dart#buildDraftJson/buildDraftMarkdown`。

## 反馈管理（管理端）

- 入口：我的 → 管理服务 tab →「反馈管理」（`feedback_manage` feature，能力 `union.feedback.list`）。
- 列表/详情/处理：状态流转、回复、满意度、关联知识资源、处理记录。
- 模块字段在列表卡片与详情页展示，便于定位。

## 复制文本与提交 AI 修复

- **部分复制**：列表卡片内容、详情页反馈内容/回复/满意度评论、AI 诊断（OCR/根因/修复建议）均为 `SelectableText`，可长按选区复制。
- **一键复制（全部）**：
  - 详情页右上角「复制完整反馈」：拼装 ID/用户/分类/模块/状态/消息ID/关联资源/内容/回复/满意度/处理记录 → 剪贴板。
  - 列表卡片行内复制图标：复制该条反馈内容。
- **提交 AI 工具修复**：在线修复面板底部「复制完整报告（提交 AI 修复）」按钮，将「反馈信息 + 反馈内容 + OCR + 摘要 + 根因 + 模块 + 代码文件 + 修复建议 + 本机复现指引」拼装为 Markdown 复制到剪贴板，可直接粘贴给 Claude / GLM 等 AI 工具进行修复。
- 实现：`frontend/lib/pages/admin/feedback_page.dart`、`frontend/lib/widgets/feedback_repair.dart`（`Clipboard.setData`，无新增依赖）。

## AI 在线修复（核心）

反馈详情点击「在线修复」→ `POST /api/v1/feedback/:id/ai-repair`（能力 `admin.feedback.write`）。

流程：
1. **截图解析**：复用 `Zhipu4VClient`（GLM-4.6V-Flash）OCR 反馈截图（blob 从 `feedback_screenshots` 读取）。
2. **AI 诊断**：文本模型（DeepSeek/智谱 failover）基于「内容 + 模块 + OCR 文本」输出 JSON：模块、摘要、代码文件、根因、修复建议。
3. **降级**：LLM/视觉不可用时，用本地关键词模块映射（`moduleFilesMap`）兜底返回 `matched_files`。

实现位置：
- 后端：`internal/service/feedback_service.go`（`AIRepair`）、`internal/handler/feedback_handler.go`、路由 `app.go`。
- 前端：`widgets/feedback_repair.dart`、`providers/feedback_provider.dart`（`aiRepair`）。
- 模型配置：`ZHIPU_4V_MODEL`（服务器 `/etc/wxx/env`，现为 `glm-4.6v-flash`）。

## 模块枚举同步

前端 `models.dart#feedbackModules` 与后端 `feedback_service.go#moduleFilesMap/moduleKeywords` 需保持一致；新增模块时两处同步更新。
