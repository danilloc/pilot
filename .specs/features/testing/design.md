# Pilot — Testing Strategy

**Goal**: 80%+ code coverage  
**Tools**: Go: testify, Vitest (frontend)  
**CI/CD**: GitHub Actions  

---

## Testing Pyramid

```
         E2E Tests (10%)
           ↑
   Integration Tests (20%)
           ↑
       Unit Tests (70%)
```

---

## Backend Testing (Go)

### Unit Tests

**Auth Service**
```go
func TestLoginWithUberCode(t *testing.T) {
    // Test valid code
    // Test invalid code
    // Test token encryption
    // Test JWT generation
}

func TestTokenEncryption(t *testing.T) {
    // Test AES-256-GCM
    // Test invalid keys
}
```

**Stats Service**
```go
func TestCalculateDailyStats(t *testing.T) {
    // Test with 0 trips
    // Test with 10 trips
    // Test earnings calculation
    // Test cache hit
}
```

**Models Validation**
```go
func TestDriverValidation(t *testing.T) {
    // Test invalid email
    // Test invalid rating
    // Test valid driver
}
```

### Integration Tests

**Auth Flow**
```go
func TestOAuthLoginFlow(t *testing.T) {
    // 1. Simulate Uber OAuth code
    // 2. Call POST /auth/uber-login
    // 3. Verify JWT returned
    // 4. Verify driver in database
}
```

**Trip Sync**
```go
func TestTripSync(t *testing.T) {
    // 1. Mock Uber API
    // 2. Call POST /trips/sync
    // 3. Verify trips in database
    // 4. Verify stats calculated
}
```

**Stats Calculation**
```go
func TestStatsCalculation(t *testing.T) {
    // 1. Insert test trips
    // 2. Call GET /stats/today
    // 3. Verify totals
    // 4. Verify per-hour calculation
}
```

### End-to-End Tests

**Complete User Journey**
```go
func TestCompleteUserJourney(t *testing.T) {
    // 1. Login (OAuth)
    // 2. Sync trips from Uber
    // 3. Set daily goal
    // 4. Check stats
    // 5. Logout
}
```

---

## Frontend Testing (Vue/Nuxt)

### Component Unit Tests

```typescript
describe('EarningsCard.vue', () => {
  it('displays correct earnings', () => {
    const wrapper = mount(EarningsCard, {
      props: { totalEarned: 450.75 }
    })
    expect(wrapper.text()).toContain('R$ 450,75')
  })

  it('shows trip count', () => {
    const wrapper = mount(EarningsCard, {
      props: { totalTrips: 8 }
    })
    expect(wrapper.text()).toContain('8 corridas')
  })
})
```

### Composable Tests

```typescript
describe('useStats', () => {
  it('fetches daily stats', async () => {
    const { stats, fetchDailyStats } = useStats()
    await fetchDailyStats()
    expect(stats.value).toBeDefined()
  })

  it('handles API errors', async () => {
    const { error, fetchDailyStats } = useStats()
    // Mock error
    await fetchDailyStats()
    expect(error.value).toBeDefined()
  })
})
```

### Store Tests

```typescript
describe('useAuthStore', () => {
  it('sets token', () => {
    const store = useAuthStore()
    store.setToken('jwt-token')
    expect(store.token).toBe('jwt-token')
  })

  it('clears auth on logout', () => {
    const store = useAuthStore()
    store.setToken('jwt-token')
    store.clearAuth()
    expect(store.token).toBeNull()
  })
})
```

### E2E Tests

```typescript
describe('User Authentication Flow', () => {
  it('logs in with OAuth', async () => {
    await page.goto('http://localhost:3000/login')
    await page.click('[data-testid="uber-login"]')
    // Simulate OAuth flow
    await page.waitForNavigation()
    expect(page.url()).toContain('localhost:3000')
  })

  it('displays dashboard after login', async () => {
    // Login first
    await page.goto('http://localhost:3000')
    await expect(page.locator('[data-testid="earnings-card"]')).toBeVisible()
  })
})
```

---

## Bot Testing (n8n)

### Message Parsing

```typescript
test('parse /comecou_jornada command', () => {
  const msg = '/comecou_jornada'
  const cmd = parseCommand(msg)
  expect(cmd).toBe('comecou_jornada')
})

test('extract driver ID from WhatsApp', () => {
  const phoneId = '5511999999999'
  const driverId = getDriverId(phoneId)
  expect(driverId).toBeNumber()
})
```

### Workflow Tests

```typescript
test('journey workflow sends confirmation', async () => {
  const result = await runWorkflow('comecou_jornada', {
    phone_id: '5511999999999'
  })
  
  expect(result.message_sent).toBe(true)
  expect(result.journey_id).toBeDefined()
})
```

---

## Database Testing

### Migration Tests

```go
func TestMigrations(t *testing.T) {
    // Test each migration up
    // Test rollback
    // Verify schema after each
}

func TestIndexes(t *testing.T) {
    // Verify indexes created
    // Test index performance
}
```

### Query Tests

```go
func TestDailyEarningsQuery(t *testing.T) {
    // Insert test data
    // Run query
    // Verify results
    // Check performance (< 100ms)
}
```

---

## API Testing

### REST Endpoint Tests

```bash
# Test auth
curl -X POST http://localhost:8080/api/auth/uber-login \
  -H "Content-Type: application/json" \
  -d '{"code": "valid_code"}'

# Test protected route
curl -X GET http://localhost:8080/api/drivers/me \
  -H "Authorization: Bearer <jwt_token>"

# Test rate limiting
for i in {1..101}; do
  curl -X GET http://localhost:8080/api/stats/today \
    -H "Authorization: Bearer <jwt_token>"
done
# 101st should return 429
```

---

## Load Testing

### k6 Tests

```javascript
import http from 'k6/http'
import { check, sleep } from 'k6'

export let options = {
  stages: [
    { duration: '30s', target: 20 },   // Warm-up
    { duration: '1m30s', target: 100 }, // Ramp-up
    { duration: '1m', target: 100 },   // Stay
    { duration: '30s', target: 0 },    // Ramp-down
  ]
}

export default function () {
  let res = http.get('http://localhost:8080/api/stats/today', {
    headers: {
      'Authorization': `Bearer ${__ENV.JWT_TOKEN}`
    }
  })
  
  check(res, {
    'status is 200': (r) => r.status === 200,
    'response time < 500ms': (r) => r.timings.duration < 500
  })
  
  sleep(1)
}
```

---

## Security Testing

### OWASP Top 10

```
✅ SQL Injection: Prepared statements
✅ XSS: HTML escaping
✅ CSRF: Token validation
✅ Auth bypass: JWT validation
✅ Sensitive data: Encryption
✅ XML entities: Not applicable
✅ Broken access control: Role checking
✅ Insecure deserialization: Not applicable
✅ Injection: Input validation
✅ Insecure logging: No secrets logged
```

### HTTPS/TLS Testing

```bash
# Test TLS version
openssl s_client -connect api.pilot.com:443 -tls1_3

# Test certificate
openssl x509 -in cert.pem -text -noout
```

---

## Test Configuration

### Backend (Go)

```toml
# .github/workflows/test-backend.yml
name: Backend Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    services:
      mysql:
        image: mysql:8
        env:
          MYSQL_ROOT_PASSWORD: root
    
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-go@v2
        with:
          go-version: '1.21'
      - run: go test ./... -v -cover
      - run: go tool cover -func=coverage.out
```

### Frontend (Vue)

```javascript
// vitest.config.ts
import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  test: {
    globals: true,
    environment: 'jsdom',
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html'],
      exclude: ['node_modules/', 'dist/']
    }
  }
})
```

---

## Coverage Goals

```
Backend (Go):
├─ Handlers: 80%
├─ Services: 85%
├─ Repositories: 90%
├─ Models: 95%
└─ Overall: 80%+

Frontend (Vue):
├─ Components: 75%
├─ Composables: 85%
├─ Stores: 90%
└─ Overall: 70%+

Bot (n8n):
├─ Workflows: 80%
└─ Overall: 80%+
```

