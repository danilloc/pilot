# 🚀 PILOT — Setup Windows (Híbrido)

**Sua Config**: Windows 10/11 + Docker Desktop + Go/Node Local  
**Tempo Total**: ~45 min  

---

## 📋 Pré-requisitos

### 1️⃣ **Git** (Essencial)
```
https://git-scm.com/download/win

✅ Selecionar:
  - Use Git Bash here (context menu)
  - Use Windows' default console
  - Checkout Windows-style line endings
```

### 2️⃣ **Docker Desktop** (Principal)
```
https://www.docker.com/products/docker-desktop/

⚠️ Após instalar:
  - Reiniciar Windows
  - Ativar WSL2 (vai pedir)
  - Deixar rodando (tray)
```

### 3️⃣ **Go 1.21+**
```
https://golang.org/dl/

Passo a passo:
1. Download: go1.21.x.windows-amd64.msi
2. Executar installer (next, next, finish)
3. Abrir PowerShell (admin) e verificar:
   
   go version
   # Output: go version go1.21.x windows/amd64
```

### 4️⃣ **Node.js 20 LTS**
```
https://nodejs.org/

Passo a passo:
1. Download: Node-v20.x.x-x64.msi
2. Executar installer (next, next, finish)
3. Fechar e reabrir PowerShell
4. Verificar:
   
   node -v    # v20.x.x
   npm -v     # 10.x.x
```

---

## ✅ Verificação (após instalar tudo)

Abrir **PowerShell** (admin) e rodar:

```powershell
# Verificar tudo
Write-Host "🔍 Verificando setup..." -ForegroundColor Green

$checks = @(
    ("Docker", { docker --version }),
    ("Go", { go version }),
    ("Node", { node -v }),
    ("npm", { npm -v })
)

foreach ($check in $checks) {
    try {
        $output = & $check[1]
        Write-Host "✅ $($check[0]): $output" -ForegroundColor Green
    } catch {
        Write-Host "❌ $($check[0]): NOT INSTALLED" -ForegroundColor Red
    }
}
```

---

## 📁 Setup Projeto

### 1. Criar Diretório

```powershell
# Abrir PowerShell
mkdir C:\dev\pilot
cd C:\dev\pilot
```

### 2. Clonar Repositório (ou criar do zero)

```powershell
# Se tiver repo
git clone https://seu-repo/pilot.git .

# Ou criar estrutura:
mkdir pilot-backend
mkdir pilot-frontend
mkdir scripts
```

### 3. Estrutura Final

```
C:\dev\pilot\
├── docker-compose.yml
├── .env.local
├── pilot-backend\
│   ├── cmd\
│   ├── internal\
│   ├── go.mod
│   └── go.sum
├── pilot-frontend\
│   ├── components\
│   ├── pages\
│   ├── package.json
│   └── package-lock.json
└── scripts\
    ├── setup.ps1
    ├── reset-db.ps1
    └── start.ps1
```

---

## 🐳 docker-compose.yml

Criar arquivo `C:\dev\pilot\docker-compose.yml`:

```yaml
version: '3.9'

services:
  # ===== DATABASE =====
  mysql:
    image: mysql:8.0
    container_name: pilot_mysql
    platform: linux/amd64  # ⚠️ Important pra Windows
    environment:
      MYSQL_ROOT_PASSWORD: root
      MYSQL_DATABASE: pilot_db
      MYSQL_USER: pilot_app
      MYSQL_PASSWORD: pilot_secret_123
    ports:
      - "3306:3306"
    volumes:
      - mysql_data:/var/lib/mysql
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - pilot_network

  # ===== CACHE =====
  redis:
    image: redis:7-alpine
    container_name: pilot_redis
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - pilot_network

  # ===== BOT ORCHESTRATION =====
  n8n:
    image: n8nio/n8n:latest
    container_name: pilot_n8n
    ports:
      - "5678:5678"
    environment:
      - N8N_BASIC_AUTH_ACTIVE=true
      - N8N_BASIC_AUTH_USER=admin
      - N8N_BASIC_AUTH_PASSWORD=admin123
      - DB_TYPE=sqlite
      - NODE_ENV=development
    volumes:
      - n8n_data:/root/.n8n
    networks:
      - pilot_network
    depends_on:
      - mysql
      - redis

volumes:
  mysql_data:
  redis_data:
  n8n_data:

networks:
  pilot_network:
    driver: bridge
```

---

## 🔧 .env.local

Criar arquivo `C:\dev\pilot\.env.local`:

```env
# ===== SERVER =====
PORT=8080
ENVIRONMENT=development
GIN_MODE=debug

# ===== DATABASE =====
DB_HOST=localhost
DB_PORT=3306
DB_USER=pilot_app
DB_PASSWORD=pilot_secret_123
DB_NAME=pilot_db
DB_SSLMODE=disable

# ===== ENCRYPTION =====
ENCRYPTION_KEY=0123456789abcdef0123456789abcdef01234567890123=
ENCRYPTION_SALT=0123456789abcdef0123456789abcdef

# ===== JWT =====
JWT_SECRET=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef01234567
JWT_EXPIRY=24h

# ===== OAUTH2 (Uber) =====
UBER_CLIENT_ID=your_client_id_here
UBER_CLIENT_SECRET=your_client_secret_here
UBER_REDIRECT_URI=http://localhost:3000/auth/callback

# ===== REDIS =====
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=

# ===== LOGGING =====
LOG_LEVEL=debug
LOG_FORMAT=json

# ===== SECURITY =====
CORS_ORIGINS=http://localhost:3000,http://localhost:5678
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS=100
RATE_LIMIT_WINDOW=60
```

---

## 🚀 Scripts PowerShell

### `scripts/setup.ps1`

```powershell
# C:\dev\pilot\scripts\setup.ps1
# Executar: powershell -ExecutionPolicy Bypass -File scripts\setup.ps1

param(
    [switch]$SkipDocker = $false
)

Write-Host "`n🚀 PILOT Setup (Windows)" -ForegroundColor Cyan
Write-Host "========================`n" -ForegroundColor Cyan

# Funções auxiliares
function Check-Command {
    param([string]$Command)
    $null = Get-Command $Command -ErrorAction SilentlyContinue
    return $?
}

function Test-Service {
    param([string]$Service)
    try {
        docker ps > $null 2>&1
        return $true
    } catch {
        return $false
    }
}

# 1. Verificar pré-requisitos
Write-Host "📋 Verificando pré-requisitos..." -ForegroundColor Yellow

$prereqs = @("docker", "go", "node", "npm")
$missing = @()

foreach ($cmd in $prereqs) {
    if (Check-Command $cmd) {
        if ($cmd -eq "docker") {
            $version = docker --version
        } elseif ($cmd -eq "go") {
            $version = go version
        } else {
            $version = & $cmd -v
        }
        Write-Host "  ✅ $cmd : $version" -ForegroundColor Green
    } else {
        Write-Host "  ❌ $cmd : NÃO INSTALADO" -ForegroundColor Red
        $missing += $cmd
    }
}

if ($missing.Count -gt 0) {
    Write-Host "`n❌ Instale primeiro:" -ForegroundColor Red
    $missing | ForEach-Object { Write-Host "   - $_" }
    exit 1
}

# 2. Iniciar Docker
if (-not $SkipDocker) {
    Write-Host "`n🐳 Iniciando Docker..." -ForegroundColor Yellow
    
    if (Test-Service "docker") {
        Write-Host "   ✅ Docker já rodando" -ForegroundColor Green
    } else {
        Write-Host "   ⏳ Aguardando Docker iniciar..." -ForegroundColor Yellow
        Start-Sleep -Seconds 2
    }
}

# 3. Containers
Write-Host "`n📦 Iniciando containers..." -ForegroundColor Yellow
docker-compose up -d

if ($LASTEXITCODE -eq 0) {
    Write-Host "   ✅ Containers iniciados" -ForegroundColor Green
} else {
    Write-Host "   ❌ Erro ao iniciar containers" -ForegroundColor Red
    exit 1
}

# 4. Aguardar MySQL
Write-Host "`n⏳ Aguardando MySQL conectar..." -ForegroundColor Yellow
$attempts = 0
$max_attempts = 30

while ($attempts -lt $max_attempts) {
    try {
        docker exec pilot_mysql mysqladmin ping -h localhost -u root -proot > $null 2>&1
        Write-Host "   ✅ MySQL conectado" -ForegroundColor Green
        break
    } catch {
        $attempts++
        Write-Host "   ⏳ Tentativa $attempts/$max_attempts..." -ForegroundColor Yellow
        Start-Sleep -Seconds 1
    }
}

if ($attempts -eq $max_attempts) {
    Write-Host "   ❌ MySQL não respondeu após 30s" -ForegroundColor Red
    exit 1
}

# 5. Frontend deps
Write-Host "`n📦 Instalando dependências frontend..." -ForegroundColor Yellow
cd pilot-frontend

if (Test-Path "package-lock.json") {
    Write-Host "   ℹ️  package-lock.json encontrado, usando npm ci" -ForegroundColor Cyan
    npm ci
} else {
    Write-Host "   ℹ️  Executando npm install" -ForegroundColor Cyan
    npm install
}

if ($LASTEXITCODE -eq 0) {
    Write-Host "   ✅ Dependências instaladas" -ForegroundColor Green
} else {
    Write-Host "   ❌ Erro ao instalar dependências" -ForegroundColor Red
    exit 1
}

cd ..

# 6. Backend setup
Write-Host "`n📦 Setup backend..." -ForegroundColor Yellow
cd pilot-backend

go mod download
if ($LASTEXITCODE -eq 0) {
    Write-Host "   ✅ Go modules downloaded" -ForegroundColor Green
}

cd ..

# 7. Resumo
Write-Host "`n✅ Setup completo!`n" -ForegroundColor Green
Write-Host "Próximas ações:" -ForegroundColor Cyan
Write-Host "1. Abra 3 PowerShells:" -ForegroundColor White
Write-Host "   - PowerShell 1 (já deixe docker-compose running)" -ForegroundColor Gray
Write-Host "   - PowerShell 2: cd pilot-backend && go run ./cmd/server" -ForegroundColor Gray
Write-Host "   - PowerShell 3: cd pilot-frontend && npm run dev" -ForegroundColor Gray
Write-Host "`n📍 URLs:" -ForegroundColor Cyan
Write-Host "   Backend: http://localhost:8080" -ForegroundColor White
Write-Host "   Frontend: http://localhost:3000" -ForegroundColor White
Write-Host "   n8n: http://localhost:5678 (admin/admin123)" -ForegroundColor White
Write-Host "   MySQL: localhost:3306" -ForegroundColor White
Write-Host "   Redis: localhost:6379" -ForegroundColor White
```

### `scripts/start.ps1`

```powershell
# C:\dev\pilot\scripts\start.ps1
# Executar: powershell -ExecutionPolicy Bypass -File scripts\start.ps1

Write-Host "`n🚀 PILOT Start" -ForegroundColor Cyan
Write-Host "===============`n" -ForegroundColor Cyan

# 1. Docker
Write-Host "🐳 Iniciando Docker..." -ForegroundColor Yellow
docker-compose up -d

Write-Host "`n✅ Infraestrutura rodando!" -ForegroundColor Green

# 2. Abrir em novo terminal
Write-Host "`n📖 Abra 2 PowerShells novos:` -ForegroundColor Cyan
Write-Host "`n  Terminal 1 (Backend):`n    cd pilot-backend && go run ./cmd/server`n" -ForegroundColor Gray
Write-Host "  Terminal 2 (Frontend):`n    cd pilot-frontend && npm run dev`n" -ForegroundColor Gray

Write-Host "📍 URLs:`n" -ForegroundColor Cyan
Write-Host "   Backend:  http://localhost:8080" -ForegroundColor White
Write-Host "   Frontend: http://localhost:3000`n" -ForegroundColor White
```

### `scripts/reset-db.ps1`

```powershell
# C:\dev\pilot\scripts\reset-db.ps1
# Executar: powershell -ExecutionPolicy Bypass -File scripts\reset-db.ps1

Write-Host "`n🗑️  PILOT Reset Database" -ForegroundColor Red
Write-Host "========================`n" -ForegroundColor Red

$confirm = Read-Host "Tem certeza? Vai deletar todos os dados (s/n)"

if ($confirm -ne "s") {
    Write-Host "❌ Cancelado" -ForegroundColor Yellow
    exit 0
}

Write-Host "`n⏳ Deletando volumes..." -ForegroundColor Yellow
docker-compose down -v

Write-Host "`n⏳ Reiniciando containers..." -ForegroundColor Yellow
docker-compose up -d

Write-Host "`n⏳ Aguardando MySQL..." -ForegroundColor Yellow
Start-Sleep -Seconds 10

Write-Host "`n✅ Database resetado!" -ForegroundColor Green
Write-Host "   Rodas as migrations manualmente quando o backend iniciar" -ForegroundColor Gray
```

---

## 🎬 Primeiro Setup (Completo)

### Passo 1: Abrir PowerShell (Admin)

```powershell
# Permitir scripts
Set-ExecutionPolicy -ExecutionPolicy Bypass -Scope CurrentUser -Force

# Ir pro diretório
cd C:\dev\pilot

# Rodar setup
powershell -ExecutionPolicy Bypass -File scripts\setup.ps1
```

### Passo 2: Abrir 2 PowerShells Novos

**PowerShell 1 (Backend - Porta 8080):**
```powershell
cd C:\dev\pilot\pilot-backend
go run ./cmd/server
```

**PowerShell 2 (Frontend - Porta 3000):**
```powershell
cd C:\dev\pilot\pilot-frontend
npm run dev
```

### Passo 3: Testar

```powershell
# Em outro PowerShell, testar:
curl http://localhost:8080/api/health
curl http://localhost:3000
```

---

## 🔄 Workflow Diário

### Iniciar Tudo

```powershell
# PowerShell 1: Infra (pode deixar rodando)
cd C:\dev\pilot
docker-compose up -d

# PowerShell 2: Backend
cd C:\dev\pilot\pilot-backend
go run ./cmd/server

# PowerShell 3: Frontend
cd C:\dev\pilot\pilot-frontend
npm run dev

# Agora acessa:
# Frontend: http://localhost:3000
# Backend: http://localhost:8080
```

### Parar Tudo

```powershell
# Em qualquer PowerShell:
cd C:\dev\pilot
docker-compose down

# Dados persistem em volumes! Reset só com:
# docker-compose down -v
```

### Reset Database (Zerar dados)

```powershell
powershell -ExecutionPolicy Bypass -File scripts\reset-db.ps1
```

---

## 🛠️ Troubleshooting Windows

### ❌ Docker não inicia

```powershell
# 1. WSL2 kernel update
wsl --update

# 2. Reiniciar Docker Desktop
# Tray → Docker icon → Restart

# 3. Checar status
docker info
```

### ❌ Porta 3306 em uso

```powershell
# Achar processo
netstat -ano | findstr :3306

# Matar processo (PID em decimal)
taskkill /PID <PID> /F

# Ou trocar no docker-compose:
# ports:
#   - "3307:3306"
```

### ❌ Go módulos lento

```powershell
# Usar proxy China (mais rápido)
go env -w GOPROXY=https://goproxy.cn,direct

# Ou resetar
go env -w GOPROXY=https://proxy.golang.org,direct
```

### ❌ Node dependencies erro

```powershell
# Limpar cache npm
npm cache clean --force

# Reinstalar
rm -r node_modules package-lock.json
npm install
```

---

## 📊 Ports Utilizadas

| Serviço | Porta | URL |
|---------|-------|-----|
| Backend (Go) | 8080 | http://localhost:8080 |
| Frontend (Nuxt) | 3000 | http://localhost:3000 |
| n8n Bot | 5678 | http://localhost:5678 |
| MySQL | 3306 | localhost:3306 |
| Redis | 6379 | localhost:6379 |

Se alguma porta tá em uso, troque em `docker-compose.yml`.

---

## ✨ Dicas Windows-Specific

### 📌 Terminal Melhor

```powershell
# Usar Windows Terminal (Microsoft Store)
# Mais rápido e bonitão que PowerShell default
```

### 📌 Path Longo

Windows tem limite de 260 caracteres. Se der erro:

```powershell
# Enable long paths
New-ItemProperty -Path "HKLM:\SYSTEM\CurrentControlSet\Control\FileSystem" `
  -Name "LongPathsEnabled" -Value 1 -PropertyType DWORD -Force
```

### 📌 Performance

```powershell
# Se docker tá lento:
# Docker Desktop → Settings → Resources → Memory: 8GB+ RAM
```

---

## ✅ Checklist Final

- [ ] Docker Desktop instalado e rodando
- [ ] Go 1.21+ instalado
- [ ] Node 20+ instalado
- [ ] npm instalado
- [ ] scripts\setup.ps1 rodou com sucesso
- [ ] docker-compose up funcionando
- [ ] Backend rodando na porta 8080
- [ ] Frontend rodando na porta 3000
- [ ] Pode acessar http://localhost:3000
- [ ] MySQL conectado (docker ps mostra saudável)

---

**Tá pronto? É só rodar `setup.ps1` e sair! 🚀**

