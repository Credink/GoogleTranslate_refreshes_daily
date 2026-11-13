# Build Windows app + optional NSIS installer + portable zip.
# Usage (from repo root):
#   powershell -ExecutionPolicy Bypass -File .\scripts\package-windows.ps1
# Requires: Go, Node/npm, Wails CLI. NSIS (makensis) optional for .exe installer.

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

$BinDir = Join-Path $Root "build\bin"
$DistDir = Join-Path $Root "build\dist"
New-Item -ItemType Directory -Force -Path $BinDir, $DistDir | Out-Null

Write-Host "==> wails build"
wails build
if ($LASTEXITCODE -ne 0) { throw "wails build failed" }

$Exe = Join-Path $BinDir "GoogleTranslateHostsUpdater.exe"
if (-not (Test-Path $Exe)) {
    throw "missing $Exe"
}

# Portable zip (always)
$Zip = Join-Path $DistDir "GoogleTranslateHostsUpdater-windows-amd64-portable.zip"
if (Test-Path $Zip) { Remove-Item $Zip -Force }
Compress-Archive -Path $Exe -DestinationPath $Zip -Force
Write-Host "Portable zip: $Zip"

# NSIS installer when makensis is available
$makensis = Get-Command makensis -ErrorAction SilentlyContinue
if (-not $makensis) {
    $candidates = @(
        "$env:LOCALAPPDATA\Programs\NSIS\makensis.exe",
        "C:\Program Files (x86)\NSIS\makensis.exe",
        "C:\Program Files\NSIS\makensis.exe",
        (Join-Path $Root "tools\nsis\makensis.exe")
    )
    foreach ($c in $candidates) {
        if (Test-Path $c) {
            $makensis = Get-Item $c
            break
        }
    }
}

if ($makensis) {
    Write-Host "==> wails build -nsis (makensis: $($makensis.Source -or $makensis.FullName))"
    wails build -nsis
    if ($LASTEXITCODE -ne 0) {
        Write-Warning "wails -nsis failed; portable zip is still available"
    } else {
        Get-ChildItem $BinDir -Filter "*installer*.exe" | ForEach-Object {
            Copy-Item $_.FullName $DistDir -Force
            Write-Host "Installer: $(Join-Path $DistDir $_.Name)"
        }
    }
} else {
    Write-Warning "makensis not found — skipped NSIS installer. Install NSIS then re-run, or: wails build -nsis"
}

Write-Host @"

Done.
Install / run:
  1) Portable: unzip and run GoogleTranslateHostsUpdater.exe
  2) NSIS: run *installer*.exe (optional component enables logon autostart)
Enable autostart later: open Settings → check「开机自启动」→ 保存
"@
