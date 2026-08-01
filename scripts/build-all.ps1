# 蔚小芯前端全量构建脚本
# 顺序构建 Web + APK（可靠，避免并行进程冲突）
# 用法:
#   pwsh scripts/build-all.ps1              # 版本号 patch 自动 +1 后构建
#   pwsh scripts/build-all.ps1 -NoVersionBump # 保持当前版本号构建
param(
    [switch]$NoVersionBump,
    [string]$ReleaseDate = "2026-07-20"
)

$ErrorActionPreference = "Stop"
$frontend = Resolve-Path "$PSScriptRoot/../frontend"
$buildLog = "$env:TEMP/wxx-build-$(Get-Date -Format 'yyyyMMdd-HHmmss').log"
$ts = Get-Date -Format 'yyyy-MM-dd HH:mm:ss'
$pubspec = Join-Path $frontend "pubspec.yaml"
$releaseConfig = Join-Path $frontend "lib/config/release_config.dart"
$downloadsDir = Join-Path $frontend "build/web/downloads"
$webDownloadsDir = Join-Path $frontend "web/downloads"
$baseUrl = "https://wxx-agent.pages.dev"

function Get-AppVersion {
    $text = Get-Content -LiteralPath $pubspec -Raw
    if ($text -notmatch '(?m)^version:\s*(\d+)\.(\d+)\.(\d+)\+(\d+)\s*$') {
        throw "无法从 frontend/pubspec.yaml 读取 version: x.y.z+n"
    }
    return [pscustomobject]@{
        Major = [int]$Matches[1]
        Minor = [int]$Matches[2]
        Patch = [int]$Matches[3]
        Build = [int]$Matches[4]
    }
}

function Set-AppVersion($version) {
    $versionName = "$($version.Major).$($version.Minor).$($version.Patch)"
    $buildNumber = $version.Build
    $apkFileName = "蔚小芯-v$versionName.apk"
    $encodedFileName = [uri]::EscapeDataString($apkFileName)
    $apkUrl = "$baseUrl/downloads/$encodedFileName"

    $pubspecText = Get-Content -LiteralPath $pubspec -Raw
    $pubspecText = [regex]::Replace($pubspecText, '(?m)^version:\s*\d+\.\d+\.\d+\+\d+\s*$', "version: $versionName+$buildNumber")
    Set-Content -LiteralPath $pubspec -Value $pubspecText -Encoding UTF8

    $dart = @"
/// 发布信息配置。
///
/// ``scripts/build-all.ps1`` 会在发布构建时同步更新本文件、pubspec 版本
/// 与 Web 静态发布清单，确保 Web 首页二维码指向最新 APK。
class ReleaseConfig {
  static const String version = '$versionName';
  static const int buildNumber = $buildNumber;
  static const String releaseDate = '$ReleaseDate';
  static const String apkFileName = '$apkFileName';
  static const String apkDownloadUrl = '$apkUrl';
  static const String webUrl = '$baseUrl';
}
"@
    Set-Content -LiteralPath $releaseConfig -Value $dart -Encoding UTF8

    return [pscustomobject]@{
        Version = $versionName
        BuildNumber = $buildNumber
        ApkFileName = $apkFileName
        ApkUrl = $apkUrl
    }
}

$version = Get-AppVersion
if (-not $NoVersionBump) {
    $version.Patch += 1
    $version.Build += 1
}
$release = Set-AppVersion $version

Write-Output "========================================"
Write-Output "蔚小芯前端全量构建"
Write-Output "时间: $ts"
Write-Output "日志: $buildLog"
Write-Output "版本: v$($release.Version)+$($release.BuildNumber)"
Write-Output "APK: $($release.ApkFileName)"
Write-Output "========================================"
Write-Output ""

# ── 1. 构建 Web ──
Write-Output ">>> [1/2] 构建 Web..."
$env:FLUTTER_STORAGE_BASE_URL = "https://flutter-ohos.obs.cn-south-1.myhuaweicloud.com"
Set-Location $frontend
# 注入三家地图 AK：百度/高德/腾讯，校园导航页根据 provider 选择使用。
flutter build web --release `
  --dart-define=BAIDU_MAP_AK=OUouSU6WbYExGTlnDEFqqruhTH60KAwO `
  --dart-define=GAODE_MAP_AK=a2f48050b8ec16aca88db4d25c035fe6 `
  --dart-define=TENXUN_MAP_AK=E5IBZ-ZSUC3-EQN3G-R2B5G-A7H4J-TQFIR *>> $buildLog
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
        New-Item -ItemType Directory -Force -Path $downloadsDir | Out-Null
        New-Item -ItemType Directory -Force -Path $webDownloadsDir | Out-Null
        $downloadApk = Join-Path $downloadsDir $release.ApkFileName
        Copy-Item $src $downloadApk -Force
        $releaseJson = @{
            app = "蔚小芯"
            version = $release.Version
            build_number = $release.BuildNumber
            release_date = $ReleaseDate
            apk_file = $release.ApkFileName
            apk_url = $release.ApkUrl
        } | ConvertTo-Json -Depth 4
        Set-Content -LiteralPath (Join-Path $downloadsDir "release.json") -Value $releaseJson -Encoding UTF8
        Set-Content -LiteralPath (Join-Path $webDownloadsDir "release.json") -Value $releaseJson -Encoding UTF8
        $size = (Get-Item $dst).Length / 1MB
        Write-Output "  OK  $dst ($([math]::Round($size)) MB)"
        Write-Output "  APK 下载: build/web/downloads/$($release.ApkFileName)"
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
Write-Output "  Version: v$($release.Version)+$($release.BuildNumber)"
Write-Output "========================================"
