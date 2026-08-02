# Pilot — API Tasks

**Feature**: `api-endpoints`  
**Branch**: `features/api-endpoints`  
**Assignee**: danillobrito-sr  

---

## Task List

| T# | Issue | Task | Deps | Time |
|----|-------|------|------|------|
| T15 | #65 | Auth handlers (login/me/logout) | T8 | 2h |
| T16 | #66 | Driver handlers (profile/sync) | T9 | 2h |
| T17 | #67 | Trips handlers (list/get/sync) | T10 | 2h |
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
**Time**: 2h  

### Gate

- [ ] POST /auth/uber-login funciona
- [ ] Token encrypted antes de armazenar
- [ ] JWT retornado ao cliente
- [ ] GET /auth/me retorna motorista
- [ ] POST /auth/logout blacklist token
- [ ] Tests passam (valid/invalid codes)

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
**Time**: 2h  

### Gate

- [ ] GET /drivers/me funciona
- [ ] PUT /drivers/me atualiza
- [ ] POST /drivers/sync-profile sincroniza Uber
- [ ] Validação de inputs
- [ ] Tests passam

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
**Time**: 2h  

### Gate

- [ ] GET /trips lista com paginação
- [ ] GET /trips/:id retorna trip
- [ ] POST /trips/sync sincroniza Uber
- [ ] Filtros funcionam
- [ ] Tests passam

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

