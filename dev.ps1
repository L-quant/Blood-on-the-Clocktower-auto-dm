<# 
  Blood on the Clocktower Auto-DM — 开发环境一键启动脚本
  
  用法: .\dev.ps1
  功能:
    1. 启动 Docker 依赖 (MySQL/Redis/RabbitMQ/Qdrant)
    2. 启动后端 (air 热重载: 改 .go 文件自动重编译重启)
    3. 启动前端 (Vue HMR: 改 .vue/.js/.scss 文件自动刷新浏览器)
  
  按 Ctrl+C 终止所有进程
#>

$ErrorActionPreference = "Stop"
$ROOT = $PSScriptRoot

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  BotC Auto-DM Dev Environment" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan

# ---------- 1. Docker ----------
Write-Host "`n[1/3] Starting Docker services..." -ForegroundColor Yellow
Push-Location "$ROOT\backend"
docker-compose up -d
Pop-Location

# Wait for MySQL
Write-Host "  Waiting for MySQL to be healthy..." -ForegroundColor Gray
$retries = 0
while ($retries -lt 30) {
    $status = docker inspect --format='{{.State.Health.Status}}' botc_mysql 2>$null
    if ($status -eq "healthy") { break }
    Start-Sleep -Seconds 2
    $retries++
}
if ($retries -ge 30) {
    Write-Host "  WARNING: MySQL may not be ready yet" -ForegroundColor Red
}
Write-Host "  Docker services ready." -ForegroundColor Green

# ---------- 2. Backend (air hot-reload) ----------
Write-Host "`n[2/3] Starting backend with hot-reload (air)..." -ForegroundColor Yellow
$backendJob = Start-Job -ScriptBlock {
    Set-Location "$using:ROOT\backend"
    # Load .env
    if (Test-Path ".env") {
        Get-Content ".env" | Where-Object { $_ -and $_ -notmatch '^\s*#' } | ForEach-Object {
            $parts = $_ -split '=', 2
            if ($parts.Length -eq 2) {
                [Environment]::SetEnvironmentVariable($parts[0].Trim(), $parts[1].Trim(), "Process")
            }
        }
    }
    & air 2>&1
}
Write-Host "  Backend PID: $($backendJob.Id) (air watching for .go changes)" -ForegroundColor Green

# Wait for backend health
Start-Sleep -Seconds 5
$backendReady = $false
for ($i = 0; $i -lt 20; $i++) {
    try {
        $resp = Invoke-WebRequest -Uri "http://localhost:8888/health" -UseBasicParsing -TimeoutSec 2 -ErrorAction SilentlyContinue
        if ($resp.StatusCode -eq 200) { $backendReady = $true; break }
    } catch {}
    Start-Sleep -Seconds 2
}
if ($backendReady) {
    Write-Host "  Backend healthy at http://localhost:8888" -ForegroundColor Green
} else {
    Write-Host "  WARNING: Backend may not be ready, check logs with: Receive-Job $($backendJob.Id)" -ForegroundColor Red
}

# ---------- 3. Frontend (Vue HMR) ----------
Write-Host "`n[3/3] Starting frontend with HMR..." -ForegroundColor Yellow
$frontendJob = Start-Job -ScriptBlock {
    Set-Location "$using:ROOT\frontend"
    & npm run serve 2>&1
}
Write-Host "  Frontend PID: $($frontendJob.Id) (Vue HMR watching for file changes)" -ForegroundColor Green

Start-Sleep -Seconds 10
Write-Host "`n========================================" -ForegroundColor Cyan
Write-Host "  All services running!" -ForegroundColor Green
Write-Host "  Frontend : http://localhost:8092" -ForegroundColor White
Write-Host "  Backend  : http://localhost:8888" -ForegroundColor White
Write-Host "  Grafana  : http://localhost:3000" -ForegroundColor White
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "  Backend hot-reload: edit any .go file -> auto rebuild" -ForegroundColor Gray
Write-Host "  Frontend HMR: edit .vue/.js/.scss -> auto refresh browser" -ForegroundColor Gray
Write-Host ""
Write-Host "  Press Ctrl+C to stop all services" -ForegroundColor Yellow
Write-Host ""

# Keep alive and show logs
try {
    while ($true) {
        # Forward any job output
        Receive-Job $backendJob -ErrorAction SilentlyContinue | Write-Host
        Receive-Job $frontendJob -ErrorAction SilentlyContinue | Write-Host
        Start-Sleep -Seconds 2
    }
} finally {
    Write-Host "`nShutting down..." -ForegroundColor Yellow
    Stop-Job $backendJob -ErrorAction SilentlyContinue
    Stop-Job $frontendJob -ErrorAction SilentlyContinue
    Remove-Job $backendJob -Force -ErrorAction SilentlyContinue
    Remove-Job $frontendJob -Force -ErrorAction SilentlyContinue
    # Kill any lingering processes
    Get-Process -Name "node" -ErrorAction SilentlyContinue | Stop-Process -Force
    Get-Process -Name "air" -ErrorAction SilentlyContinue | Stop-Process -Force
    Write-Host "All dev processes stopped." -ForegroundColor Green
}
