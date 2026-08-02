# one-shot-mempalace-up.ps1
# 一键启动 Laputa + mempalace 一体化测试。
#
# 步骤:
#   1. 连 Memurai (PING)
#   2. 清理上次残留
#   3. 启动 laputa.exe daemon (默认秒级 tick, dry-run=false)
#   4. 验证 14 section 落 Redis
#   5. 验证 daily rhythm 报告生成
#
# 用法: powershell -ExecutionPolicy Bypass -File scripts\one-shot-mempalace-up.ps1 [-DaemonSeconds 3] [-Prefix laputa:mempalace-test:]

param(
    [int]$DaemonSeconds = 3,
    [string]$Prefix = "laputa:mempalace-test:",
    [string]$RedisAddr = "localhost:6379",
    [string]$LaputaExe = ""
)

$ErrorActionPreference = "Stop"
$repo = (Get-Item $PSScriptRoot).Parent.FullName
if ([string]::IsNullOrEmpty($LaputaExe)) {
    $LaputaExe = Join-Path $repo "laputa.exe"
}
if (-not (Test-Path $LaputaExe)) {
    Write-Host "Building $LaputaExe ..."
    & go build -o $LaputaExe ./cmd/laputa
    if ($LASTEXITCODE -ne 0) { throw "build failed" }
}

$memuraiCli = "C:\Program Files\Memurai\memurai-cli.exe"
if (-not (Test-Path $memuraiCli)) { throw "memurai-cli not found at $memuraiCli" }

Write-Host "[1/5] ping memurai"
& $memuraiCli -h ($RedisAddr -replace ':.*','') -p ($RedisAddr -replace '.*:','') ping | Tee-Object -Variable pong | Out-Null
if ($LASTEXITCODE -ne 0) { throw "memurai unreachable" }

$verifyDir = Join-Path $env:TEMP "laputa-mempalace-verify"
if (Test-Path $verifyDir) { Remove-Item -Recurse -Force $verifyDir }

Write-Host "[2/5] clean previous laputa keys under $Prefix"
$existing = & $memuraiCli --scan "$Prefix*" 2>$null
if ($existing) { $existing | ForEach-Object { & $memuraiCli del $_ | Out-Null } }

Write-Host "[3/5] starting laputa daemon for $DaemonSeconds s (prefix=$Prefix)"
$stderrLog = Join-Path $env:TEMP "laputa-mempalace-stderr.log"
$stdoutLog = Join-Path $env:TEMP "laputa-mempalace-stdout.log"
Remove-Item -Force $stderrLog, $stdoutLog -ErrorAction SilentlyContinue

$proc = Start-Process -FilePath $LaputaExe `
    -ArgumentList @("-dir", $verifyDir, "-store", "redis", "-redis-addr", $RedisAddr, "-redis-prefix", $Prefix, "-cmd", "daemon", "-daemon-tick", "200ms") `
    -PassThru `
    -RedirectStandardError $stderrLog `
    -RedirectStandardOutput $stdoutLog `
    -WindowStyle Hidden

Start-Sleep -Seconds $DaemonSeconds
Stop-Process -Id $proc.Id -Force -ErrorAction SilentlyContinue
Start-Sleep -Milliseconds 800

Write-Host "[4/5] verifying laputa sections in redis"
$keys = & $memuraiCli --scan "${Prefix}*" 2>$null
$prefixEscaped = [regex]::Escape($Prefix)
$pattern = "^${prefixEscaped}\d{2}-"
$sectionCount = ($keys | Where-Object { $_ -match $pattern }).Count
Write-Host "    sections found: $sectionCount"
if ($sectionCount -lt 14) {
    Write-Host "    WARN expected >=14 sections"
} else {
    Write-Host "    OK 14+ sections persisted"
}

Write-Host "[5/5] verifying daily rhythm report"
$daily = & $memuraiCli get "${Prefix}07-daily" 2>$null
if ($daily -match 'Daily Rhythm Report') {
    Write-Host "    OK daily rhythm report generated"
} else {
    Write-Host "    FAIL daily report missing"
    Write-Host "    raw value:"
    Write-Host "    $daily"
}

Write-Host ""
Write-Host "=== daemon stderr ==="
if (Test-Path $stderrLog) { Get-Content $stderrLog }
