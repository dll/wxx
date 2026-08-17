# OpenClaw‑3项目流水线部署脚本合集
> 更新：WXX路径已调整为 `E:\2026-2027\2026-2027-1\MyProjects\wxx`
> 项目清单：WXX / TSLOMS / EQS
> 版本：OpenClaw 2026.7.1-2

## 1. Setup‑3Projects‑OpenClaw.ps1（一键创建全部workspace目录）
```powershell
<#
OpenClaw 3‑项目流水线团队目录生成脚本
项目：WXX, TSLOMS, EQS
更新后 WXX路径：E:\2026-2027\2026-2027-1\MyProjects\wxx
#>

# -------- WXX --------
$wxxRoot = "E:\2026-2027\2026-2027-1\MyProjects\wxx\.openclaw\workspaces"
New-Item -Path "$wxxRoot\leader-wxx" -ItemType Directory -Force | Out-Null
New-Item -Path "$wxxRoot\pm-wxx" -ItemType Directory -Force | Out-Null
New-Item -Path "$wxxRoot\dev-refactor-wxx" -ItemType Directory -Force | Out-Null
New-Item -Path "$wxxRoot\qa-regression-wxx" -ItemType Directory -Force | Out-Null
New-Item -Path "$wxxRoot\reviewer-audit-wxx" -ItemType Directory -Force | Out-Null

# -------- TSLOMS --------
$tslomsRoot = "E:\2026-2027\2026-2027-1\MyProjects\tsloms\.openclaw\workspaces"
New-Item -Path "$tslomsRoot\leader-tsloms" -ItemType Directory -Force | Out-Null
New-Item -Path "$tslomsRoot\pm-tsloms" -ItemType Directory -Force | Out-Null
New-Item -Path "$tslomsRoot\dev-refactor-tsloms" -ItemType Directory -Force | Out-Null
New-Item -Path "$tslomsRoot\qa-regression-tsloms" -ItemType Directory -Force | Out-Null
New-Item -Path "$tslomsRoot\reviewer-audit-tsloms" -ItemType Directory -Force | Out-Null

# -------- EQS --------
$eqsRoot = "E:\2026-2027\2026-2027-1\MyProjects\eqs\.openclaw\workspaces"
New-Item -Path "$eqsRoot\leader-eqs" -ItemType Directory -Force | Out-Null
New-Item -Path "$eqsRoot\pm-eqs" -ItemType Directory -Force | Out-Null
New-Item -Path "$eqsRoot\dev-refactor-eqs" -ItemType Directory -Force | Out-Null
New-Item -Path "$eqsRoot\qa-regression-eqs" -ItemType Directory -Force | Out-Null
New-Item -Path "$eqsRoot\reviewer-audit-eqs" -ItemType Directory -Force | Out-Null

Write-Host "✅ 全部 15 个 workspace 文件夹创建完成"
```

## 2. Write‑AgentsMd.ps1（自动写入三份AGENTS.md调度文件）
```powershell
#==== WXX AGENTS.md ====
$wxxAgentPath = "E:\2026-2027\2026-2027-1\MyProjects\wxx\.openclaw\workspaces\leader-wxx\AGENTS.md"
$wxxContent = @'
# WXX项目重构流水线‑多Agent协作规则
##团队成员
1. pm‑wxx：需求核对专员，只读，输出pm‑checklist.md
2. dev‑refactor‑wxx：重构开发专员，读写项目源码，输出refactor‑notes.md
3. qa‑regression‑wxx：回归测试专员，执行测试，输出qa‑report.md
4. reviewer‑audit‑wxx：代码审核专员，只读评审源码，输出audit‑report.md

#流水线顺序（串行协作）
当收到指令【启动WXX项目重构优化任务】，你（leader‑wxx）严格按下面流水线分步调度子Agent，上一步完成再执行下一步：
步骤1：spawn pm‑wxx，任务：读取本项目原有代码与需求，核对本次重构优化范围，输出重构核对清单pm‑checklist.md，不要修改代码。等待pm‑wxx完成。
步骤2：spawn dev‑refactor‑wxx，任务：基于pm‑checklist.md，对项目代码重构优化；改善代码结构、性能、可读性；不改变原有业务逻辑；输出refactor‑notes.md重构改动记录；等待开发完成。
步骤3：spawn qa‑regression‑wxx，任务：读取重构后的代码，编写回归测试，验证原有业务功能没有退化；输出qa‑report.md；等待测试完成。
步骤4：spawn reviewer‑audit‑wxx，任务：评审重构代码，检查代码质量、潜在风险；输出audit‑report.md；禁止修改任何源码；等待评审完成。
步骤5：汇总四份文档，生成refactor‑final‑summary.md，结束流水线任务。

#禁止
1.不要并行一次性启动4个子Agent；严格流水线先后顺序。
2.子Agent不能互相直接发起spawn；全部调度指令只能由leader‑wxx发出。
'@
Set-Content -Path $wxxAgentPath -Value $wxxContent -Encoding utf8


#==== TSLOMS AGENTS.md ====
$tslomsAgentPath = "E:\2026-2027\2026-2027-1\MyProjects\tsloms\.openclaw\workspaces\leader-tsloms\AGENTS.md"
$tslomsContent = @'
# TSLOMS项目重构流水线‑多Agent协作规则
##团队成员
1. pm‑tsloms：需求核对专员，只读，输出pm‑checklist.md
2. dev‑refactor‑tsloms：重构开发专员，读写项目源码，输出refactor‑notes.md
3. qa‑regression‑tsloms：回归测试专员，执行测试，输出qa‑report.md
4. reviewer‑audit‑tsloms：代码审核专员，只读评审源码，输出audit‑report.md

#流水线顺序（串行协作）
当收到指令【启动TSLOMS项目重构优化任务】，你（leader‑tsloms）严格按下面流水线分步调度子Agent，上一步完成再执行下一步：
步骤1：spawn pm‑tsloms，任务：读取本项目原有代码与需求，核对本次重构优化范围，输出重构核对清单pm‑checklist.md，不要修改代码。等待pm‑tsloms完成。
步骤2：spawn dev‑refactor‑tsloms，任务：基于pm‑checklist.md，对项目代码重构优化；改善代码结构、性能、可读性；不改变原有业务逻辑；输出refactor‑notes.md重构改动记录；等待开发完成。
步骤3：spawn qa‑regression‑tsloms，任务：读取重构后的代码，编写回归测试，验证原有业务功能没有退化；输出qa‑report.md；等待测试完成。
步骤4：spawn reviewer‑audit‑tsloms，任务：评审重构代码，检查代码质量、潜在风险；输出audit‑report.md；禁止修改任何源码；等待评审完成。
步骤5：汇总四份文档，生成refactor‑final‑summary.md，结束流水线任务。

#禁止
1.不要并行一次性启动4个子Agent；严格流水线先后顺序。
2.子Agent不能互相直接发起spawn；全部调度指令只能由leader‑tsloms发出。
'@
Set-Content -Path $tslomsAgentPath -Value $tslomsContent -Encoding utf8


#==== EQS AGENTS.md ====
$eqsAgentPath = "E:\2026-2027\2026-2027-1\MyProjects\eqs\.openclaw\workspaces\leader-eqs\AGENTS.md"
$eqsContent = @'
# EQS项目重构流水线‑多Agent协作规则
##团队成员
1. pm‑eqs：需求核对专员，只读，输出pm‑checklist.md
2. dev‑refactor‑eqs：重构开发专员，读写项目源码，输出refactor‑notes.md
3. qa‑regression‑eqs：回归测试专员，执行测试，输出qa‑report.md
4. reviewer‑audit‑eqs：代码审核专员，只读评审源码，输出audit‑report.md

#流水线顺序（串行协作）
当收到指令【启动EQS项目重构优化任务】，你（leader‑eqs）严格按下面流水线分步调度子Agent，上一步完成再执行下一步：
步骤1：spawn pm‑eqs，任务：读取本项目原有代码与需求，核对本次重构优化范围，输出重构核对清单pm‑checklist.md，不要修改代码。等待pm‑eqs完成。
步骤2：spawn dev‑refactor‑eqs，任务：基于pm‑checklist.md，对项目代码重构优化；改善代码结构、性能、可读性；不改变原有业务逻辑；输出refactor‑notes.md重构改动记录；等待开发完成。
步骤3：spawn qa‑regression‑eqs，任务：读取重构后的代码，编写回归测试，验证原有业务功能没有退化；输出qa‑report.md；等待测试完成。
步骤4：spawn reviewer‑audit‑eqs，任务：评审重构代码，检查代码质量、潜在风险；输出audit‑report.md；禁止修改任何源码；等待评审完成。
步骤5：汇总四份文档，生成refactor‑final‑summary.md，结束流水线任务。

#禁止
1.不要并行一次性启动4个子Agent；严格流水线先后顺序。
2.子Agent不能互相直接发起spawn；全部调度指令只能由leader‑eqs发出。
'@
Set-Content -Path $eqsAgentPath -Value $eqsContent -Encoding utf8

Write-Host "✅ 三份AGENTS.md文件写入完成"
```

## 3. openclaw.json（全局网关完整配置）
> 文件存放路径：`C:\Users\ldl\.openclaw\openclaw.json`
>
> ⚠️ **字段校验说明**：本版本（2026.7.1-2）JSON schema 校验严格，必须使用 `tools.profile`
> 字段（取值 `minimal` / `coding` / `messaging` / `full`）。旧写法 `toolProfile` 会导致
> `agents.list.N: Invalid input` 校验失败、网关拒绝启动。
> 不得在 agent item 顶层写 `provider`（该字段仅存在于 `models.<id>` 内部）。
```json
{
  "gateway": {
    "mode": "local",
    "port": 18789
  },
  "models": {
    "mode": "merge",
    "providers": {
      "deepseek-openclaw": {
        "baseUrl": "https://api.deepseek.com/v1",
        "apiKey": "<YOUR_DEEPSEEK_API_KEY>",
        "api": "openai-completions",
        "models": [
          { "id": "deepseek-v4-pro", "name": "DeepSeek V4 Pro",
            "cost": { "input": 1.68, "output": 3.36 }, "contextWindow": 1000000 },
          { "id": "deepseek-v4-flash", "name": "DeepSeek V4 Flash",
            "cost": { "input": 0.14, "output": 0.28 }, "contextWindow": 1000000 }
        ]
      }
    }
  },
  "agents": {
    "defaults": {
      "models": {
        "deepseek-openclaw/deepseek-v4-pro": { "alias": "Pro" },
        "deepseek-openclaw/deepseek-v4-flash": { "alias": "Flash" }
      },
      "model": {
        "primary": "deepseek-openclaw/deepseek-v4-flash",
        "fallbacks": [ "deepseek-openclaw/deepseek-v4-pro" ]
      }
    },
    "list": [
      {
        "id": "leader-wxx",
        "workspace": "E:\\2026-2027\\2026-2027-1\\MyProjects\\wxx\\.openclaw\\workspaces\\leader-wxx",
        "subagents": {
          "allowAgents": ["pm-wxx","dev-refactor-wxx","qa-regression-wxx","reviewer-audit-wxx"]
        },
        "tools": { "profile": "coding" }
      },
      {
        "id": "pm-wxx",
        "workspace": "E:\\2026-2027\\2026-2027-1\\MyProjects\\wxx\\.openclaw\\workspaces\\pm-wxx",
        "tools": { "profile": "coding" }
      },
      {
        "id": "dev-refactor-wxx",
        "workspace": "E:\\2026-2027\\2026-2027-1\\MyProjects\\wxx\\.openclaw\\workspaces\\dev-refactor-wxx",
        "tools": { "profile": "coding" }
      },
      {
        "id": "qa-regression-wxx",
        "workspace": "E:\\2026-2027\\2026-2027-1\\MyProjects\\wxx\\.openclaw\\workspaces\\qa-regression-wxx",
        "tools": { "profile": "coding" }
      },
      {
        "id": "reviewer-audit-wxx",
        "workspace": "E:\\2026-2027\\2026-2027-1\\MyProjects\\wxx\\.openclaw\\workspaces\\reviewer-audit-wxx",
        "tools": { "profile": "coding" }
      },
      {
        "id": "leader-tsloms",
        "workspace": "E:\\2026-2027\\2026-2027-1\\MyProjects\\tsloms\\.openclaw\\workspaces\\leader-tsloms",
        "subagents": {
          "allowAgents": ["pm-tsloms","dev-refactor-tsloms","qa-regression-tsloms","reviewer-audit-tsloms"]
        },
        "tools": { "profile": "coding" }
      },
      {
        "id": "pm-tsloms",
        "workspace": "E:\\2026-2027\\2026-2027-1\\MyProjects\\tsloms\\.openclaw\\workspaces\\pm-tsloms",
        "tools": { "profile": "coding" }
      },
      {
        "id": "dev-refactor-tsloms",
        "workspace": "E:\\2026-2027\\2026-2027-1\\MyProjects\\tsloms\\.openclaw\\workspaces\\dev-refactor-tsloms",
        "tools": { "profile": "coding" }
      },
      {
        "id": "qa-regression-tsloms",
        "workspace": "E:\\2026-2027\\2026-2027-1\\MyProjects\\tsloms\\.openclaw\\workspaces\\qa-regression-tsloms",
        "tools": { "profile": "coding" }
      },
      {
        "id": "reviewer-audit-tsloms",
        "workspace": "E:\\2026-2027\\2026-2027-1\\MyProjects\\tsloms\\.openclaw\\workspaces\\reviewer-audit-tsloms",
        "tools": { "profile": "coding" }
      },
      {
        "id": "leader-eqs",
        "workspace": "E:\\2026-2027\\2026-2027-1\\MyProjects\\eqs\\.openclaw\\workspaces\\leader-eqs",
        "subagents": {
          "allowAgents": ["pm-eqs","dev-refactor-eqs","qa-regression-eqs","reviewer-audit-eqs"]
        },
        "tools": { "profile": "coding" }
      },
      {
        "id": "pm-eqs",
        "workspace": "E:\\2026-2027\\2026-2027-1\\MyProjects\\eqs\\.openclaw\\workspaces\\pm-eqs",
        "tools": { "profile": "coding" }
      },
      {
        "id": "dev-refactor-eqs",
        "workspace": "E:\\2026-2027\\2026-2027-1\\MyProjects\\eqs\\.openclaw\\workspaces\\dev-refactor-eqs",
        "tools": { "profile": "coding" }
      },
      {
        "id": "qa-regression-eqs",
        "workspace": "E:\\2026-2027\\2026-2027-1\\MyProjects\\eqs\\.openclaw\\workspaces\\qa-regression-eqs",
        "tools": { "profile": "coding" }
      },
      {
        "id": "reviewer-audit-eqs",
        "workspace": "E:\\2026-2027\\2026-2027-1\\MyProjects\\eqs\\.openclaw\\workspaces\\reviewer-audit-eqs",
        "tools": { "profile": "coding" }
      }
    ]
  },
  "tools": {
    "sessions": {
      "visibility": "all"
    },
    "agentToAgent": {
      "enabled": true,
      "allow": [
        "leader-wxx", "pm-wxx", "dev-refactor-wxx", "qa-regression-wxx", "reviewer-audit-wxx",
        "leader-tsloms", "pm-tsloms", "dev-refactor-tsloms", "qa-regression-tsloms", "reviewer-audit-tsloms",
        "leader-eqs", "pm-eqs", "dev-refactor-eqs", "qa-regression-eqs", "reviewer-audit-eqs"
      ]
    }
  },
  "runtime": {
    "default_timeout": 3600
  }
}
```

> ⚠️ **多 Agent 协作必需项**（缺失会导致流水线无法启动）：
> - `tools.sessions.visibility = "all"`：允许子 Agent 会话互相可见，
>   否则 leader 的 `sessions_send` 会被拒绝。
> - `tools.agentToAgent.enabled = true` + `allow`：开启 Agent 间互调工具
>   （leader spawn 子 Agent 依赖此能力）。
> - **子 Agent 不要用 `tools.profile: "minimal"`**（2026-08-16 实测根因）：
>   `minimal` 仅含 `session_status` 一个工具，子 Agent 启动时无任何可调用工具，
>   报错 `No callable tools remain after resolving explicit tool allowlist`。
>   修复：把 pm / reviewer 等子 Agent 的 profile 改为 `coding`
>   （与 dev-refactor / qa-regression 一致，获得文件/运行时工具）。
>   注意：tsloms、eqs 的 pm/reviewer 也需一并改为 `coding`，否则同样无法启动。
> - 若用 `tools.allow` 显式清单，确保包含 `read` / `write` / `exec` 等实际可调用工具，
>   不要写空/错 allow。

## 4. 部署执行步骤
```powershell
# 步骤1：创建全部目录树
.\Setup‑3Projects‑OpenClaw.ps1

# 步骤2：写入三份AGENTS.md调度规则
.\Write‑AgentsMd.ps1

# 步骤3：合并替换 C:\Users\ldl\.openclaw\openclaw.json
#   - 保留现有 models / agents.defaults（DeepSeek），只改 agents.list
#   - 覆盖前先备份：Copy-Item C:\Users\ldl\.openclaw\openclaw.json C:\Users\ldl\.openclaw\openclaw.json.bak

# 步骤4：重启网关，加载所有Agent（注意：不支持 --allow-unconfigured 参数）
openclaw gateway stop
openclaw gateway start
```

## 5. 自动验证脚本（Verify‑OpenClawDeploy.ps1）
> 校验配置合法性 → 等待网关就绪 → 确认模型加载 → 列出 Agent。
```powershell
# ===== 自动验证：部署是否成功 =====
$ErrorActionPreference = "Stop"

Write-Host "== 1/4 校验配置合法性 =="
$v = openclaw config validate 2>&1 | Out-String
if ($v -match "Invalid config|invalid|error") { Write-Host "❌ 配置校验失败:`n$v" } else { Write-Host "✅ 配置合法" }

Write-Host "== 2/4 重启网关 =="
openclaw gateway stop 2>&1 | Out-Null
Start-Sleep -Seconds 3
openclaw gateway start 2>&1 | Out-Null

Write-Host "== 3/4 等待网关就绪（最长120秒）=="
$ready = $false
for ($i = 0; $i -lt 24; $i++) {
    Start-Sleep -Seconds 5
    $log = "C:\Users\ldl\AppData\Local\Temp\openclaw\openclaw-" + (Get-Date -Format "yyyy-MM-dd") + ".log"
    if (Test-Path $log) {
        $tail = Get-Content $log -Tail 30 | Out-String
        if ($tail -match "gateway ready") { $ready = $true; Write-Host "✅ 网关已就绪 (第 $($i*5+5) 秒)"; break }
        if ($tail -match "Invalid config|startup_failed|Gateway failed to start") { Write-Host "❌ 网关启动失败:"; Write-Host $tail; break }
    }
}
if (-not $ready) { Write-Host "⚠️ 网关未在120秒内就绪，请检查日志" }

Write-Host "== 4/4 Agent 与模型确认 =="
openclaw agents list 2>&1 | Select-Object -First 30
```

## 6. 启动流水线任务示例命令（交互式 TUI）
> 用法说明（版本 2026.7.1-2）：
> - **必须用 `openclaw tui`（不带 `--local`）连接网关**；`openclaw chat` 是
>   `tui --local` 的别名（本地嵌入式），无法使用全局配置的 Agent 与模型。
> - 不存在 `--agent` 参数和 `switch-agent` 命令。
> - 指定 Agent 有两种方式（均可多轮交互）：
>   1. **`--session agent:<id>:main`（任意目录下，推荐）**：直接选中目标 Agent。
>   2. **cd 到目标 Agent 的 workspace 目录再 `openclaw tui`**：TUI 自动选中该 Agent
>      （见 docs/cli/tui.md「Inside an agent workspace it auto-selects that agent」）。
> - `leader-wxx` 为 default Agent：在**项目根目录**启动 `openclaw tui` 即自动选中
>   wxx；tsloms/eqs 的 leader 非 default，**不能**只靠项目根目录启动（会落到 wxx）。
> - 注意：若 `OPENCLAW_CONFIG_PATH` 环境变量被设置，会遮蔽全局配置导致 CLI 查不到
>   Agent；新开窗口一般无此问题，必要时 `Remove-Item Env:OPENCLAW_CONFIG_PATH`。
### WXX项目（leader-wxx 为 default，两种方式均可）
```powershell
# 方式1：项目根目录（因 leader-wxx 是 default，自动选中）
cd E:\2026-2027\2026-2027-1\MyProjects\wxx
openclaw tui

# 方式2（等价）：显式指定
openclaw tui --session agent:leader-wxx:main
```
TUI会话内发送（可多轮交互）：
```
启动WXX项目重构优化任务
```

### TSLOMS项目
```powershell
# 方式1（推荐）：显式指定，任意目录下均可
openclaw tui --session agent:leader-tsloms:main

# 方式2：cd 到 leader-tsloms 的 workspace 目录启动
cd "E:\2026-2027\2026-2027-1\MyProjects\tsloms\.openclaw\workspaces\leader-tsloms"
openclaw tui
```
TUI会话内发送：
```
启动TSLOMS项目重构优化任务
```

### EQS项目
```powershell
# 方式1（推荐）：显式指定，任意目录下均可
openclaw tui --session agent:leader-eqs:main

# 方式2：cd 到 leader-eqs 的 workspace 目录启动
cd "E:\2026-2027\2026-2027-1\MyProjects\eqs\.openclaw\workspaces\leader-eqs"
openclaw tui
```
TUI会话内发送：
```
启动EQS项目重构优化任务
```