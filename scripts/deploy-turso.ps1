# 蔚小芯后端一键部署脚本（Vercel + Turso）
# 用法：
#   1. 先手动登录：vercel login
#   2. 运行此脚本：pwsh scripts/deploy-turso.ps1
#   3. 按提示输入 Turso 数据库 URL 和 Auth Token

param(
    [string]$TursoUrl = "",
    [string]$TursoToken = ""
)

$ErrorActionPreference = "Stop"
$ProjectRoot = Split-Path -Parent $PSScriptRoot

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  蔚小芯后端部署（Vercel + Turso）" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# ── 步骤 1：检查 Vercel 登录状态 ──
Write-Host "[1/6] 检查 Vercel 登录状态..." -ForegroundColor Yellow
$whoami = vercel whoami 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Host "  未登录 Vercel，正在启动登录流程..." -ForegroundColor Red
    vercel login
    if ($LASTEXITCODE -ne 0) {
        Write-Host "  Vercel 登录失败，请重试。" -ForegroundColor Red
        exit 1
    }
}
Write-Host "  Vercel 已登录: $whoami" -ForegroundColor Green

# ── 步骤 2：收集 Turso 信息 ──
Write-Host ""
Write-Host "[2/6] 收集 Turso 数据库信息..." -ForegroundColor Yellow

if ($TursoUrl -eq "") {
    Write-Host "  请在 Turso 控制台 (https://turso.tech/app) 创建数据库后，填写以下信息："
    $TursoUrl = Read-Host "  Turso 数据库 URL (如 libsql://wxx-agent-xxx.turso.io)"
}
if ($TursoToken -eq "") {
    $TursoToken = Read-Host "  Turso Auth Token"
}

if ($TursoUrl -eq "" -or $TursoToken -eq "") {
    Write-Host "  Turso URL 和 Token 不能为空！" -ForegroundColor Red
    exit 1
}

$DbPath = "$TursoUrl`?authToken=$TursoToken"
Write-Host "  数据库连接串: $DbPath" -ForegroundColor Green

# ── 步骤 3：执行数据库迁移 ──
Write-Host ""
Write-Host "[3/6] 执行数据库迁移（初始化 44 条种子数据）..." -ForegroundColor Yellow

$env:DB_PATH = $DbPath
Push-Location "$ProjectRoot\server"
go run cmd/migrate/main.go
$exitCode = $LASTEXITCODE
Pop-Location

if ($exitCode -ne 0) {
    Write-Host "  数据库迁移失败！" -ForegroundColor Red
    exit 1
}
Write-Host "  数据库迁移完成。" -ForegroundColor Green

# ── 步骤 4：设置 Vercel 环境变量 ──
Write-Host ""
Write-Host "[4/6] 设置 Vercel 环境变量..." -ForegroundColor Yellow

# 读取本地 .env 中的配置
$envFile = Get-Content "$ProjectRoot\.env" -ErrorAction SilentlyContinue
$envMap = @{}
foreach ($line in $envFile) {
    if ($line -match "^([^#=]+)=(.*)$") {
        $envMap[$matches[1].Trim()] = $matches[2].Trim()
    }
}

# 需要设置到 Vercel 的环境变量
$vercelEnv = @{
    "DB_PATH"              = $DbPath
    "JWT_SECRET"           = $envMap["JWT_SECRET"]
    "JWT_EXPIRE_HOURS"     = $envMap["JWT_EXPIRE_HOURS"]
    "ZHIPU_API_KEY"        = $envMap["ZHIPU_API_KEY"]
    "ZHIPU_BASE_URL"       = $envMap["ZHIPU_BASE_URL"]
    "ZHIPU_MODEL"          = $envMap["ZHIPU_MODEL"]
    "ZHIPU_4V_API_KEY"     = $envMap["ZHIPU_4V_API_KEY"]
    "ZHIPU_4V_MODEL"       = $envMap["ZHIPU_4V_MODEL"]
    "DEEPSEEK_API_KEY"     = $envMap["DEEPSEEK_API_KEY"]
    "DEEPSEEK_BASE_URL"    = $envMap["DEEPSEEK_BASE_URL"]
    "DEEPSEEK_MODEL"       = $envMap["DEEPSEEK_MODEL"]
    "XFYUN_APP_ID"         = $envMap["XFYUN_APP_ID"]
    "XFYUN_API_KEY"        = $envMap["XFYUN_API_KEY"]
    "XFYUN_API_SECRET"     = $envMap["XFYUN_API_SECRET"]
    "XFYUN_SPEECH_URL"     = $envMap["XFYUN_SPEECH_URL"]
    "APP_MODE"             = "release"
    "CORS_ALLOWED_ORIGINS" = "https://wxx-agent.online,https://www.wxx-agent.online,https://wxx-agent.pages.dev"
}

foreach ($key in $vercelEnv.Keys) {
    $value = $vercelEnv[$key]
    if ($value -and $value -ne "") {
        Write-Host "  设置 $key ..." -ForegroundColor Gray
        echo $value | vercel env add $key production 2>&1 | Out-Null
    }
}
Write-Host "  环境变量设置完成。" -ForegroundColor Green

# ── 步骤 5：部署到 Vercel ──
Write-Host ""
Write-Host "[5/6] 部署后端到 Vercel..." -ForegroundColor Yellow

Push-Location $ProjectRoot
vercel --prod --yes
$exitCode = $LASTEXITCODE
Pop-Location

if ($exitCode -ne 0) {
    Write-Host "  Vercel 部署失败！" -ForegroundColor Red
    exit 1
}
Write-Host "  Vercel 部署完成。" -ForegroundColor Green

# ── 步骤 6：验证部署 ──
Write-Host ""
Write-Host "[6/6] 验证部署..." -ForegroundColor Yellow
Start-Sleep -Seconds 5

$healthUrl = "https://wxx-server.vercel.app/api/v1/health"
try {
    $response = Invoke-RestMethod -Uri $healthUrl -Method Get -TimeoutSec 30
    Write-Host "  健康检查: $($response.status)" -ForegroundColor Green
    Write-Host "  数据库: $($response.dependencies.sqlite.status)" -ForegroundColor Green
    Write-Host "  FTS5: $($response.dependencies.fts5.status)" -ForegroundColor Green
} catch {
    Write-Host "  健康检查失败（可能需要等待冷启动）: $_" -ForegroundColor Yellow
    Write-Host "  请稍后手动访问: $healthUrl" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  部署完成！" -ForegroundColor Green
Write-Host "  后端地址: https://wxx-server.vercel.app" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""
