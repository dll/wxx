# 飞书-OpenClaw-多模型-多CLI工具-OPC实现方案总结报告

> 形成日期：2026-08-29
> 
> 适用范围：WXX、VOPC、GPPS、EQS、AFC、QFH 多项目独立协作链路

## 一、报告目的

本报告汇总今天形成并落地的多项目方案，统一说明 Feishu、OpenClaw Gateway、项目 Agent、大模型、CLI 工具、workspace 与 OPC 之间的关系，记录已经完成的配置、验证证据、失败根因和后续验收边界。

核心目标不是把所有项目混成一条链路，而是在**一个健康的 OpenClaw Gateway**上，保持每个项目的 Feishu App、`accountId`、route、Agent、模型、workspace、会话和代码仓库彼此独立。

## 二、总体架构

```text
Feishu/<project-account>
        ↓
唯一 OpenClaw Gateway（127.0.0.1:18789）
        ↓
项目独立 leader Agent
        ↓
项目指定 Provider/Model
        ↓
项目独立 workspace 与子 Agent
        ↓
项目 CLI 工具 / OPC / 代码仓库
```

### 统一原则

- Gateway 只允许单实例，所有项目复用既有实例。
- 每个项目使用独立 Feishu `accountId`、App、Agent、workspace、会话和 route。
- Feishu 的 `connected/works` 只能证明通道层状态，不能单独证明 route、模型调用或端到端成功。
- 配置模型、运行模型和会话模型必须分别核验。
- 真实 Secret、API Key、Token 只保存在本机安全配置或平台密钥环境中，不写入文档、Skill、Git、日志和聊天。
- 外部发布、机器人入群、真实群聊、云函数部署、生产部署和密钥轮换需要人工确认。

## 三、今日形成的项目方案

### 1. WXX：蔚小芯

定位：Flutter 客户端 + Go/Gin 后端 + Context Engine + Eino 编排 + 第三方大模型 API。

已形成内容：

- Feishu → OpenClaw → Agent → 模型/CLI → OPC 的总体方案。
- Context Engine、sources 可追溯、结构化检索和 FTS/BM25 等项目约束。
- 项目开发规范、智能体总纲、导出契约、AnswerCard、知识治理和部署文档索引。
- WXX 保持既有 `feishu/default → leader-wxx` 路由，不与其他项目共用运行资源。

### 2. VOPC：Claude 链路

```text
Feishu/claude-vopc → leader-vopc → VOPC workspace → Claude/CLI → OPC
```

已保留独立：

- Feishu account：`claude-vopc`
- Agent：`leader-vopc` 及 VOPC 子 Agent
- VOPC 专属 workspace、会话和 route

待完成：飞书应用发布、机器人入群及真实群聊验收。

### 3. GPPS：Gemini 链路

```text
Feishu/gemini-gpps → leader-gpps → GPPS workspace → Gemini/CLI → OPC
```

已保留独立：

- Feishu account：`gemini-gpps`
- Agent：`leader-gpps` 及 GPPS 子 Agent
- GPPS 专属 workspace、会话和 route
- Provider 使用既有 Gemini 配置

待完成：飞书应用发布、机器人入群及真实群聊验收。

### 4. EQS：DeepSeek 链路

```text
Feishu/deepseek-eqs → leader-eqs → EQS workspace → DeepSeek/CLI → OPC
```

已建立：

- Feishu account：`deepseek-eqs`
- App ID：已建立配置位
- Agent：`leader-eqs`
- 目标模型：`deepseek-openclaw/deepseek-v4-flash`
- 独立 route 和 EQS workspace

安全状态：真实 App Secret 尚未安全注入时，账号必须保持禁用。当前仍需完成 Secret 注入、账号探针、模型调用、应用发布、入群和端到端验收。

### 5. AFC：GLM-5.3-Flash 链路

```text
Feishu/glm-afc → leader-afc → glm-53-flash/glm-5.3-flash → AFC workspace → OPC
```

已完成：

- AFC 独立 Agent：`leader-afc`、`pm-afc`、`dev-refactor-afc`、`qa-regression-afc`、`reviewer-audit-afc`。
- AFC 独立 workspace，位于 `afc/.openclaw/workspaces/`。
- Feishu account 已统一为：`glm-afc`，显示名：`GLM-AFC`。
- route 已统一为：`Feishu/glm-afc → leader-afc`。
- Agent 的 `model` 与 `models` 已统一为：`glm-53-flash/glm-5.3-flash`。
- 已通过新建会话验证实际 Provider/Model：
  - Provider：`glm-53-flash`
  - Model：`glm-5.3-flash`
  - `fallbackUsed=false`
- 已确认 AFC 旧 Feishu 会话曾在配置更新前使用 GPT；不能用新 CLI 会话结果冒充旧会话已刷新。
- 已完成账号命名修正；重载唯一 Gateway 后应以新账号标识和新会话重新验收。

AFC 微信云开发部分：

- 新增受控文本调用云函数：`cloudfunctions/glmChat/`。
- 固定目标模型：`GLM-5.3-Flash`。
- 凭据通过 `GLM_API_KEY`、`GLM_BASE_URL` 环境变量注入。
- 已通过 `node --check`。
- 原 `aiAnalysis` 仍是模拟视频分析，不能宣称已完成真实视频理解接入。

### 6. QFH：Kimi-K3 链路

```text
Feishu/kimi-qfh → leader-qfh → kimi-k3/kimi-k3 → QFH workspace → OPC
```

项目：QFH（快修到家），目录：`E:\2026-2027\2026-2027-1\MyProjects\qfh`。

已完成：

- 生成项目协作规范：`qfh/AGENTS.md`。
- 规范结合 Kimi 长上下文、代码理解和多文件关联分析特点，要求先建立影响范围、结构化输出、事实优先和本地验证。
- 创建独立 Agent：
  - `leader-qfh`
  - `pm-qfh`
  - `dev-refactor-qfh`
  - `qa-regression-qfh`
  - `reviewer-audit-qfh`
- 创建独立 workspace：`qfh/.openclaw/workspaces/<agent-id>`。
- Feishu App ID：`cli_aa1852a26f785be3`。
- Feishu account：`kimi-qfh`。
- 显示名：`KIMI-QFH`。
- route：`Feishu/kimi-qfh → leader-qfh`。
- 模型 Provider/Model：`kimi-k3/kimi-k3`。
- `openclaw config validate` 通过。
- `openclaw agents list` 已确认 5 个 QFH Agent 均使用 `kimi-k3/kimi-k3`。

重要提醒：

- QFH 的 Feishu 账号配置已按用户要求保存为：`"enabled": true`。
- 只有在本机安全配置中的真实 App Secret 有效、应用已发布且机器人已加入目标群后，才能完成真实 Feishu 验收。
- 文档不记录、不回显真实 App Secret。

## 四、通用 Skill 的形成与修订

唯一正式通用 Skill 名称：

```text
feishu-openclaw-agent-model-tool-opc
```

当前 Workshop 提案：

```text
feishu-openclaw-agent-model-tool-opc-20260829-08894b2ed7
```

当前状态：`pending`，扫描：`clean`。

今日修订重点：

1. 先一次性盘点缺项，再执行可安全自动完成的工作。
2. 对真实 Secret、未知 Provider、Base URL、API 类型和模型 ID 一次性收集，不重复索取已知信息。
3. accountId、显示名、route、探针和报告命名必须统一。
4. 模型展示名不能代替 Provider/Model ID。
5. Agent 的 `model`、`models`、route、Gateway 运行态和会话 `model_change` 必须逐层核对。
6. 旧会话保留旧模型时，必须标记“旧会话未刷新”，不得宣称切换成功。
7. 新 CLI 会话成功不能代替旧 Feishu 会话验收。
8. fallback 接管、仅凭模型自报身份或仅凭通道 connected，均不能通过模型验收。
9. 配置修改前备份，按 JSON5 schema 最小增量修改，不使用原生 `JSON.parse`，不整体覆盖配置。
10. 只复用唯一 Gateway，禁止启动第二实例。

旧的项目绑定或混合 Skill 提案未应用，不得将其当作正式通用 Skill 使用；由于 Workshop 审批请求曾过期，仍需在 Workshop UI 中拒绝或隔离。

## 五、典型错误与根因

### AFC 旧 Feishu 会话模型错配

现象：新 CLI 会话返回 GLM，但旧 Feishu 会话仍使用 GPT。

根因：

1. 旧 Feishu 会话在静态配置改为 GLM 之前已经初始化。
2. 会话 JSONL 首条 `model_change` 已记录 GPT。
3. 后续 Feishu 消息沿用会话级模型状态。
4. 修改磁盘配置不会自动覆盖 Gateway 内存中的既有会话上下文。
5. 因此新 CLI 会话和旧 Feishu 会话属于不同验收对象。

正确做法：

- 重载或重启现有唯一 Gateway，使其加载新配置。
- 不启动第二 Gateway。
- 重启后新建或刷新目标 Feishu 会话。
- 检查会话首条 `model_change`、实际响应元数据、请求日志和 fallback 状态。
- 在证据齐全前，不宣称旧会话已切换。

### 账号名称遗漏

现象：AFC 模型已改为 GLM，但 Feishu 仍显示 `afc`。

根因：只修改了 Agent/模型，没有同步修改 Feishu accountId、显示名和 route。

改进：通用 Skill v3 增加命名一致性和旧命名残留核验，要求完整迁移 accountId、显示名、route、探针和报告。

### 模型展示名与模型 ID 混淆

现象：Kimi 显示名为 Kimi-K3，但历史配置中存在 `kimi-k3/kimi-k2.7-code`，与 Provider 实际登记的 `kimi-k3` 不一致。

改进：QFH 使用已核验的完整引用：

```text
kimi-k3/kimi-k3
```

以后必须同时核对 Provider、模型 ID、Agent `model` 和 Agent `models`。

## 六、配置、备份与验证记录

OpenClaw 主配置：

```text
C:\Users\ldl\.openclaw\openclaw.json
```

今日相关备份包括：

```text
openclaw.json.bak-afc-20260829-214235
openclaw.json.bak-afc-agent-20260829-220204
openclaw.json.bak-afc-glm-20260829-221648
openclaw.json.bak-glm-afc-20260829-225942
openclaw.json.bak-qfh-20260829-230605
openclaw.json.bak-qfh-report-20260829-233456
```

已完成的关键检查：

```text
openclaw config validate                         通过
openclaw agents list                             QFH 5 个 Agent 模型正确
node --check afc/cloudfunctions/glmChat/index.js 通过
AFC 新会话实际模型探针                         GLM 成功，fallbackUsed=false
```

## 七、当前状态总表

| 项目 | Feishu account | Agent | 目标模型 | 配置 | 运行/端到端验收 |
|---|---|---|---|---|---|
| WXX | `default` | `leader-wxx` | 既有 WXX 模型链路 | 已有 | 按既有方案维护 |
| VOPC | `claude-vopc` | `leader-vopc` | Claude | 已有 | 发布、入群、E2E 待完成 |
| GPPS | `gemini-gpps` | `leader-gpps` | Gemini | 已有 | 发布、入群、E2E 待完成 |
| EQS | `deepseek-eqs` | `leader-eqs` | DeepSeek V4 Flash | 已建立占位 | Secret、发布、入群、E2E 待完成 |
| AFC | `glm-afc` | `leader-afc` | GLM-5.3-Flash | 已配置 | 新会话已验证；旧 Feishu 会话需重载后复验 |
| QFH | `kimi-qfh` | `leader-qfh` | Kimi-K3 | 已配置并启用 | 需平台发布、入群和真实消息验收 |

## 八、后续执行顺序

1. 确认唯一 Gateway 重新加载最新配置；不启动第二实例。
2. 重新验收 `Feishu/glm-afc → leader-afc → GLM-5.3-Flash`。
3. 重新验收 `Feishu/kimi-qfh → leader-qfh → Kimi-K3`。
4. 完成 QFH Feishu 应用发布、机器人入群和真实群聊探针。
5. 完成 VOPC、GPPS 的发布、入群和真实群聊验收。
6. 安全注入 EQS Secret 后完成 DeepSeek 探针和端到端验收。
7. 在人工确认后部署 AFC `glmChat` 云函数并验证异常场景。
8. 复现并处理 AFC 既有视频分类属性测试失败，再重跑完整测试。
9. 审核并应用唯一通用 Skill；拒绝或隔离旧项目绑定提案。
10. 持续区分六类状态：配置存在、通道连接、route 命中、模型调用、会话刷新、端到端成功。

## 九、结论

今天已经形成一套可复用但不绑定项目的多模型、多 CLI 工具、OPC 协作方法：

- **通用在流程，独立在资源**；
- **配置成功不等于运行成功**；
- **新会话成功不等于旧会话刷新**；
- **通道连接不等于路由和模型验收**；
- **所有外部动作和敏感配置必须保留人工确认边界**。

当前 QFH 的配置、Agent、workspace、route 和模型引用已建立，账号配置已保存为 `enabled: true`；QFH 的真实 Feishu 端到端结果仍以应用发布、入群和实际消息探针为准。
