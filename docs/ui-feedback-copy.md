# 任务方案 — 反馈管理：复制文本 + 提交 AI 工具修复

> 版本：v1.0 · 2026-08-11

## 背景与目标

反馈管理（`frontend/lib/pages/admin/feedback_page.dart` + `widgets/feedback_repair.dart`）目前文本不可选中复制，
管理员处理反馈时需要手动摘录内容或截屏。目标：

1. **复制文本（全部/部分）**：反馈列表卡片、详情页反馈内容、回复、AI 诊断各文本支持选中复制；
   提供「复制完整反馈」一键复制全部字段。
2. **提交 AI 工具修复**：一键生成完整 Markdown 诊断报告（反馈信息 + 截图 OCR + 根因 + 代码文件 + 修复建议 + 复现指引），
   复制到剪贴板后可直接粘贴给 Claude/GLM 等 AI 工具进行修复。

## 范围

### 做
- `feedback_page.dart`：
  - `_FeedbackCard` 内容改 `SelectableText`（部分复制）。
  - `FeedbackDetailPage` 反馈内容、回复、满意度评论改 `SelectableText`。
  - 详情页顶部新增「复制完整反馈」按钮：拼装 ID/用户/分类/模块/状态/内容/回复/资源/处理记录 → 剪贴板。
- `feedback_repair.dart`（在线修复面板）：
  - 诊断文本区（OCR/根因/修复建议/代码文件/复现指引）改 `SelectableText`。
  - 新增「复制完整报告」按钮：生成 Markdown（含反馈信息 + AI 诊断 + 代码定位 + 复现步骤），复制到剪贴板，供提交 AI 工具修复。
  - 复制成功用 SnackBar 提示。
- `ui-feedback.md` 文档登记新交互。

### 不做
- 不改后端接口（纯前端交互增量）。
- 不做「直接调 AI 工具写代码」的联调（仅提供可复制的完整上下文文本）。

## 技术要点
- 剪贴板：沿用现有 `Clipboard.setData(ClipboardData(text: ...))`（见 chat_page.dart:738）。
- 无新增依赖、无 schema 变更、无后端改动。
- 组装报告用 Dart 字符串（`///` 聚合字段 + 代码文件列表）。

## 步骤拆分
1. `feedback_page.dart` 卡片/详情文本 SelectableText +「复制完整反馈」按钮。
2. `feedback_repair.dart` 文本 SelectableText +「复制完整报告」按钮。
3. `flutter analyze` + 测试 + 构建验证。
4. 文档登记 + 提交。

## 验收标准
- 反馈内容可整段/选区复制；「复制完整反馈」将全部字段复制到剪贴板。
- 在线修复面板「复制完整报告」生成含反馈 + 诊断 + 代码文件 + 复现步骤的 Markdown。
- `flutter analyze` 0 error / 0 warning；`flutter test` 通过；Web 构建成功。

## 回滚与检查点
- 纯前端改动，Git 单次提交即可回滚。