# Pilot — Core Backend Design

**Feature**: `core-backend`  
**Status**: 🟡 In Design  
**Segurança**: 🔒 CRÍTICA

---

## Arquitetura Geral

```
┌─────────────────────────────────────────────┐
│           Frontend (Nuxt PWA)               │
└──────────────────┬──────────────────────────┘
                   │ HTTPS only
                   ▼
┌─────────────────────────────────────────────┐
│    API Gateway (Rate Limit + CORS)          │
├─────────────────────────────────────────────┤
│  - TLS 1.3+                                 │
│  - Rate limiting (Redis)                    │
│  - Request validation                       │
│  - Response encryption (opcional)           │
└──────────────┬──────────────────────────────┘
               │
      ┌────────┴────────┐
      ▼                 ▼
┌────────────┐    ┌──────────────┐
│   Auth     │    │   Handlers   │
│  Service   │    │   (Routes)   │
└─────┬──────┘    └───────┬──────┘
      │                   │
      └────────┬──────────┘
               ▼
        ┌──────────────┐
        │  Services    │
        │ (Business)   │
        └──────┬───────┘
               ▼
        ┌──────────────┐
        │Repositories  │
        │  (Database)  │
        └──────┬───────┘
               ▼
        ┌──────────────┐
        │    MySQL     │
        │   (Encrypted)│
        └──────────────┘
```

---

## 🔐 Segurança: Pilares Principais

### 1. **Autenticação & Autorização**

```go
// JWT + OAuth2
├─ OAuth2 Uber (motorista login)
├─ JWT tokens (24h expiry)
├─ Refresh tokens (7 dias, rotacionados)
├─ Token blacklist (Redis)
└─ Invalidação imediata (logout)

// Validação
├─ Scope checking
├─ Rate limiting por driver
├─ IP whitelisting (opcional)
└─ Device fingerprinting (futuro)
```

### 2. **Criptografia de Dados Sensíveis**

```
Dados Sensíveis:
├─ oauth_token → AES-256-GCM (chaveia com app secret)
├─ oauth_refresh_token → AES-256-GCM
├─ phone → hash + salt (search não é necessário)
└─ payment_method → vaulted (3rd party)

Key Management:
├─ Master key em environment variable
├─ Rotation strategy (90 dias)
├─ Separate keys por ambiente (prod/staging/dev)
└─ Key audit logging
```

### 3. **API Security**

```
Requests:
├─ HTTPS only (TLS 1.3+)
├─ CORS whitelist (domínios conhecidos)
├─ Content-Type validation
├─ Payload size limit (1MB)
└─ Request ID tracing (logging)

Responses:
├─ Sem secrets em erro (não expor stack trace)
├─ Security headers (HSTS, X-Frame-Options, CSP)
├─ Rate limiting por endpoint
├─ Response timeout (30s max)
└─ Sanitização de output
```

### 4. **Database Security**

```
MySQL:
├─ Connection via encrypted tunnel (SSL)
├─ Prepared statements (previne SQL injection)
├─ Least privilege user (app user ≠ admin)
├─ Backup encrypted (AES-256)
├─ Audit logging (quem acessou o quê)
└─ Masking de dados sensíveis em logs

Data:
├─ Encryption at rest (optional, por tabela)
├─ Deletion strategy (soft-delete + 90 dias)
├─ GDPR compliance (right to be forgotten)
└─ Data isolation por driver (não vê outros)
```

### 5. **Input Validation & Sanitization**

```
Validação:
├─ Email format (RFC 5322)
├─ Phone format (E.164)
├─ UUID validation
├─ Date/time validation
├─ Numeric ranges (não aceita -999999999)
└─ String length limits

Sanitização:
├─ Trim whitespace
├─ Escape special chars
├─ Remove null bytes
├─ HTML escape (XSS prevention)
└─ SQL escape (prepared statements)
```

### 6. **Logging & Monitoring**

```
O QUE LOGAR:
✅ Login attempts (sucesso/falha)
✅ API errors (código, mensagem genérica)
✅ Database queries (sem valores sensíveis)
✅ Rate limit violations
✅ Suspicious patterns (múltiplas falhas)

O QUE NÃO LOGAR:
❌ Tokens (JWT, OAuth, refresh)
❌ Senhas ou secrets
❌ Phone numbers (masked ok)
❌ Valores de corridas (pode revelar localização)
❌ Full stack traces em produção

Formato:
{
  "timestamp": "2024-08-01T15:30:00Z",
  "event": "auth.login.success",
  "driver_id": 12345,
  "user_agent": "Mozilla/5.0...",
  "ip": "192.168.1.1",
  "severity": "INFO"
}
```

### 7. **Third-Party Integrations**

```
Uber API:
├─ Token armazenado encrypted
├─ Refresh automático antes de expirar
├─ Timeout (30s)
├─ Retry com backoff exponencial
├─ Rate limit respect (não sobrecarregar)
└─ Error handling sem expor detalhes

WhatsApp Business API:
├─ Webhook signature validation (HMAC-SHA256)
├─ Message encryption (E2E)
├─ Rate limiting (não enviar 1000 msgs/s)
├─ Audit log (quem mandou o quê)
└─ Unsubscribe handling (GDPR)
```

### 8. **Error Handling**

```
Production:
├─ Erro genérico pro cliente: {"error": "Something went wrong"}
├─ Error code único: "AUTH_001_INVALID_TOKEN"
├─ Detalhes full em logs (só pra dev)
└─ Stack trace apenas em staging

Development:
├─ Stack trace completo
├─ Query logging
├─ Performance metrics
└─ Debug mode
```

---

## Models Com Segurança

### Driver Model

```go
type Driver struct {
    ID                   int64
    UUID                 string    // Public ID (não ID interno)
    UberID               string    // Único, encrypted no banco
    
    Name                 string
    Email                string    // Unique, hashed
    Phone                string    // E164, hash only
    ProfilePictureURL    string    // Validado (no malware)
    
    Rating               float64
    TotalTrips           int
    AccountStatus        string    // ACTIVE, SUSPENDED, BANNED
    
    OAuthToken           string    // AES-256-GCM encrypted
    OAuthRefreshToken    string    // AES-256-GCM encrypted
    TokenExpiresAt       time.Time
    
    CreatedAt            time.Time
    UpdatedAt            time.Time
    LastSyncedAt         *time.Time
    
    // Campos de segurança (não exponho via API)
    FailedLoginAttempts  int       `json:"-"`
    LastLoginIP          string    `json:"-"` // Audit
    LastLoginTime        time.Time `json:"-"`
    IsBanned             bool      `json:"-"`
    BanReason            string    `json:"-"`
}

// Método: Redact (remove dados sensíveis antes de retornar)
func (d *Driver) Redact() {
    d.OAuthToken = ""
    d.OAuthRefreshToken = ""
    d.FailedLoginAttempts = 0
    d.LastLoginIP = ""
}

// Método: Validate
func (d *Driver) Validate() error {
    // Validações de negócio
    if !isValidEmail(d.Email) {
        return ErrInvalidEmail
    }
    if !isValidE164(d.Phone) {
        return ErrInvalidPhone
    }
    if d.Rating < 0 || d.Rating > 5 {
        return ErrInvalidRating
    }
    if d.IsBanned {
        return ErrDriverBanned
    }
    return nil
}
```

### Trip Model

```go
type Trip struct {
    ID           int64
    UUID         string    // Public ID
    DriverID     int64     // Foreign key
    UberTripID   string    // Unique, não expor internamente
    
    StartedAt    time.Time
    EndedAt      time.Time
    DistanceKM   float64
    FareValue    float64
    Currency     string
    
    City         string
    Status       string    // COMPLETED, CANCELLED, NO_SHOW
    
    // Coordenadas: não retornar ao cliente (privacidade)
    StartLat     float64 `json:"-"`
    StartLng     float64 `json:"-"`
    EndLat       float64 `json:"-"`
    EndLng       float64 `json:"-"`
    
    CreatedAt    time.Time
    UpdatedAt    time.Time
}

// Validações
func (t *Trip) Validate() error {
    if t.DistanceKM <= 0 {
        return ErrInvalidDistance
    }
    if t.FareValue < 0 {
        return ErrInvalidFare
    }
    if t.EndedAt.Before(t.StartedAt) {
        return ErrInvalidTimeRange
    }
    return nil
}
```

---

## Services (Business Logic)

### Auth Service

```go
type AuthService struct {
    driverRepo    *repository.DriverRepository
    jwtManager    *jwt.Manager
    encryptor     *crypto.Encryptor
    logger        *log.Logger
    cache         *cache.Redis
}

// LoginWithUberCode: Secure OAuth flow
func (s *AuthService) LoginWithUberCode(ctx context.Context, code string) (string, *Driver, error) {
    // 1. Validar code (não vazio, válido)
    if code == "" {
        s.logger.Warn("empty_code_provided")
        return "", nil, ErrInvalidCode
    }
    
    // 2. Exchange code por token (timeout 10s)
    ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
    defer cancel()
    
    token, err := s.oauthCfg.Exchange(ctx, code)
    if err != nil {
        s.logger.Error("oauth_exchange_failed", "error", err.Error())
        return "", nil, ErrOAuthFailed
    }
    
    // 3. Validar token expiração
    if token.Expiry.Before(time.Now()) {
        return "", nil, ErrTokenExpired
    }
    
    // 4. Fetch profile (timeout 10s)
    ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
    defer cancel()
    
    profile, err := oauth.GetUberProfile(s.oauthCfg, token)
    if err != nil {
        s.logger.Error("uber_profile_fetch_failed", "error", err.Error())
        return "", nil, ErrProfileFetchFailed
    }
    
    // 5. Validar profile
    if err := profile.Validate(); err != nil {
        return "", nil, err
    }
    
    // 6. Encrypt tokens
    encryptedAccessToken, err := s.encryptor.Encrypt(token.AccessToken)
    if err != nil {
        s.logger.Error("token_encryption_failed")
        return "", nil, ErrEncryptionFailed
    }
    
    encryptedRefreshToken, err := s.encryptor.Encrypt(token.RefreshToken)
    if err != nil {
        s.logger.Error("refresh_token_encryption_failed")
        return "", nil, ErrEncryptionFailed
    }
    
    // 7. Upsert driver
    driver := &Driver{
        UberID:               profile.DriverID,
        Name:                 profile.FirstName + " " + profile.LastName,
        Email:                profile.Email,
        Phone:                profile.MobilePhoneNumber,
        ProfilePictureURL:    profile.PictureURL,
        Rating:               profile.Rating,
        TotalTrips:           profile.TripCount,
        AccountStatus:        "ACTIVE",
        OAuthToken:           encryptedAccessToken,
        OAuthRefreshToken:    encryptedRefreshToken,
        TokenExpiresAt:       token.Expiry,
        FailedLoginAttempts:  0,
        LastLoginIP:          ctx.Value("client_ip").(string),
        LastLoginTime:        time.Now(),
    }
    
    if err := s.driverRepo.UpsertDriver(ctx, driver); err != nil {
        s.logger.Error("driver_upsert_failed", "error", err.Error())
        return "", nil, ErrDatabaseError
    }
    
    // 8. Gerar JWT
    jwtToken, err := s.jwtManager.GenerateToken(driver.ID, 24*time.Hour)
    if err != nil {
        s.logger.Error("jwt_generation_failed")
        return "", nil, ErrJWTGenerationFailed
    }
    
    // 9. Log success
    s.logger.Info("login_success", "driver_id", driver.ID, "uber_id", driver.UberID)
    
    // 10. Redact sensitive data
    driver.Redact()
    
    return jwtToken, driver, nil
}

// ValidateToken: Check JWT + blacklist
func (s *AuthService) ValidateToken(token string) (*jwt.Claims, error) {
    // 1. Check blacklist (Redis)
    blacklisted, err := s.cache.IsBlacklisted(token)
    if err != nil {
        s.logger.Error("blacklist_check_failed")
        return nil, ErrInternalError
    }
    if blacklisted {
        return nil, ErrTokenBlacklisted
    }
    
    // 2. Validate JWT signature
    claims, err := s.jwtManager.ValidateToken(token)
    if err != nil {
        s.logger.Warn("invalid_token_provided", "error", err.Error())
        return nil, ErrInvalidToken
    }
    
    return claims, nil
}

// Logout: Blacklist token
func (s *AuthService) Logout(token string) error {
    expiresIn := time.Until(time.Now().Add(24 * time.Hour))
    if err := s.cache.Blacklist(token, expiresIn); err != nil {
        s.logger.Error("logout_failed")
        return ErrLogoutFailed
    }
    
    s.logger.Info("logout_success")
    return nil
}
```

---

## Middleware Security

### JWT Auth Middleware

```go
func JWTAuth(jwtMgr *jwt.Manager) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. Extract token
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.JSON(401, gin.H{"error": "missing_token"})
            c.Abort()
            return
        }
        
        parts := strings.Split(authHeader, " ")
        if len(parts) != 2 || parts[0] != "Bearer" {
            c.JSON(401, gin.H{"error": "invalid_token_format"})
            c.Abort()
            return
        }
        
        token := parts[1]
        
        // 2. Validate
        claims, err := jwtMgr.ValidateToken(token)
        if err != nil {
            c.JSON(401, gin.H{"error": "invalid_token"})
            c.Abort()
            return
        }
        
        // 3. Check expiry
        if time.Now().After(claims.ExpiresAt) {
            c.JSON(401, gin.H{"error": "token_expired"})
            c.Abort()
            return
        }
        
        // 4. Add to context (use driver_id, not token)
        c.Set("driver_id", claims.DriverID)
        c.Set("client_ip", c.ClientIP())
        
        c.Next()
    }
}
```

### Rate Limiting Middleware

```go
func RateLimit(cache *cache.Redis) gin.HandlerFunc {
    return func(c *gin.Context) {
        driverID := c.GetInt64("driver_id")
        endpoint := c.Request.URL.Path
        
        key := fmt.Sprintf("rate_limit:%d:%s", driverID, endpoint)
        
        count, err := cache.Incr(key)
        if err != nil {
            c.AbortWithStatus(500)
            return
        }
        
        // 100 requests per minute per driver per endpoint
        if count == 1 {
            cache.Expire(key, 60*time.Second)
        }
        
        if count > 100 {
            c.JSON(429, gin.H{"error": "too_many_requests"})
            c.Abort()
            return
        }
        
        c.Set("rate_limit_remaining", 100-count)
        c.Next()
    }
}
```

---

## Error Handling

```go
type APIError struct {
    Code      string `json:"code"`
    Message   string `json:"message"`
    RequestID string `json:"request_id"`
    Timestamp string `json:"timestamp"`
}

// Custom errors (nunca expor stack trace)
var (
    ErrInvalidToken       = APIError{Code: "AUTH_001", Message: "Invalid token"}
    ErrTokenExpired       = APIError{Code: "AUTH_002", Message: "Token expired"}
    ErrTokenBlacklisted   = APIError{Code: "AUTH_003", Message: "Token no longer valid"}
    ErrOAuthFailed        = APIError{Code: "AUTH_004", Message: "Authentication failed"}
    ErrDriverBanned       = APIError{Code: "AUTH_005", Message: "Account suspended"}
    ErrInvalidEmail       = APIError{Code: "VAL_001", Message: "Invalid email"}
    ErrInvalidPhone       = APIError{Code: "VAL_002", Message: "Invalid phone"}
    ErrInvalidRating      = APIError{Code: "VAL_003", Message: "Invalid rating"}
    ErrDatabaseError      = APIError{Code: "DB_001", Message: "Database error"}
    ErrInternalError      = APIError{Code: "ERR_500", Message: "Internal server error"}
)

// Global error handler
func ErrorHandler(c *gin.Context) {
    c.Next()
    
    if len(c.Errors) > 0 {
        err := c.Errors.Last()
        
        // Log full error (with stack trace)
        logger.Error("request_error", "error", err.Error(), "path", c.Request.URL.Path)
        
        // Return generic error to client
        c.JSON(500, APIError{
            Code:      "ERR_500",
            Message:   "Something went wrong",
            RequestID: c.GetString("request_id"),
            Timestamp: time.Now().Format(time.RFC3339),
        })
    }
}
```

---

## Dependências

```
Backend:
├─ github.com/gin-gonic/gin (web framework)
├─ gorm.io/gorm (ORM, com prepared statements)
├─ gorm.io/driver/mysql
├─ github.com/golang-jwt/jwt/v5 (JWT)
├─ golang.org/x/oauth2 (OAuth2)
├─ github.com/redis/go-redis/v9 (cache + blacklist)
├─ github.com/joho/godotenv (.env loading)
├─ go.uber.org/zap (structured logging)
├─ golang.org/x/crypto (encryption)
├─ github.com/google/uuid (UUID generation)
└─ github.com/urfave/cli/v2 (CLI tools)

Segurança:
├─ crypto/aes (AES-256-GCM)
├─ crypto/rand (random generation)
├─ crypto/hmac (HMAC signing)
└─ golang.org/x/crypto/bcrypt (password hashing, se usar)
```

---

## Environment Variables (Segurança)

```env
# Server
PORT=8080
GIN_MODE=release  # NUNCA "debug" em produção
ENVIRONMENT=production

# Database
DB_HOST=mysql
DB_PORT=3306
DB_USER=pilot_app  # Least privilege user
DB_PASSWORD=<super-secure-password>
DB_NAME=pilot_db
DB_SSLMODE=require  # Encrypted connection

# Encryption
ENCRYPTION_KEY=<32-byte-key-base64>  # AES-256
ENCRYPTION_SALT=<16-byte-salt-base64>

# JWT
JWT_SECRET=<64-byte-secret-base64>  # HS256
JWT_EXPIRY=24h

# OAuth2
UBER_CLIENT_ID=<from-developer.uber.com>
UBER_CLIENT_SECRET=<keep-secret>
UBER_REDIRECT_URI=https://app.pilot.com/auth/callback

# Redis (cache + rate limit + blacklist)
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=<if-required>
REDIS_DB=0

# WhatsApp (fase 2)
WHATSAPP_BUSINESS_API_KEY=<token>
WHATSAPP_WEBHOOK_SECRET=<HMAC-secret>

# Logging
LOG_LEVEL=info  # info, warn, error
LOG_FORMAT=json

# Security
CORS_ORIGINS=https://app.pilot.com,https://staging.pilot.com
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS=100
RATE_LIMIT_WINDOW=60

# Sentry (error tracking)
SENTRY_DSN=<sentry-project-dsn>
```

