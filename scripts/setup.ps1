# PILOT Setup Script for Windows
# Run: powershell -ExecutionPolicy Bypass -File setup.ps1

param(
    [switch]$SkipDocker = $false,
    [switch]$Verbose = $false
)

# ===== COLORS =====
$ColorGreen = "Green"
$ColorRed = "Red"
$ColorYellow = "Yellow"
$ColorCyan = "Cyan"
$ColorGray = "Gray"

# ===== FUNCTIONS =====
function Write-Title {
    param([string]$Text)
    Write-Host "" -ForegroundColor $ColorCyan
    Write-Host $Text -ForegroundColor $ColorCyan -BackgroundColor Black
    Write-Host ("=" * $Text.Length) -ForegroundColor $ColorCyan
}

function Write-Success {
    param([string]$Text)
    Write-Host "  [OK] $Text" -ForegroundColor $ColorGreen
}

function Write-ErrorCustom {
    param([string]$Text)
    Write-Host "  [ERROR] $Text" -ForegroundColor $ColorRed
}

function Write-WarningCustom {
    param([string]$Text)
    Write-Host "  [WARNING] $Text" -ForegroundColor $ColorYellow
}

function Write-InfoCustom {
    param([string]$Text)
    Write-Host "  [INFO] $Text" -ForegroundColor $ColorGray
}

function Check-Command {
    param([string]$Command)
    $null = Get-Command $Command -ErrorAction SilentlyContinue
    return $?
}

function Test-Docker {
    try {
        $null = docker ps -q
        return $true
    } catch {
        return $false
    }
}

# ===== MAIN =====
Write-Title "PILOT Setup (Windows - Hybrid)"

# 1. Check Prerequisites
Write-Title "Checking prerequisites"

$prereqs = @(
    @("Docker", "docker"),
    @("Go", "go"),
    @("Node.js", "node"),
    @("npm", "npm")
)

$missing = @()

foreach ($prereq in $prereqs) {
    $name = $prereq[0]
    $cmd = $prereq[1]
    
    if (Check-Command $cmd) {
        try {
            if ($cmd -eq "docker") {
                $version = & docker --version
            } elseif ($cmd -eq "go") {
                $version = & go version
            } else {
                $version = & $cmd -v
            }
            Write-Success "$name : $version"
        } catch {
            Write-ErrorCustom "$cmd : Error getting version"
            $missing += $cmd
        }
    } else {
        Write-ErrorCustom "$name : NOT INSTALLED"
        $missing += $cmd
    }
}

if ($missing.Count -gt 0) {
    Write-Host ""
    Write-Host "ERROR - Please install first:" -ForegroundColor $ColorRed
    foreach ($cmd in $missing) {
        Write-Host "   - $cmd" -ForegroundColor $ColorRed
    }
    Write-Host ""
    Write-Host "Download links:" -ForegroundColor $ColorCyan
    Write-Host "   Docker: https://www.docker.com/products/docker-desktop/" -ForegroundColor $ColorGray
    Write-Host "   Go: https://golang.org/dl/" -ForegroundColor $ColorGray
    Write-Host "   Node.js: https://nodejs.org/" -ForegroundColor $ColorGray
    Write-Host ""
    exit 1
}

# 2. Check Docker Running
if (-not $SkipDocker) {
    Write-Title "Checking Docker"
    
    if (Test-Docker) {
        Write-Success "Docker is running"
    } else {
        Write-WarningCustom "Docker not running. Starting..."
        Start-Sleep -Seconds 3
        if (Test-Docker) {
            Write-Success "Docker started"
        } else {
            Write-ErrorCustom "Docker not responding. Start manually."
            exit 1
        }
    }
}

# 3. Start Containers
Write-Title "Starting containers"

Write-Host "  Running: docker-compose up -d" -ForegroundColor $ColorGray
docker-compose up -d

if ($LASTEXITCODE -eq 0) {
    Write-Success "Containers started"
} else {
    Write-ErrorCustom "Error starting containers"
    exit 1
}

# 4. Wait for MySQL
Write-Title "Waiting for MySQL"

$attempts = 0
$max_attempts = 30

while ($attempts -lt $max_attempts) {
    try {
        $null = docker exec pilot_mysql mysqladmin ping -h localhost -u root -proot
        Write-Success "MySQL connected"
        break
    } catch {
        $attempts++
        Write-Host "  Attempt $attempts/$max_attempts..." -ForegroundColor $ColorYellow
        Start-Sleep -Seconds 1
    }
}

if ($attempts -eq $max_attempts) {
    Write-ErrorCustom "MySQL did not respond after 30s"
    docker-compose logs mysql
    exit 1
}

# 5. Test Redis
Write-Title "Checking Redis"

try {
    $null = docker exec pilot_redis redis-cli ping
    Write-Success "Redis connected"
} catch {
    Write-ErrorCustom "Redis not responding"
}

# 6. Download Go modules
Write-Title "Installing Go modules"

if (Test-Path "pilot-backend/go.mod") {
    Push-Location pilot-backend
    Write-Host "  Running: go mod download" -ForegroundColor $ColorGray
    go mod download
    if ($LASTEXITCODE -eq 0) {
        Write-Success "Go modules downloaded"
    } else {
        Write-WarningCustom "Error downloading modules (can continue)"
    }
    Pop-Location
} else {
    Write-InfoCustom "pilot-backend/go.mod not found (skip)"
}

# 7. Install Frontend Dependencies
Write-Title "Installing frontend dependencies"

if (Test-Path "pilot-frontend/package.json") {
    Push-Location pilot-frontend
    
    if (Test-Path "package-lock.json") {
        Write-Host "  Running: npm ci (using package-lock.json)" -ForegroundColor $ColorGray
        npm ci
    } else {
        Write-Host "  Running: npm install" -ForegroundColor $ColorGray
        npm install
    }
    
    if ($LASTEXITCODE -eq 0) {
        Write-Success "Frontend dependencies installed"
    } else {
        Write-ErrorCustom "Error installing frontend dependencies"
        Pop-Location
        exit 1
    }
    Pop-Location
} else {
    Write-InfoCustom "pilot-frontend/package.json not found (skip)"
}

# 8. Summary
Write-Title "Setup complete!"

Write-Host ""
Write-Host "Available URLs:" -ForegroundColor $ColorCyan
Write-Host "   Backend:   http://localhost:8080" -ForegroundColor "White"
Write-Host "   Frontend:  http://localhost:3000" -ForegroundColor "White"
Write-Host "   n8n Bot:   http://localhost:5678 (admin/admin123)" -ForegroundColor "White"
Write-Host "   MySQL:     localhost:3306" -ForegroundColor "White"
Write-Host "   Redis:     localhost:6379" -ForegroundColor "White"

Write-Host ""
Write-Host "Next steps - Open 2 NEW PowerShell windows:" -ForegroundColor $ColorCyan

Write-Host ""
Write-Host "  Terminal 1 (Backend):" -ForegroundColor "White"
Write-Host "    cd pilot-backend" -ForegroundColor $ColorGray
Write-Host "    go run ./cmd/server" -ForegroundColor $ColorGray

Write-Host ""
Write-Host "  Terminal 2 (Frontend):" -ForegroundColor "White"
Write-Host "    cd pilot-frontend" -ForegroundColor $ColorGray
Write-Host "    npm run dev" -ForegroundColor $ColorGray

Write-Host ""
Write-Host "View logs:" -ForegroundColor $ColorCyan
Write-Host "   docker-compose logs -f mysql" -ForegroundColor $ColorGray
Write-Host "   docker-compose logs -f redis" -ForegroundColor $ColorGray
Write-Host "   docker-compose logs -f n8n" -ForegroundColor $ColorGray

Write-Host ""
Write-Host "To stop everything:" -ForegroundColor $ColorCyan
Write-Host "   docker-compose down" -ForegroundColor $ColorGray

Write-Host ""