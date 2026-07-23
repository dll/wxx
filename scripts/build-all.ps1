# 蔚小芯前端全量构建脚本
# 顺序构建 Web + APK（可靠，避免并行进程冲突）
# 用法: pwsh scripts/build-all.ps1

$ErrorActionPreference = "Stop"
$frontend = Resolve-Path "$PSScriptRoot/../frontend"
$buildLog = "$env:TEMP/wxx-build-$(Get-Date -Format 'yyyyMMdd-HHmmss').log"
$ts = Get-Date -Format 'yyyy-MM-dd HH:mm:ss'

Write-Output "========================================"
Write-Output "蔚小芯前端全量构建"
Write-Output "时间: $ts"
Write-Output "日志: $buildLog"
Write-Output "========================================"
Write-Output ""

# ── 1. 构建 Web ──
Write-Output ">>> [1/2] 构建 Web..."
$env:FLUTTER_STORAGE_BASE_URL = "https://flutter-ohos.obs.cn-south-1.myhuaweicloud.com"
Set-Location $frontend
flutter build web --release --web-renderer html *>> $buildLog
$webOk = $LASTEXITCODE -eq 0

if ($webOk) {
    $size = (Get-Item "build/web/index.html").Length / 1KB
    Write-Output "  OK  build/web/ (index.html $([math]::Round($size)) KB)"
} else {
    Write-Output "  FAILED (exit $LASTEXITCODE)"
    Get-Content $buildLog -Tail 5
}

Write-Output ""

# ── 2. 构建 APK ──
Write-Output ">>> [2/2] 构建 APK..."
$androidDir = "$frontend/android"
Set-Location $androidDir

$args = @(
    "assembleRelease"
    "--stacktrace"
    "-Pverbose=true"
    "-Ptarget-platform=android-arm,android-arm64,android-x64"
    "-Ptarget=lib\main.dart"
    "-Pbase-application-name=android.app.Application"
    "-Pdart-defines=RkxVVFRFUl9WRVJTSU9OPTMuMzUuMQ==,RkxVVFRFUl9DSEFOTkVMPXN0YWJsZQ==,RkxVVFRFUl9HSVRfVVJMPWh0dHBzOi8vZ2l0aHViLmNvbS9mbHV0dGVyL2ZsdXR0ZXIuZ2l0,RkxVVFRFUl9GUkFNRVdPUktfUkVWSVNJT049MjBmODI3NDkzOQ==,RkxVVFRFUl9FTkdJTkVfUkVWSVNJT049MWU5YTgxMWJmOA==,RkxVVFRFUl9EQVJUX1ZFUlNJT049My45LjA="
    "-Pdart-obfuscation=false"
    "-Ptrack-widget-creation=true"
    "-Ptree-shake-icons=true"
)
& ".\gradlew.bat" $args *>> $buildLog
$apkOk = $LASTEXITCODE -eq 0

if ($apkOk) {
    # Gradle 输出重定向到 frontend/build/，从那里查找 APK
    $candidates = @(
        "$frontend/build/app/outputs/flutter-apk/app-release.apk"
        "$androidDir/build/app/outputs/flutter-apk/app-release.apk"
    )
    $src = $null
    foreach ($p in $candidates) { if (Test-Path $p) { $src = $p; break } }
    if ($src) {
        $dst = "$frontend/build/app/outputs/flutter-apk/weixiaoxin-release.apk"
        Copy-Item $src $dst -Force
        $size = (Get-Item $dst).Length / 1MB
        Write-Output "  OK  $dst ($([math]::Round($size)) MB)"
    } else {
        Write-Output "  OK  (Gradle 退出码 0，但未找到 APK 产物)"
    }
} else {
    Write-Output "  FAILED (exit $LASTEXITCODE)"
    Get-Content $buildLog | Select-String -Pattern "(ERROR|FAILED|BUILD FAILED|error:)" | Select-Object -Last 5
}

Write-Output ""
Write-Output "========================================"
Write-Output "构建结果"
if ($webOk) { Write-Output "  Web:  OK" } else { Write-Output "  Web: FAIL" }
if ($apkOk) { Write-Output "  APK:  OK" } else { Write-Output "  APK: FAIL" }
Write-Output "========================================"
