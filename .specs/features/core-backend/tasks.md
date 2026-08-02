# Pilot — Core Backend Tasks

**Feature**: `core-backend`  
**Branch**: `features/core-backend`  
**Assignee**: danillobrito-sr  
**Milestone**: Pilot MVP  

---

## 📋 Task List Overview

| Task | Issue | Status | Deps | Approx |
|------|-------|--------|------|--------|
| T1   | #51   | 🟡 Ready | None | 4h |
| T2   | #52   | ⏳ Blocked | T1 | 3h |
| T3   | #53   | ⏳ Blocked | T1 | 3h |
| T4   | #54   | ⏳ Blocked | T2, T3 | 2h |
| T5   | #55   | ⏳ Blocked | T1 | 3h |
| T6   | #56   | ⏳ Blocked | T5 | 2h |
| T7   | #57   | ⏳ Blocked | T1 | 4h |
| T8   | #58   | ⏳ Blocked | T7 | 3h |
| T9   | #59   | ⏳ Blocked | T7 | 3h |
| T10  | #60   | ⏳ Blocked | T7 | 3h |

---

## T1: Setup Projeto Go + Estrutura Base

**Issue**: #51  
**Assignee**: danillobrito-sr  
**Status**: 🟡 Ready (independente)  
**Tempo Estimado**: 4 horas  
**Dependências**: Nenhuma  

### Descrição

Inicializar projeto Go com:
- go.mod e dependências core
- Estrutura de diretórios (cmd, internal, pkg)
- Config com .env
- Logger estruturado (zap)
- Error handling customizado
- Encryption utilities (AES-256-GCM)

### Gate (Definição de Pronto)

- [ ] `go.mod` criado com dependências principais
- [ ] Estrutura de pastas criada (/cmd, /internal, /pkg)
- [ ] `config.go` carrega todas as variáveis de .env
- [ ] `logger.go` com structured logging (JSON)
- [ ] `errors.go` com tipos de erro customizados
- [ ] `crypto.go` com AES-256-GCM encrypt/decrypt
- [ ] `main.go` inicia server sem erros
- [ ] `.env.example` criado e documentado
- [ ] `go mod tidy` funciona
- [ ] Código compilável com `go build ./cmd/server`

### Checklist de Implementação

```go
// cmd/server/main.go
package main

import (
    "log"
    "os"
    
    "github.com/joho/godotenv"
    "seu-usuario/pilot-backend/internal/config"
    "seu-usuario/pilot-backend/pkg/logger"
)

func main() {
    // 1. Load .env
    if err := godotenv.Load(); err != nil {
        log.Fatal("Error loading .env file")
    }
    
    // 2. Load config
    cfg := config.Load()
    
    // 3. Setup logger
    log := logger.New(cfg.LogLevel)
    defer log.Sync()
    
    log.Info("server.starting", "port", cfg.Port, "env", cfg.Environment)
    
    // 4. Start server (vazio por enquanto)
    log.Info("server.ready")
}
```

```go
// internal/config/config.go
package config

import (
    "os"
    "strconv"
    "time"
)

type Config struct {
    // Server
    Port        int
    Environment string // development, staging, production
    
    // Database
    DBHost     string
    DBPort     int
    DBUser     string
    DBPassword string
    DBName     string
    DBSSLMode  string
    
    // Encryption
    EncryptionKey  string // base64 32 bytes
    EncryptionSalt string // base64 16 bytes
    
    // JWT
    JWTSecret string        // base64 64 bytes
    JWTExpiry time.Duration
    
    // OAuth
    UberClientID     string
    UberClientSecret string
    UberRedirectURI  string
    
    // Redis
    RedisHost string
    RedisPort int
    
    // Logging
    LogLevel string // debug, info, warn, error
    
    // Security
    CORSOrigins     []string
    RateLimitEnabled bool
}

func Load() *Config {
    return &Config{
        Port:        getEnvInt("PORT", 8080),
        Environment: getEnv("ENVIRONMENT", "development"),
        
        DBHost:     getEnv("DB_HOST", "localhost"),
        DBPort:     getEnvInt("DB_PORT", 3306),
        DBUser:     getEnv("DB_USER", ""),
        DBPassword: getEnv("DB_PASSWORD", ""),
        DBName:     getEnv("DB_NAME", "pilot_db"),
        DBSSLMode:  getEnv("DB_SSLMODE", "disable"),
        
        EncryptionKey:  getEnv("ENCRYPTION_KEY", ""),
        EncryptionSalt: getEnv("ENCRYPTION_SALT", ""),
        
        JWTSecret: getEnv("JWT_SECRET", ""),
        JWTExpiry: time.Hour * 24,
        
        UberClientID:     getEnv("UBER_CLIENT_ID", ""),
        UberClientSecret: getEnv("UBER_CLIENT_SECRET", ""),
        UberRedirectURI:  getEnv("UBER_REDIRECT_URI", ""),
        
        RedisHost: getEnv("REDIS_HOST", "localhost"),
        RedisPort: getEnvInt("REDIS_PORT", 6379),
        
        LogLevel: getEnv("LOG_LEVEL", "info"),
    }
}

func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
    if value := os.Getenv(key); value != "" {
        if intVal, err := strconv.Atoi(value); err == nil {
            return intVal
        }
    }
    return defaultValue
}
```

```go
// pkg/logger/logger.go
package logger

import (
    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
)

type Logger struct {
    *zap.SugaredLogger
}

func New(level string) *Logger {
    config := zap.NewProductionConfig()
    
    switch level {
    case "debug":
        config.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
    case "info":
        config.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
    case "warn":
        config.Level = zap.NewAtomicLevelAt(zapcore.WarnLevel)
    case "error":
        config.Level = zap.NewAtomicLevelAt(zapcore.ErrorLevel)
    default:
        config.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
    }
    
    config.Encoding = "json"
    
    logger, _ := config.Build()
    return &Logger{logger.Sugar()}
}
```

```go
// pkg/crypto/crypto.go
package crypto

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "encoding/base64"
    "io"
)

type Encryptor struct {
    key []byte
}

func NewEncryptor(keyBase64 string) (*Encryptor, error) {
    key, err := base64.StdEncoding.DecodeString(keyBase64)
    if err != nil {
        return nil, err
    }
    if len(key) != 32 {
        return nil, errors.New("key must be 32 bytes (AES-256)")
    }
    return &Encryptor{key: key}, nil
}

func (e *Encryptor) Encrypt(plaintext string) (string, error) {
    block, err := aes.NewCipher(e.key)
    if err != nil {
        return "", err
    }
    
    aead, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }
    
    nonce := make([]byte, aead.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return "", err
    }
    
    ciphertext := aead.Seal(nonce, nonce, []byte(plaintext), nil)
    return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (e *Encryptor) Decrypt(ciphertext string) (string, error) {
    data, err := base64.StdEncoding.DecodeString(ciphertext)
    if err != nil {
        return "", err
    }
    
    block, err := aes.NewCipher(e.key)
    if err != nil {
        return "", err
    }
    
    aead, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }
    
    nonceSize := aead.NonceSize()
    if len(data) < nonceSize {
        return "", errors.New("ciphertext too short")
    }
    
    nonce, ciphertext := data[:nonceSize], data[nonceSize:]
    plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
    
    return string(plaintext), err
}
```

```go
// internal/models/errors.go
package models

import "fmt"

type ErrorCode string

const (
    // Auth errors
    ErrCodeInvalidToken     ErrorCode = "AUTH_001"
    ErrCodeTokenExpired     ErrorCode = "AUTH_002"
    ErrCodeTokenBlacklisted ErrorCode = "AUTH_003"
    ErrCodeOAuthFailed      ErrorCode = "AUTH_004"
    ErrCodeDriverBanned     ErrorCode = "AUTH_005"
    
    // Validation errors
    ErrCodeInvalidEmail    ErrorCode = "VAL_001"
    ErrCodeInvalidPhone    ErrorCode = "VAL_002"
    ErrCodeInvalidRating   ErrorCode = "VAL_003"
    ErrCodeInvalidDistance ErrorCode = "VAL_004"
    ErrCodeInvalidFare     ErrorCode = "VAL_005"
    
    // Database errors
    ErrCodeDatabaseError ErrorCode = "DB_001"
    ErrCodeNotFound      ErrorCode = "DB_002"
    
    // Server errors
    ErrCodeInternalError ErrorCode = "ERR_500"
    ErrCodeEncryption    ErrorCode = "ERR_ENC"
)

type APIError struct {
    Code      ErrorCode `json:"code"`
    Message   string    `json:"message"`
    RequestID string    `json:"request_id,omitempty"`
    Timestamp string    `json:"timestamp,omitempty"`
}

func NewAPIError(code ErrorCode, message string) *APIError {
    return &APIError{
        Code:    code,
        Message: message,
    }
}

func (e *APIError) Error() string {
    return fmt.Sprintf("%s: %s", e.Code, e.Message)
}
```

### Commit

```
feat(core-backend): setup go project with encryption and config

- Initialize go.mod with core dependencies
- Create project structure (cmd, internal, pkg)
- Add config loader with .env support
- Implement structured JSON logging (zap)
- Add AES-256-GCM encryption utilities
- Define custom error types and codes
- Add main.go entry point
- Create .env.example with all required vars

Closes #51
```

---

## T2: Database Connection + ORM Setup

**Issue**: #52  
**Assignee**: danillobrito-sr  
**Status**: ⏳ Blocked  
**Tempo Estimado**: 3 horas  
**Dependências**: T1  

### Descrição

Configurar conexão com MySQL usando GORM:
- Connection pooling
- SSL/TLS obrigatório
- Query logging (dev only)
- Health check

### Gate (Definição de Pronto)

- [ ] GORM conecta com sucesso ao MySQL
- [ ] Connection pooling configurado (10-50 connections)
- [ ] SSL/TLS habilitado para produção
- [ ] Query logging apenas em development
- [ ] Health check endpoint funciona
- [ ] Migrations podem rodar (`migrate up`)
- [ ] Testes de connection passam
- [ ] Erro de conexão é tratado gracefully

### Implementação

```go
// pkg/db/mysql.go
package db

import (
    "fmt"
    "time"
    
    "gorm.io/driver/mysql"
    "gorm.io/gorm"
    "gorm.io/gorm/logger"
    "seu-usuario/pilot-backend/internal/config"
    appLogger "seu-usuario/pilot-backend/pkg/logger"
)

func Connect(cfg *config.Config, log *appLogger.Logger) *gorm.DB {
    dsn := fmt.Sprintf(
        "%s:%s@tcp(%s:%d)/%s?parseTime=true&loc=Local&tls=%s",
        cfg.DBUser,
        cfg.DBPassword,
        cfg.DBHost,
        cfg.DBPort,
        cfg.DBName,
        cfg.DBSSLMode,
    )
    
    var logLevel logger.LogLevel
    if cfg.Environment == "production" {
        logLevel = logger.Silent
    } else {
        logLevel = logger.Info
    }
    
    db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
        Logger: logger.Default.LogMode(logLevel),
    })
    
    if err != nil {
        log.Fatal("db.connection_failed", "error", err.Error())
    }
    
    // Connection pooling
    sqlDB, _ := db.DB()
    sqlDB.SetMaxIdleConns(10)
    sqlDB.SetMaxOpenConns(50)
    sqlDB.SetConnMaxLifetime(time.Hour)
    
    log.Info("db.connected", "host", cfg.DBHost)
    return db
}

func HealthCheck(db *gorm.DB) error {
    sqlDB, err := db.DB()
    if err != nil {
        return err
    }
    return sqlDB.Ping()
}
```

### Commit

```
feat(core-backend): setup database connection with GORM

- Configure MySQL connection with DSN
- Setup connection pooling (10-50 connections)
- Enable SSL/TLS for production
- Add query logging (development only)
- Implement health check function
- Configure connection lifetime settings

Closes #52
```

---

## T3: JWT & Token Management

**Issue**: #53  
**Assignee**: danillobrito-sr  
**Status**: ⏳ Blocked  
**Tempo Estimado**: 3 horas  
**Dependências**: T1  

### Descrição

Implementar JWT token generation/validation:
- HS256 signing
- Expiry validation
- Token rotation
- Refresh tokens (opcional fase 1)

### Gate (Definição de Pronto)

- [ ] JWT token gerado corretamente
- [ ] Claims validadas (iss, exp, aud)
- [ ] Token inválido é rejeitado
- [ ] Token expirado é rejeitado
- [ ] Signing secret é carregado do .env
- [ ] Testes passam (valid/invalid/expired tokens)

### Implementação

```go
// pkg/jwt/jwt.go
package jwt

import (
    "errors"
    "time"
    
    "github.com/golang-jwt/jwt/v5"
)

type Claims struct {
    DriverID int64  `json:"driver_id"`
    Email    string `json:"email"`
    jwt.RegisteredClaims
}

type Manager struct {
    secret string
}

func NewManager(secret string) *Manager {
    return &Manager{secret: secret}
}

func (m *Manager) GenerateToken(driverID int64, email string, expiry time.Duration) (string, error) {
    now := time.Now()
    claims := Claims{
        DriverID: driverID,
        Email:    email,
        RegisteredClaims: jwt.RegisteredClaims{
            IssuedAt:  jwt.NewNumericDate(now),
            ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
            Issuer:    "pilot-app",
            Audience:  []string{"pilot-drivers"},
        },
    }
    
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    tokenString, err := token.SignedString([]byte(m.secret))
    if err != nil {
        return "", err
    }
    
    return tokenString, nil
}

func (m *Manager) ValidateToken(tokenString string) (*Claims, error) {
    claims := &Claims{}
    
    token, err := jwt.ParseWithClaims(
        tokenString,
        claims,
        func(token *jwt.Token) (interface{}, error) {
            if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
                return nil, errors.New("unexpected signing method")
            }
            return []byte(m.secret), nil
        },
    )
    
    if err != nil {
        return nil, err
    }
    
    if !token.Valid {
        return nil, errors.New("invalid token")
    }
    
    if time.Now().After(claims.ExpiresAt.Time) {
        return nil, errors.New("token expired")
    }
    
    return claims, nil
}
```

### Commit

```
feat(core-backend): implement JWT token management

- Add JWT token generation with HS256
- Implement token validation and expiry checking
- Define Claims structure with driver info
- Add manager with NewManager() constructor
- Create unit tests for valid/invalid/expired tokens

Closes #53
```

---

## T4: Models & Validation

**Issue**: #54  
**Assignee**: danillobrito-sr  
**Status**: ⏳ Blocked  
**Tempo Estimado**: 2 horas  
**Dependências**: T2, T3  

### Descrição

Criar models principais com validação:
- Driver
- Trip  
- DailyGoal
- PaymentRecord

Cada model tem método `Validate()`.

### Gate (Definição de Pronto)

- [ ] Driver model definido com campos corretos
- [ ] Trip model definido
- [ ] DailyGoal model definido
- [ ] PaymentRecord model definido
- [ ] Cada model tem método Validate()
- [ ] Validações testadas (email, phone, ratings)
- [ ] JSON marshaling funciona

### Commit

```
feat(core-backend): add data models with validation

- Create Driver model (GORM tags)
- Create Trip model (GORM tags)
- Create DailyGoal model (GORM tags)
- Create PaymentRecord model (GORM tags)
- Implement Validate() for each model
- Add JSON serialization tests

Closes #54
```

---

## T5: Repository Pattern (CRUD Layer)

**Issue**: #55  
**Assignee**: danillobrito-sr  
**Status**: ⏳ Blocked  
**Tempo Estimado**: 3 horas  
**Dependências**: T1  

### Descrição

Implementar repositories (data access layer):
- DriverRepository
- TripRepository
- GoalRepository
- PaymentRepository

Cada um com CRUD + queries específicas.

### Gate (Definição de Pronto)

- [ ] DriverRepository com GetByID, Create, Update
- [ ] TripRepository com GetByDateRange, Create
- [ ] GoalRepository com GetByDate, Create, Update
- [ ] PaymentRepository com GetByDateRange, Create
- [ ] Usar prepared statements (GORM handles isso)
- [ ] Testes de repository passam

### Commit

```
feat(core-backend): add repository layer for data access

- Create DriverRepository with CRUD operations
- Create TripRepository with GetByDateRange
- Create GoalRepository with GetByDate
- Create PaymentRepository with GetByDateRange
- Add interface for repository contract
- Implement prepared statements via GORM

Closes #55
```

---

## T7: Middleware & Authentication

**Issue**: #57  
**Assignee**: danillobrito-sr  
**Status**: ⏳ Blocked  
**Tempo Estimado**: 4 horas  
**Dependências**: T1  

### Descrição

Implementar middleware seguro:
- JWT Auth middleware
- Rate limiting middleware
- CORS middleware
- Error handler middleware
- Request logging middleware

### Gate (Definição de Pronto)

- [ ] JWT Auth valida token corretamente
- [ ] Token inválido retorna 401
- [ ] Rate limiting bloqueia após 100 req/min
- [ ] CORS whitelist apenas domínios permitidos
- [ ] Error handler não expõe stack trace
- [ ] Request logging em JSON (com request_id)
- [ ] Middleware chain funciona

### Commit

```
feat(core-backend): add security middleware

- Implement JWT authentication middleware
- Add rate limiting middleware (100 req/min)
- Configure CORS with domain whitelist
- Add global error handler (no stack trace in prod)
- Implement request logging with request_id
- Add middleware chain to all routes

Closes #57
```

---

## T8: Auth Handler & OAuth2

**Issue**: #58  
**Assignee**: danillobrito-sr  
**Status**: ⏳ Blocked  
**Tempo Estimado**: 3 horas  
**Dependências**: T7  

### Descrição

Implementar handlers de autenticação:
- POST /api/auth/uber-login (código → JWT)
- GET /api/auth/me (retorna motorista)
- POST /api/auth/logout (blacklist token)

### Gate (Definição de Pronto)

- [ ] POST /auth/uber-login valida código
- [ ] Token OAuth2 é encrypted antes de armazenar
- [ ] JWT retornado ao cliente
- [ ] GET /auth/me retorna motorista autenticado
- [ ] Dados sensíveis são redacted (Redact())
- [ ] POST /auth/logout blacklist token
- [ ] Testes de auth passam

### Commit

```
feat(core-backend): implement authentication handlers

- Add POST /api/auth/uber-login with OAuth2 flow
- Implement token encryption before storage
- Add GET /api/auth/me endpoint
- Implement POST /api/auth/logout with blacklist
- Add data redaction for sensitive fields
- Create integration tests for auth flow

Closes #58
```

---

## T9: Driver Handler & Profile

**Issue**: #59  
**Assignee**: danillobrito-sr  
**Status**: ⏳ Blocked  
**Tempo Estimado**: 3 horas  
**Dependências**: T7  

### Descrição

Implementar handlers para perfil do motorista:
- GET /api/drivers/me
- PUT /api/drivers/me
- POST /api/drivers/sync-profile

### Gate (Definição de Pronto)

- [ ] GET /drivers/me retorna motorista autenticado
- [ ] PUT /drivers/me atualiza dados
- [ ] POST /drivers/sync-profile sincroniza Uber
- [ ] Validação de inputs funciona
- [ ] Testes passam

### Commit

```
feat(core-backend): implement driver profile handlers

- Add GET /api/drivers/me endpoint
- Add PUT /api/drivers/me for updates
- Add POST /api/drivers/sync-profile for Uber sync
- Implement input validation
- Add profile update tests

Closes #59
```

---

## T10: Trips Handler & Sync

**Issue**: #60  
**Assignee**: danillobrito-sr  
**Status**: ⏳ Blocked  
**Tempo Estimado**: 3 horas  
**Dependências**: T7  

### Descrição

Implementar handlers para corridas:
- GET /api/trips (com paginação/filtro)
- GET /api/trips/:id
- POST /api/trips/sync

### Gate (Definição de Pronto)

- [ ] GET /trips lista com paginação
- [ ] GET /trips/:id retorna trip específica
- [ ] POST /trips/sync sincroniza da Uber API
- [ ] Filtros por data funcionam
- [ ] Testes passam

### Commit

```
feat(core-backend): implement trips handlers

- Add GET /api/trips with pagination
- Add GET /api/trips/:id endpoint
- Add POST /api/trips/sync for Uber sync
- Implement date filtering
- Add trip listing tests

Closes #60
```

---

## Fluxo de Execução

```
COMECE:
  └─ T1 (Setup básico)
     ├─ T2 (Database)
     ├─ T3 (JWT)
     └─ T5 (Repository)

DEPOIS:
  └─ T4 (Models - depende de T2 e T3)

SEGURANÇA:
  └─ T7 (Middleware - depende de T1)

DEPOIS:
  ├─ T8 (Auth - depende de T7)
  ├─ T9 (Driver - depende de T7)
  └─ T10 (Trips - depende de T7)

SINALIZE ANTES DE CONTINUAR:
  Se T2 (eliezersilva-spec) ainda não terminou T8-T12,
  aguarde antes de tentar implementar T14/T16+
```

---

## Commands Úteis

```bash
# Compilar
go build ./cmd/server

# Testes
go test ./... -v

# Coverage
go test ./... -cover

# Lint
golangci-lint run ./...

# Format
go fmt ./...

# Vet
go vet ./...

# Run (dev)
go run ./cmd/server

# Migrations (quando T2 estiver pronto)
go run ./cmd/migrate/main.go up
go run ./cmd/migrate/main.go down
```

---

## Notas de Segurança

✅ **Por favor, revisar:**
- Encryption de tokens (AES-256-GCM)
- JWT secret length (64 bytes mínimo)
- Database password em .env (nunca em código)
- Rate limiting configurado (100 req/min/driver)
- CORS whitelist apenas domínios conhecidos
- Error handling sem stack trace em prod
- Logging estruturado (JSON, sem secrets)

