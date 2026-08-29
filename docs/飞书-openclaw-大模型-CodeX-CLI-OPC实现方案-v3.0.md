# 飞书 → OpenClaw → 大模型 → CodeX CLI → OPC：WXX 实现方案

> 文档状态：已截稿（以本机配置和验收记录为准）
> 更新日期：2026-08-29
> 适用项目：WXX（蔚小芯）
> 敏感信息原则：本文不记录 App Secret、API Key、Token 或其他私密凭据。

## 1. 项目定位

WXX 是面向计算机科学与工程学院（网络空间安全学院）的智能体项目，采用 Flutter 客户端、Go/Gin 后端、Context Engine、Eino 编排和第三方大模型 API。本文只描述 WXX，不包含其他项目或其他项目的运行细节。

## 2. 目标链路

```text
飞书 WXX 机器人
  → OpenClaw Gateway
  → feishu/default
  → leader-wxx
  → gpt-56-sol/gpt-5.6-sol
  → WXX 项目工作区
  → OPC 多 Agent 协作 / CodeX CLI 开发链路
```

## 3. 已核验配置

| 项目 | 结果 |
|---|---|
| Gateway | 单实例，监听 `127.0.0.1:18789` |
| Gateway/CLI 版本 | `2026.7.1-2` |
| Feishu 账号 | `default`，已启用 |
| Feishu binding | `feishu/default → leader-wxx` |
| 主 Agent | `leader-wxx` |
| 主模型 | `gpt-56-sol/gpt-5.6-sol` |
| WXX 工作区 | `E:\2026-2027\2026-2027-1\MyProjects\wxx` |
| Feishu 探针 | `connected, works` |

## 4. OPC 协作边界

WXX Agent 资源包括：

```text
leader-wxx
pm-wxx
dev-refactor-wxx
qa-regression-wxx
reviewer-audit-wxx
```

`leader-wxx` 负责计划、任务拆分、结果汇总和变更控制；子 Agent 按职责执行需求分析、开发、测试和审查。多个 Agent 不得同时修改同一文件。涉及密钥、发布、真实外部联调、数据清理或破坏性操作，必须先人工确认。

## 5. CodeX CLI 使用约束

1. 在 WXX 根目录或明确允许的工作区运行 CodeX CLI。
2. 先阅读项目规范和相关文档，再输出计划。
3. 计划经人工确认后再编码。
4. 编码后执行最小必要的测试、Lint、构建或静态检查。
5. 检查 Git diff，确认无敏感信息、越界文件或无关修改。
6. 保留可回滚提交或配置备份。

CodeX CLI 的实际登录状态、版本和可用模型应以运行 Gateway 的同一 Windows 用户环境现场检查结果为准，不在本文臆造未验证信息。

## 6. 飞书使用与安全

- 私聊按当前账号策略处理；
- 群聊必须明确 @WXX 机器人；
- App Secret、API Key、Token 只保存在本机安全配置或 SecretRefs；
- 不把凭据写入本文、Git、日志或飞书消息；
- 不启动第二个 Gateway。

## 7. 验收标准

```text
Gateway：Connectivity probe: ok
Feishu default：enabled, configured, running, connected, works
路由：feishu/default → leader-wxx
模型：leader-wxx → gpt-56-sol/gpt-5.6-sol
```

消息验收：在 WXX 测试群明确 @机器人，发送只读问题，确认回复来自 `leader-wxx`；代码变更任务必须遵守“阅读 → 计划 → 人工确认 → 编码 → 测试 → diff 检查”流程。

## 8. 版本记录

| 版本 | 日期 | 内容 |
|---|---|---|
| v3.0 | 2026-08-29 | 重建并审核 WXX 单项目方案，统一链路、模型、CodeX CLI 约束和验收标准 |
