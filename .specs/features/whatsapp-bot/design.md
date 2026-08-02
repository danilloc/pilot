# Pilot — WhatsApp Bot Design

**Platform**: WhatsApp Business API  
**Orchestration**: n8n  
**Backend Integration**: Go API  
**Status**: 🟡 Design Ready  

---

## Bot Flows

### Flow 1: Motorista Inicia Jornada

```
Motorista: "/comecou_jornada"
  ↓
n8n receives message
  ↓
Call POST /api/drivers/:id/journeys/start
  ↓
Bot: "✅ Jornada iniciada às 08:50"
     "Agora é só sair ganhando! 🚗💨"
  ↓
Store journey_id in Redis
  ↓
Schedule 30min update message
```

### Flow 2: Notificação a Cada 30 Minutos

```
n8n Timer (every 30 min)
  ↓
Check if driver is in active journey
  ↓
Call GET /api/stats/today
  ↓
Call GET /api/goals/progress
  ↓
Send message:
"📊 Update (1h 30m)
 Ganho: R$ 87.50
 Meta: R$ 500.00 (17%)
 [Ver análise completa]" → PWA link
  ↓
Schedule next update (30 min)
```

### Flow 3: Meta Atingida

```
n8n Timer (every 5 min when close)
  ↓
GET /api/goals/progress
  ↓
If percent_complete >= 100
  ↓
Send message:
"🎉 PARABÉNS!
 Você bateu a meta do dia!
 Total: R$ 512.30
 [Ver análise detalhada]" → PWA link
  ↓
Update goal status to COMPLETED
  ↓
Stop sending updates
```

### Flow 4: Motorista Para Jornada

```
Motorista: "/parou_jornada"
  ↓
Call POST /api/drivers/:id/journeys/end
  ↓
GET final stats for day
  ↓
Send message:
"🏁 Jornada finalizada!
 Duração: 8h 30m
 Corridas: 15
 Ganho: R$ 512.30
 Meta: R$ 500.00 ✅
 Ganho/hora: R$ 60.27

 [Ver análise completa]" → PWA link
```

### Flow 5: Resumo Diário Automático

```
n8n Daily Timer (18:00)
  ↓
GET /api/stats/today
  ↓
GET /api/drivers/me
  ↓
If driver had activity today
  ↓
Send message:
"📈 Resumo de hoje
 Corridas: 15
 Ganho: R$ 512.30
 Meta: R$ 500.00 ✅
 Ganho/hora: R$ 60.27
 Melhor corrida: R$ 87.50

 [Ver analytics completo]" → PWA link
```

---

## Message Templates

### Initiated Journey
```
✅ Jornada iniciada às 08:50
Agora é só sair ganhando! 🚗💨
```

### 30-Min Update
```
📊 Update (1h 30m)
Ganho: R$ 87.50
Meta: R$ 500.00 (17%)
[Ver análise completa]
```

### 50% Goal Reached
```
⚠️ Halfway there!
Ganho: R$ 250.00
Meta: R$ 500.00 (50%)
```

### 80% Goal Reached
```
🔥 Quase lá!
Ganho: R$ 400.00
Meta: R$ 500.00 (80%)
```

### Goal Completed
```
🎉 PARABÉNS!
Você bateu a meta do dia!
Total: R$ 512.30
[Ver análise detalhada]
```

### Journey Ended
```
🏁 Jornada finalizada!
Duração: 8h 30m
Corridas: 15
Ganho: R$ 512.30
Meta: R$ 500.00 ✅
Ganho/hora: R$ 60.27

[Ver análise completa]
```

### Daily Summary
```
📈 Resumo de hoje
Corridas: 15
Ganho: R$ 512.30
Meta: R$ 500.00 ✅
Ganho/hora: R$ 60.27
Melhor corrida: R$ 87.50

[Ver analytics completo]
```

---

## Bot Commands

```
/comecou_jornada     - Inicia jornada
/parou_jornada       - Finaliza jornada
/ganho_hoje          - Ganho de hoje
/meta_hoje           - Status da meta
/help                - Ajuda
```

---

## n8n Workflows

### Workflow 1: Handle /comecou_jornada

```json
{
  "nodes": [
    {
      "name": "Webhook",
      "type": "wh",
      "webhookMethod": "POST",
      "route": "/comecou_jornada"
    },
    {
      "name": "Extract Driver ID",
      "type": "fn",
      "function": "extract_driver_from_whatsapp_id"
    },
    {
      "name": "Call Backend",
      "type": "http",
      "method": "POST",
      "url": "http://backend:8080/api/drivers/:id/journeys/start"
    },
    {
      "name": "Send Message",
      "type": "whatsapp",
      "message": "✅ Jornada iniciada às [time]"
    },
    {
      "name": "Save to Redis",
      "type": "redis",
      "key": "journey:{{ $node.Extract.data.driver_id }}",
      "value": "{{ $node['Call Backend'].data.journey_id }}"
    },
    {
      "name": "Schedule Updates",
      "type": "fn",
      "function": "schedule_30min_updates"
    }
  ]
}
```

### Workflow 2: 30-Min Update Timer

```json
{
  "nodes": [
    {
      "name": "Timer",
      "type": "timer",
      "interval": "30m",
      "trigger": "every_x_minutes"
    },
    {
      "name": "Get Active Journeys",
      "type": "redis",
      "command": "KEYS",
      "pattern": "journey:*"
    },
    {
      "name": "Loop Drivers",
      "type": "loop",
      "items": "{{ $node['Get Active Journeys'].data }}"
    },
    {
      "name": "Get Stats",
      "type": "http",
      "method": "GET",
      "url": "http://backend:8080/api/stats/today",
      "headers": { "Authorization": "Bearer {{ $secret.BACKEND_TOKEN }}" }
    },
    {
      "name": "Get Goal Progress",
      "type": "http",
      "method": "GET",
      "url": "http://backend:8080/api/goals/progress",
      "headers": { "Authorization": "Bearer {{ $secret.BACKEND_TOKEN }}" }
    },
    {
      "name": "Format Message",
      "type": "fn",
      "function": "format_update_message"
    },
    {
      "name": "Send WhatsApp",
      "type": "whatsapp",
      "message": "{{ $node['Format Message'].data }}"
    }
  ]
}
```

---

## Go Backend Endpoints (Bot-specific)

### Start Journey

```go
// POST /api/drivers/:id/journeys/start
type StartJourneyRequest struct {
    NotifyVia string `json:"notify_via"` // "whatsapp"
}

type JourneyResponse struct {
    JourneyID      string    `json:"journey_id"`
    StartedAt      time.Time `json:"started_at"`
    DriverID       int64     `json:"driver_id"`
}
```

### End Journey

```go
// POST /api/drivers/:id/journeys/end
type EndJourneyResponse struct {
    JourneyID       string    `json:"journey_id"`
    Duration        string    `json:"duration"`
    TotalTrips      int       `json:"total_trips"`
    TotalEarnings   float64   `json:"total_earnings"`
    GoalAchieved    bool      `json:"goal_achieved"`
    EndedAt         time.Time `json:"ended_at"`
}
```

### Get Journey Status

```go
// GET /api/drivers/:id/journeys/current
type JourneyStatus struct {
    JourneyID       string    `json:"journey_id"`
    Status          string    `json:"status"` // "ACTIVE", "PAUSED", "ENDED"
    StartedAt       time.Time `json:"started_at"`
    CurrentEarnings float64   `json:"current_earnings"`
    CurrentTrips    int       `json:"current_trips"`
    GoalProgress    float64   `json:"goal_progress"`
}
```

---

## Webhook Validation (Security)

```go
func ValidateWhatsAppWebhook(body []byte, signature string, secret string) bool {
    h := hmac.New(sha256.New, []byte(secret))
    h.Write(body)
    expected := base64.StdEncoding.EncodeToString(h.Sum(nil))
    
    return hmac.Equal([]byte(signature), []byte(expected))
}
```

---

## Rate Limiting (Bot-specific)

```
Max messages per driver per day: 20
Max journeys per day: 5
Message queue timeout: 30s
Cooldown between messages: 2 minutes
```

---

## Error Handling (Bot)

```json
{
  "error": "something_went_wrong",
  "message": "Desculpa, algo deu errado. Tente novamente em alguns minutos.",
  "user_message": "🚫 Erro ao processar pedido. Tente /help"
}
```

---

## Environment Variables (Bot)

```env
# WhatsApp
WHATSAPP_BUSINESS_API_KEY=<token>
WHATSAPP_PHONE_ID=<phone_id>
WHATSAPP_WEBHOOK_SECRET=<hmac_secret>

# n8n
N8N_URL=http://n8n:5678
N8N_API_KEY=<key>

# Redis (message queue)
REDIS_BOT_DB=1

# Backend
BACKEND_TOKEN=<service_account_token>
```

