# repair-agent.ps1
# 本机反馈修复执行端（受控）。
# 两种模式：
#   claim   领取已审核(approved)任务，打印诊断报告供操作者/ AI 编码工具在隔离
#           worktree 内完成修改（默认模式，无副作用，不 push、不部署）。
#   verify  对指定任务的工作区执行自动验证并把结果上报服务端（passed→待验收，
#           failed→验证失败）。同样不 push、不部署。
#
# 安全边界：
#  1. 只处理服务端 approved/running 状态任务（服务端已审核/已领取）；
#  2. 不执行 git commit / git push / 部署；
#  3. token 由环境变量 WXX_REPAIR_AGENT_TOKEN 提供，绝不硬编码；
#  4. 验证结果仅上报，由管理员验收后再决定是否部署。
#
# 用法：
#   $env:WXX_REPAIR_AGENT_TOKEN = "..."
#   # 领取任务
#   pwsh -File scripts/repair-agent.ps1 -BaseUrl https://wxx-agent.online
#   # 修改完成后（在隔离分支）验证并上报
#   pwsh -File scripts/repair-agent.ps1 -Mode verify -TaskNo rt-xxxx -Worktree <path> \
#        -BaseUrl https://wxx-agent.online

param(
    [ValidateSet("claim", "verify")]
    [string]$Mode = "claim",
    [string]$BaseUrl = "https://wxx-agent.online",
    [string]$WorkerHost = $env:COMPUTERNAME,
    [string]$RepoRoot = "",
    [string]$TaskNo = "",
    [string]$Worktree = ""
)

$ErrorActionPreference = "Stop"

if ([string]::IsNullOrWhiteSpace($RepoRoot)) {
    $RepoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
}

$token = $env:WXX_REPAIR_AGENT_TOKEN
if ([string]::IsNullOrWhiteSpace($token)) {
    Write-Host "[repair-agent] 未设置 WXX_REPAIR_AGENT_TOKEN，内部端点不可用，退出。" -ForegroundColor Yellow
    exit 0
}

$base = $BaseUrl.TrimEnd('/')
$headers = @{ Authorization = "Bearer $token" }

function Invoke-Api {
    param($Method, $Path, $Body)
    $uri = "$base$Path"
    $params = @{ Method = $Method; Uri = $uri; Headers = $headers; ContentType = "application/json" }
    if ($null -ne $Body) { $params.Body = ($Body | ConvertTo-Json -Depth 10 -Compress) }
    try { return Invoke-RestMethod @params }
    catch {
        $status = $_.Exception.Response.StatusCode.value__
        Write-Host "[repair-agent] 请求失败 $Method $uri 状态=$status" -ForegroundColor Red
        return $null
    }
}

function Run-Verification {
    param([string]$Root)
    $result = [ordered]@{ GoVet = $true; GoTest = $true; FlutterAnalyze = $true; Log = "" }

    $serverDir = Join-Path $Root "server"
    if (Test-Path $serverDir) {
        Push-Location $serverDir
        try {
            go vet ./... 2>&1 | ForEach-Object { $result.Log += "$_`n" }
            if ($LASTEXITCODE -ne 0) { $result.GoVet = $false }
            go test ./... 2>&1 | ForEach-Object { $result.Log += "$_`n" }
            if ($LASTEXITCODE -ne 0) { $result.GoTest = $false }
        } finally { Pop-Location }
    }

    $frontendDir = Join-Path $Root "frontend"
    if (Test-Path (Join-Path $frontendDir "pubspec.yaml")) {
        Push-Location $frontendDir
        try {
            flutter analyze 2>&1 | ForEach-Object { $result.Log += "$_`n" }
            if ($LASTEXITCODE -ne 0) { $result.FlutterAnalyze = $false }
        } finally { Pop-Location }
    }

    $result.Passed = $result.GoVet -and $result.GoTest -and $result.FlutterAnalyze
    return $result
}

# 上报验证结果。
# 注意：服务端 RepairTaskVerifyRequest 的 go_vet/go_test/flutter_analyze/flutter_test
# 均为 string（如 "pass"/"fail"），此处发送字符串，避免 JSON bool→string 反序列化失败。
function Submit-Verify {
    param($TaskNo2, $Passed2, $GoVet2, $GoTest2, $FlutterAnalyze2, $DiffStat2, $Log2)
    $body = @{
        passed          = [bool]$Passed2
        go_vet          = $(if ($GoVet2) { "pass" } else { "fail" })
        go_test         = $(if ($GoTest2) { "pass" } else { "fail" })
        flutter_analyze = $(if ($FlutterAnalyze2) { "pass" } else { "fail" })
        flutter_test    = "skip"
        diff_stat       = $DiffStat2
        log             = $Log2
    }
    return Invoke-Api -Method Post -Path "/api/v1/internal/repair-tasks/$TaskNo2/verify" -Body $body
}

# 收集工作区 diff 摘要（仅信息展示与上报，不做任何提交）
function Get-DiffStat {
    param([string]$Root)
    Push-Location $Root
    try {
        $stat = git diff --stat 2>&1 | Out-String
        return $stat.Trim()
    } finally { Pop-Location }
}

if ($Mode -eq "verify") {
    if ([string]::IsNullOrWhiteSpace($TaskNo)) {
        Write-Host "[repair-agent][verify] 需要 -TaskNo" -ForegroundColor Red
        exit 1
    }
    $root = if ([string]::IsNullOrWhiteSpace($Worktree)) { $RepoRoot } else { $Worktree }
    if (-not (Test-Path $root)) {
        Write-Host "[repair-agent][verify] 工作区不存在: $root" -ForegroundColor Red
        exit 1
    }

    Write-Host "[repair-agent][verify] 开始验证任务 $TaskNo ..." -ForegroundColor Cyan
    $r = Run-Verification -Root $root
    $diff = Get-DiffStat -Root $root

    Write-Host "  go vet       : $(if($r.GoVet){'OK'}else{'FAILED'})"
    Write-Host "  go test      : $(if($r.GoTest){'OK'}else{'FAILED'})"
    Write-Host "  flutter analyze : $(if($r.FlutterAnalyze){'OK'}else{'FAILED'})"
    Write-Host "  passed       : $($r.Passed)"

    $resp = Submit-Verify -TaskNo2 $TaskNo -Passed2 $r.Passed -GoVet2 $r.GoVet `
        -GoTest2 $r.GoTest -FlutterAnalyze2 $r.FlutterAnalyze `
        -DiffStat2 $diff -Log2 ($r.Log.Substring(0, [Math]::Min($r.Log.Length, 4000)))
    if ($null -ne $resp) {
        Write-Host "[repair-agent][verify] 已上报，状态: $($resp.data.status)" -ForegroundColor Green
    }
    exit 0
}

# ── claim 模式（默认）──
$payload = Invoke-Api -Method Post -Path "/api/v1/internal/repair-tasks/next" -Body @{ worker_host = $WorkerHost }
if ($null -eq $payload -or $null -eq $payload.data) {
    Write-Host "[repair-agent] 当前没有可执行的修复任务。" -ForegroundColor Green
    exit 0
}
$task = $payload.data

Write-Host "`n[repair-agent] 领取到任务 $($task.task_no)：$($task.title)" -ForegroundColor Cyan
if ($task.diagnosis -and $task.diagnosis.summary) {
    Write-Host "  摘要: $($task.diagnosis.summary)" -ForegroundColor White
}
if ($task.diagnosis -and $task.diagnosis.code_files) {
    Write-Host "  相关文件:" -ForegroundColor White
    $task.diagnosis.code_files | ForEach-Object { Write-Host "    - $_" -ForegroundColor Gray }
}
if ($task.feedback_ids) {
    Write-Host "  关联反馈: $($task.feedback_ids -join ', ')" -ForegroundColor White
}

Write-Host "`n[repair-agent] 请在隔离分支 repair/$($task.task_no) 完成修复（人工或 AI 编码工具）。" -ForegroundColor Yellow
Write-Host "  示例： git worktree add ../wxx-repair-$($task.task_no) -b repair/$($task.task_no)" -ForegroundColor Gray
Write-Host "  修改完成并验证后上报：" -ForegroundColor Yellow
Write-Host "    pwsh -File scripts/repair-agent.ps1 -Mode verify -TaskNo $($task.task_no) -Worktree ../wxx-repair-$($task.task_no)" -ForegroundColor Gray
Write-Host "`n本脚本不执行 commit/push/部署，仅领取与上报，由管理员验收后决定部署。" -ForegroundColor DarkGray

exit 0
