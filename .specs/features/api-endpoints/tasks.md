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
**Time**: 2h  

### Gate

- [ ] GET /stats/today calcula stats
- [ ] GET /stats/week agrega diário
- [ ] GET /stats/month agrega semana
- [ ] Cache funciona (5 min)
- [ ] Tests passam

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
**Time**: 2h  

### Gate

- [ ] POST /goals cria meta
- [ ] GET /goals retorna meta
- [ ] GET /goals/progress calcula progresso
- [ ] PUT /goals/:id atualiza
- [ ] DELETE /goals/:id marca ABANDONED
- [ ] Tests passam

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
**Time**: 2h  

### Gate

- [ ] GET /payments lista pagamentos
- [ ] POST /payments/sync sincroniza Uber
- [ ] Paginação funciona
- [ ] Tests passam

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
**Time**: 2h  

### Gate

- [ ] Todas rotas públicas funcionam
- [ ] Todas rotas protegidas exigem JWT
- [ ] CORS whitelist funciona
- [ ] Rate limiting funciona
- [ ] Request logging JSON

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
**Time**: 1h  

### Gate

- [ ] Erro genérico pra cliente
- [ ] Request ID em response
- [ ] Stack trace só em dev
- [ ] Validação de inputs funciona
- [ ] Error codes padrão

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

### Gate

- [ ] Unit tests (handlers)
- [ ] Integration tests (end-to-end)
- [ ] Auth flow tested
- [ ] Rate limiting tested
- [ ] Coverage > 80%

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

