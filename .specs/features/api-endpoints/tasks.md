# Pilot — API Tasks

**Feature**: `api-endpoints`  
**Branch**: `features/api-endpoints`  
**Assignee**: danilloc  

---

## Task List

| T# | Issue | Task | Deps | Time |
|----|-------|------|------|------|
| T15 | #65 | ⏭️ Auth handlers (login/me/logout) — already built in core-backend | T8 | 2h |
| T16 | #66 | ⏭️ Driver handlers (profile/sync) — already built in core-backend | T9 | 2h |
| T17 | #67 | ⏭️ Trips handlers (list/get/sync) — already built in core-backend | T10 | 2h |
| T18 | #68 | Stats handlers (daily/week/month) | T4 | 2h |
| T19 | #69 | Goals handlers (CRUD) | T4 | 2h |
| T20 | #70 | Payments handlers (list/sync) | T5 | 2h |
| T21 | #71 | Routes & middleware chain | T7 | 2h |
| T22 | #72 | Error handling & validation | T7 | 1h |
| T23 | #73 | API tests (unit & integration) | All | 3h |

---

## T15: Auth Handlers

**Issue**: #65  
**Deps**: T8  
**Status**: ⏭️ Skipped — already implemented in `core-backend` T8 (same design.md, same endpoints). Verified item-by-item against the real code and a live test run before skipping, not assumed:
- `internal/handler/auth_handler.go:30,41-55` + `TestAuthHandler_UberLogin_Success` — PASS
- Encrypt-before-persist order confirmed: `internal/service/auth_service.go:82-91` (encrypt) precedes line 114 (`UpsertDriver`)
- JWT returned: `auth_handler.go:54` returns the token from `auth_service.go:128`
- `GET /auth/me`: `TestAuthHandler_Me_ReturnsAuthenticatedDriver` — PASS
- Logout blacklist: `auth_service.go:153` `redis.Blacklist(...)` + `TestAuthHandler_Logout_ThenTokenRejected` (confirms the token stops working afterward) — PASS
- 12/12 real tests passing (5 handler + 7 service) against live MySQL/Redis, re-run 2026-08-02

**Time**: 2h  

### Gate

- [x] POST /auth/uber-login funciona
- [x] Token encrypted antes de armazenar
- [x] JWT retornado ao cliente
- [x] GET /auth/me retorna motorista
- [x] POST /auth/logout blacklist token
- [x] Tests passam (valid/invalid codes)

### Commit

```
feat(api): add authentication endpoints

- Implement POST /auth/uber-login
- Implement GET /auth/me
- Implement POST /auth/logout
- Add token encryption before storage
- Add auth integration tests

Closes #65
```

---

## T16: Driver Handlers

**Issue**: #66  
**Deps**: T9  
**Status**: ⏭️ Skipped — already implemented in `core-backend` T9. Verified against real code + live test run:
- `internal/handler/driver_handler.go:27-29` registers GET/PUT `/me`, POST `/sync-profile`
- `TestDriverHandler_UpdateMe_InvalidPhone` + `TestDriverService_UpdateProfile_InvalidPhone` — input validation confirmed (rejects malformed phone)
- 10/10 real tests passing (5 handler + 5 service), re-run 2026-08-02

**Time**: 2h  

### Gate

- [x] GET /drivers/me funciona
- [x] PUT /drivers/me atualiza
- [x] POST /drivers/sync-profile sincroniza Uber
- [x] Validação de inputs
- [x] Tests passam

### Commit

```
feat(api): add driver profile endpoints

- Implement GET /drivers/me
- Implement PUT /drivers/me
- Implement POST /drivers/sync-profile
- Add input validation
- Add driver tests

Closes #66
```

---

## T17: Trips Handlers

**Issue**: #67  
**Deps**: T10  
**Status**: ⏭️ Skipped — already implemented in `core-backend` T10. Verified against real code + live test run:
- `internal/handler/trip_handler.go:41-57,60-74,81-103` — list (paginated), get by id, sync
- Filters: `parseDateRange` (start_date/end_date) + status passed through to `tripRepo.GetByDateRange`
- 9/9 real tests passing (4 handler + 5 service), re-run 2026-08-02
- Note: the Uber trip field mapping used by `SyncTrips` was written before the real Uber payload was confirmed (see `uber-integration.md`) and is corrected as part of this branch's Uber client rework (T20 area), not re-litigated here since the handler/service/test wiring itself is unchanged and still passes.

**Time**: 2h  

### Gate

- [x] GET /trips lista com paginação
- [x] GET /trips/:id retorna trip
- [x] POST /trips/sync sincroniza Uber
- [x] Filtros funcionam
- [x] Tests passam

### Commit

```
feat(api): add trips endpoints

- Implement GET /trips with pagination
- Implement GET /trips/:id
- Implement POST /trips/sync
- Add date filtering
- Add trips tests

Closes #67
```

---

## T18: Stats Handlers

**Issue**: #68  
**Deps**: T4  
**Status**: ✅ Done  
**Time**: 2h  

### Gate

- [x] GET /stats/today calcula stats
- [x] GET /stats/week agrega diário
- [x] GET /stats/month agrega semana
- [x] Cache funciona (5 min)
- [x] Tests passam

### Commit

```
feat(api): add stats endpoints

- Implement GET /stats/today
- Implement GET /stats/week
- Implement GET /stats/month
- Add stats caching
- Add stats tests

Closes #68
```

---

## T19: Goals Handlers

**Issue**: #69  
**Deps**: T4  
**Status**: ✅ Done  
**Time**: 2h  

### Gate

- [x] POST /goals cria meta
- [x] GET /goals retorna meta
- [x] GET /goals/progress calcula progresso
- [x] PUT /goals/:id atualiza
- [x] DELETE /goals/:id marca ABANDONED
- [x] Tests passam

### Commit

```
feat(api): add goals endpoints

- Implement POST /goals
- Implement GET /goals
- Implement GET /goals/progress
- Implement PUT /goals/:id
- Implement DELETE /goals/:id
- Add goals tests

Closes #69
```

---

## T20: Payments Handlers

**Issue**: #70  
**Deps**: T5  
**Status**: ✅ Done  
**Time**: 2h  

### Gate

- [x] GET /payments lista pagamentos
- [x] POST /payments/sync sincroniza Uber
- [x] Paginação funciona
- [x] Tests passam

**Nota**: esta task também reformou `internal/oauth` inteiro (UberClient) pra
usar o mapeamento confirmado em `uber-integration.md` (`MockUberClient` +
`RealUberClient` atrás da mesma interface, `UBER_USE_MOCK=true` por
padrão), já que T17 (trips, feita antes desse mapeamento existir) e T20
compartilham o mesmo client. Corrigido no processo: conversão milhas→km
que faltava em `TripService.SyncTrips`, e mapeamento de status Uber
(`completed`/`driver_canceled`/`rider_canceled`→terminal,
`accepted`/`arriving`/`in_progress`→não sincroniza ainda).

### Commit

```
feat(api): add payments endpoints

- Implement GET /payments
- Implement POST /payments/sync
- Add pagination
- Add payments tests

Closes #70
```

---

## T21: Routes & Middleware

**Issue**: #71  
**Deps**: T7  
**Status**: ✅ Done  
**Time**: 2h  

### Gate

- [x] Todas rotas públicas funcionam
- [x] Todas rotas protegidas exigem JWT
- [x] CORS whitelist funciona
- [x] Rate limiting funciona — reworked from a single global 100 req/min
      into per-category limiters matching design.md's Rate Limiting table
      exactly (Auth 5/min, Trips 30/min, Stats 60/min, Goals 30/min;
      Drivers/Payments keep the general default since design.md doesn't
      list them). Verified via `X-RateLimit-Limit` header per category
      plus an end-to-end 6-requests-blocked-on-the-6th test for Auth.
- [x] Request logging JSON

### Commit

```
feat(api): setup routes and middleware chain

- Register all public routes
- Register all protected routes
- Setup middleware chain
- Configure CORS whitelist
- Add request logging

Closes #71
```

---

## T22: Error Handling

**Issue**: #72  
**Deps**: T7  
**Status**: ✅ Done  
**Time**: 1h  

### Gate

- [x] Erro genérico pra cliente
- [x] Request ID em response
- [x] Stack trace só em dev — real gap found: `zap.NewProductionConfig()`
      attaches a stacktrace to every Error-level log regardless of
      environment. Fixed by making `logger.New` take an `environment`
      param and disabling stacktrace capture unless it's `"development"`.
- [x] Validação de inputs funciona
- [x] Error codes padrão — AUTH_/VAL_/DB_/ERR_ namespaces, consistent
      `ErrorCode.StatusCode()` mapping

### Commit

```
feat(api): implement error handling

- Add global error handler
- Implement request ID tracking
- Add input validation middleware
- Define error code standards
- Hide stack traces in production

Closes #72
```

---

## T23: API Tests

**Issue**: #73  
**Time**: 3h  
**Status**: ✅ Done — verified with a real test run against live MySQL/Redis, not assumed:
- Unit tests (handlers): all 6 handlers (auth, driver, trip, stats, goal, payment) have request-level tests, including error-path branches (invalid body/date/id, driver-not-found via a "ghost" JWT for a non-existent driver_id, invalid query params) added in this pass to `trip_handler_test.go`, `driver_handler_test.go`, `payment_handler_test.go`, `auth_handler_test.go`, `goal_handler_test.go`
- Integration tests (end-to-end): sync→list round trips against real MySQL through the full router (`TestTripHandler_SyncThenList`, `TestPaymentHandler_SyncThenList`, etc.)
- Auth flow: `TestAuthHandler_UberLogin_Success`, `TestAuthHandler_Me_*`, `TestAuthHandler_Logout_ThenTokenRejected` (blacklist)
- Rate limiting: `TestRouter_RateLimitersMatchDesignDoc`, `TestRouter_AuthRateLimit_BlocksAfterFive` (T21)
- Coverage > 80%: **81.3%** aggregate (`go test ./... -coverpkg=./... -coverprofile=coverage.out` then `go tool cover -func=coverage.out`), up from 67.7% at the start of this pass. `cmd/server`'s `main()` (0%) is excluded as not safely unit-testable (contains `log.Fatalw`/`os.Exit`).
- `go build ./...`, `go vet ./...`, and the full `go test ./...` suite (all packages) pass clean, 0 failures.
- Bug found and fixed while closing this gap: `StatsService.Today`/`Week` (`internal/service/stats_service.go`) used MySQL's `CURDATE()` (server timezone, UTC in this environment) as the "today" boundary while `ended_at` is written using the app server's local wall-clock time (UTC-3 in this sandbox) — for a ~3h window around UTC midnight, the DB's "today" and the app's "today" diverged, silently dropping same-day trips from `/api/stats/today` and `/api/stats/week`. Fixed by computing day boundaries in Go and passing them as parameters (matching the pattern `Month()` already used, and the same sargable half-open-range shape from the database slice's fix), instead of relying on the DB's own clock.
- **Pending**: per the tlc-spec-driven skill, a fresh independent Verifier sub-agent (author ≠ verifier) must run after this task and produce `.specs/features/api-endpoints/validation.md` before the feature is declared complete — not yet dispatched.

### Gate

- [x] Unit tests (handlers)
- [x] Integration tests (end-to-end)
- [x] Auth flow tested
- [x] Rate limiting tested
- [x] Coverage > 80%

### Commit

```
feat(api): add comprehensive tests

- Add unit tests for all handlers
- Add integration tests for flows
- Add auth tests
- Add rate limit tests
- Reach 80%+ coverage

Closes #73
```

