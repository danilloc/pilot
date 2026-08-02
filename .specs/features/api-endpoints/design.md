# Pilot — API Specification

**Feature**: `api-endpoints`  
**Protocol**: REST + HTTP/2  
**Auth**: JWT + OAuth2  
**Docs**: OpenAPI 3.0  

---

## Base URL

- Development: `http://localhost:8080/api`
- Production: `https://api.pilot.com/api`

---

## 1. Authentication Endpoints

### POST /auth/uber-login
```json
{
  "code": "string",
  "state": "string"
}
→
{
  "token": "jwt...",
  "driver": { ... }
}
```

### GET /auth/me
```
Headers: Authorization: Bearer <token>
→
{
  "id": 1,
  "uuid": "...",
  "name": "João",
  "email": "joao@email.com",
  "rating": 4.8,
  ...
}
```

### POST /auth/logout
```
Headers: Authorization: Bearer <token>
→
{
  "success": true
}
```

---

## 2. Driver Endpoints

### GET /drivers/me
```
Headers: Authorization: Bearer <token>
→ { driver object }
```

### PUT /drivers/me
```json
{
  "phone": "+5511999999999",
  "default_daily_goal": 350.00
}
→ { updated driver }
```

### POST /drivers/sync-profile
```
Headers: Authorization: Bearer <token>
→
{
  "success": true,
  "synced_at": "2024-08-01T15:00:00Z"
}
```

---

## 3. Trips Endpoints

### GET /trips
```
Query:
  ?start_date=2024-08-01
  &end_date=2024-08-07
  &limit=50
  &offset=0
  &status=COMPLETED

→
{
  "data": [ trip objects ],
  "pagination": { limit, offset, total, page, total_pages }
}
```

### GET /trips/:id
```
→ { trip object with coordinates }
```

### POST /trips/sync
```json
{
  "limit": 50
}
→
{
  "success": true,
  "synced_count": 12,
  "synced_at": "2024-08-01T15:10:00Z"
}
```

---

## 4. Stats Endpoints

### GET /stats/today
```
→
{
  "date": "2024-08-01",
  "total_earned": 450.75,
  "total_trips": 8,
  "avg_fare": 56.34,
  "total_distance": 87.5,
  "earnings_per_hour": 112.69,
  "active_hours": 4.0
}
```

### GET /stats/week
```
→
{
  "period": "2024-07-26 to 2024-08-01",
  "days": [
    {
      "date": "2024-08-01",
      "earnings": 450.75,
      "trips": 8
    }
  ],
  "summary": { ... }
}
```

### GET /stats/month
```
→
{
  "month": "2024-08",
  "weeks": [ ... ],
  "summary": { ... }
}
```

---

## 5. Goals Endpoints

### POST /goals
```json
{
  "goal_date": "2024-08-01",
  "goal_amount": 350.00
}
→ { goal object }
```

### GET /goals
```
Query: ?date=2024-08-01
→ { goal object with progress }
```

### GET /goals/progress
```
→
{
  "goal_amount": 350.00,
  "actual_amount": 280.50,
  "percent_complete": 80.14,
  "remaining": 69.50
}
```

### PUT /goals/:id
```json
{
  "goal_amount": 400.00
}
→ { updated goal }
```

### DELETE /goals/:id
```
→
{
  "success": true,
  "status": "ABANDONED"
}
```

---

## 6. Payments Endpoints

### GET /payments
```
Query:
  ?start_date=2024-07-01
  &end_date=2024-08-01
  &limit=50

→
{
  "data": [ payment objects ],
  "pagination": { ... }
}
```

### POST /payments/sync
```
→
{
  "success": true,
  "synced_count": 3,
  "synced_at": "2024-08-01T16:00:00Z"
}
```

---

## 7. Health Endpoint

### GET /health
```
→
{
  "status": "healthy",
  "version": "1.0.0",
  "database": "connected",
  "cache": "connected"
}
```

---

## Error Handling

```json
{
  "code": "AUTH_001",
  "message": "Invalid token",
  "request_id": "req-123456",
  "timestamp": "2024-08-01T15:00:00Z"
}
```

**Status Codes:**
- 200: OK
- 201: Created
- 400: Bad Request
- 401: Unauthorized
- 403: Forbidden
- 404: Not Found
- 429: Rate Limited
- 500: Internal Error
- 503: Service Unavailable

---

## Rate Limiting

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 45
X-RateLimit-Reset: 1627839600
```

**Limits:**
- Auth: 5 req/min
- Trips: 30 req/min
- Stats: 60 req/min
- Goals: 30 req/min

---

## Pagination

```
GET /trips?limit=20&offset=0

Response:
{
  "data": [...],
  "pagination": {
    "limit": 20,
    "offset": 0,
    "total": 150,
    "page": 1,
    "total_pages": 8
  }
}
```

