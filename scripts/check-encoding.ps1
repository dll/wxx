param(
    [string]$Root = (Resolve-Path "$PSScriptRoot/..")
)

$ErrorActionPreference = "Stop"

$files = @()
$files += Get-ChildItem -Path (Join-Path $Root "frontend/lib") -Recurse -Filter "*.dart" -ErrorAction SilentlyContinue
$files += Get-ChildItem -Path (Join-Path $Root "frontend/web") -Recurse -File -ErrorAction SilentlyContinue |
    Where-Object { $_.Extension -in ".html", ".js", ".json", ".manifest" }
$files += Get-ChildItem -Path (Join-Path $Root "docs") -Recurse -Filter "*.md" -ErrorAction SilentlyContinue
$files += Get-ChildItem -Path (Join-Path $Root "scripts") -Recurse -Filter "*.ps1" -ErrorAction SilentlyContinue
if (Test-Path (Join-Path $Root "AGENTS.md")) {
    $files += Get-Item -LiteralPath (Join-Path $Root "AGENTS.md")
}

$bad = @()
foreach ($f in $files) {
    if ($f.PSIsContainer) {
        continue
    }
    $bytes = [System.IO.File]::ReadAllBytes($f.FullName)
    if ($bytes.Length -ge 2 -and $bytes[0] -eq 0xFF -and $bytes[1] -eq 0xFE) {
        $bad += "UTF16LE $($f.FullName)"
        continue
    }
    if ($bytes.Length -ge 3 -and $bytes[0] -eq 0xEF -and $bytes[1] -eq 0xBB -and $bytes[2] -eq 0xBF) {
        $bad += "UTF8BOM $($f.FullName)"
        continue
    }
    try {
        $null = [System.Text.UTF8Encoding]::new($false, $true).GetString($bytes)
    } catch {
        $bad += "BADUTF8 $($f.FullName)"
    }
}

if ($bad.Count -eq 0) {
    Write-Output "ENCODING_OK"
    exit 0
}

$bad | Select-Object -First 100
Write-Error "源码编码校验失败：以上文件不是 UTF-8（且无 BOM/UTF-16）。请用编辑器以 UTF-8 重新保存后再构建。"
exit 1
