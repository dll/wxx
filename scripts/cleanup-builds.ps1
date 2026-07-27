# 蔚小芯 Cloudflare Pages 构建产物清理脚本
# 列出部署历史，保留最近 N 个 production 部署，删除更早的版本。
# 用法:
#   pwsh scripts/cleanup-builds.ps1                    # 保留最近 5 个 production 部署
#   pwsh scripts/cleanup-builds.ps1 -Keep 10           # 保留最近 10 个
#   pwsh scripts/cleanup-builds.ps1 -Keep 5 -DryRun   # 试运行，不实际删除
param(
    [int]$Keep = 5,
    [switch]$DryRun
)

$ErrorActionPreference = "Stop"
$projectName = "wxx-agent"

Write-Output "=== 蔚小芯 Cloudflare Pages 构建产物清理 ==="
Write-Output "项目: $projectName"
Write-Output "保留 production 部署数: $Keep"
if ($DryRun) { Write-Output "模式: 试运行（不删除）" }
Write-Output ""

# ── 1. 获取 wrangler，优先本地，否则全局/npx ──
function Get-WranglerCmd {
    $localWrangler = Join-Path $PSScriptRoot "..\node_modules\.bin\wrangler"
    if (Get-Command "wrangler" -ErrorAction SilentlyContinue) { return "wrangler" }
    if (Test-Path "$localWrangler.cmd") { return "$localWrangler.cmd" }
    if (Test-Path "$localWrangler.ps1") { return "& '$localWrangler.ps1'" }
    return "npx --yes wrangler"
}

$wrangler = Get-WranglerCmd
Write-Output "Wrangler 命令: $wrangler"
Write-Output ""

# ── 2. 列出所有部署 ──
Write-Output ">>> 获取部署列表..."
$listOutput = & cmd /c "($wrangler pages deployment list --project-name $projectName 2>nul) || echo __EOF__" 2>&1 | Out-String
if ($LASTEXITCODE -ne 0 -or $listOutput -match "__EOF__") {
    # 换 PowerShell 方式试
    $listOutput = & Invoke-Expression "$wrangler pages deployment list --project-name $projectName" 2>&1 | Out-String
    if ($LASTEXITCODE -ne 0) {
        Write-Error "无法获取部署列表。请确认已登录 (npx wrangler login) 且项目 $projectName 存在。"
        exit 1
    }
}

Write-Output $listOutput

# ── 3. 解析部署列表，提取 production 部署 ──
# wrangler pages deployment list 输出表格格式：
#   ┌──────────────┬─────────┬──────────┬──────────┐
#   │ Deployment   │ Created │ Branch   │ Status   │
#   ├──────────────┼─────────┼──────────┼──────────┤
#   │ <id>         │ date    │ main     │ Success  │
#   │ <id>         │ date    │ main     │ Success  │
#   └──────────────┴─────────┴──────────┴──────────┘

$lines = $listOutput -split "`n" | ForEach-Object { $_.Trim() }
$deployments = @()
$inTable = $false
foreach ($line in $lines) {
    if ($line -match '^│\s+([a-f0-9]{8,})\s+│') {
        $id = $matches[1]
        $parts = $line -split '\s+│\s+' | ForEach-Object { $_.Trim() }
        if ($parts.Count -ge 4) {
            $deployments += [pscustomobject]@{
                Id        = $parts[0]
                Created   = $parts[1]
                Branch    = $parts[2]
                Status    = $parts[3]
            }
        }
    }
}

Write-Output "解析到 $($deployments.Count) 个部署"
Write-Output ""

# ── 4. 筛选 main 分支（production），按创建时间倒序，保留最近 N 个 ──
$prodDeployments = $deployments | Where-Object { $_.Branch -eq 'main' } | Sort-Object Created -Descending

Write-Output "Production 部署（main 分支）: $($prodDeployments.Count) 个"
if ($prodDeployments.Count -le $Keep) {
    Write-Output "无需清理（$($prodDeployments.Count) ≤ $Keep）"
    exit 0
}

$toDelete = $prodDeployments | Select-Object -Skip $Keep
Write-Output "将删除 $($toDelete.Count) 个旧部署:"
$toDelete | ForEach-Object { Write-Output "  - $($_.Id)  ($($_.Created))" }
Write-Output ""

if ($DryRun) {
    Write-Output "=== 试运行结束，未执行删除 ==="
    exit 0
}

# ── 5. 逐条删除 ──
$failed = 0
foreach ($dep in $toDelete) {
    Write-Output ">>> 删除 $($dep.Id)..."
    $delOutput = & Invoke-Expression "$wrangler pages deployment delete --project-name $projectName $($dep.Id)" 2>&1 | Out-String
    if ($LASTEXITCODE -eq 0) {
        Write-Output "  OK"
    } else {
        Write-Output "  FAILED: $delOutput"
        $failed++
    }
}

Write-Output ""
Write-Output "=== 清理完成 ==="
Write-Output "保留: $Keep | 删除: $($toDelete.Count) | 失败: $failed"
if ($failed -gt 0) { exit 1 }
