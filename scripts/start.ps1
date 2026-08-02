# PILOT Start Script (Daily use)
# Run: powershell -ExecutionPolicy Bypass -File start.ps1

Write-Host ""
Write-Host "PILOT Start" -ForegroundColor Cyan
Write-Host "===========" -ForegroundColor Cyan
Write-Host ""

# Check if docker-compose is already running
Write-Host "Checking Docker..." -ForegroundColor Yellow

try {
    $running = docker-compose ps --services --filter "status=running"
    if ($running) {
        Write-Host "  [OK] Docker is running" -ForegroundColor Green
    } else {
        Write-Host "  Starting Docker..." -ForegroundColor Yellow
        docker-compose up -d
        Start-Sleep -Seconds 5
        Write-Host "  [OK] Docker started" -ForegroundColor Green
    }
} catch {
    Write-Host "  Starting Docker..." -ForegroundColor Yellow
    docker-compose up -d
    Start-Sleep -Seconds 5
    Write-Host "  [OK] Docker started" -ForegroundColor Green
}

Write-Host ""
Write-Host "Infrastructure ready!" -ForegroundColor Green
Write-Host ""

Write-Host "Available URLs:" -ForegroundColor Cyan
Write-Host "   Backend:   http://localhost:8080" -ForegroundColor White
Write-Host "   Frontend:  http://localhost:3000" -ForegroundColor White
Write-Host "   n8n Bot:   http://localhost:5678" -ForegroundColor White

Write-Host ""
Write-Host "Open 2 NEW PowerShell windows:" -ForegroundColor Cyan
Write-Host ""
Write-Host "  Terminal 1 (Backend):" -ForegroundColor White
Write-Host "    cd pilot-backend && go run ./cmd/server" -ForegroundColor Gray
Write-Host ""
Write-Host "  Terminal 2 (Frontend):" -ForegroundColor White
Write-Host "    cd pilot-frontend && npm run dev" -ForegroundColor Gray
Write-Host ""
Write-Host "To stop everything:" -ForegroundColor Yellow
Write-Host "   docker-compose down" -ForegroundColor Gray
Write-Host ""