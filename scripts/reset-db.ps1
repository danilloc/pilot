# PILOT Reset Database Script
# Run: powershell -ExecutionPolicy Bypass -File reset-db.ps1

Write-Host ""
Write-Host "PILOT Reset Database" -ForegroundColor Red
Write-Host "===================" -ForegroundColor Red
Write-Host ""

$confirm = Read-Host "WARNING: This will delete ALL database data. Type 's' to confirm"

if ($confirm -ne "s") {
    Write-Host ""
    Write-Host "Cancelled" -ForegroundColor Yellow
    Write-Host ""
    exit 0
}

Write-Host ""
Write-Host "Stopping containers..." -ForegroundColor Yellow
docker-compose down

Write-Host ""
Write-Host "Deleting volumes (data)..." -ForegroundColor Yellow
docker volume rm pilot_mysql_data

Write-Host ""
Write-Host "Restarting containers..." -ForegroundColor Yellow
docker-compose up -d

Write-Host ""
Write-Host "Waiting for MySQL (10s)..." -ForegroundColor Yellow
Start-Sleep -Seconds 10

Write-Host ""
Write-Host "Database reset complete!" -ForegroundColor Green
Write-Host "  - All data deleted" -ForegroundColor Gray
Write-Host "  - Container restarted with empty DB" -ForegroundColor Gray
Write-Host ""
Write-Host "Next step:" -ForegroundColor Cyan
Write-Host "   Run migrations when backend starts" -ForegroundColor Gray
Write-Host "   go run ./cmd/migrate/main.go up" -ForegroundColor Gray
Write-Host ""