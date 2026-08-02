# Pilot — WhatsApp Bot Tasks

**Feature**: `whatsapp-bot`  
**Branch**: `features/whatsapp-bot`  
**Assignee**: danillobrito-sr  

---

## Task List

| T# | Issue | Task | Deps | Time |
|----|-------|------|------|------|
| T34 | #84 | Setup n8n environment | - | 1h |
| T35 | #85 | Setup WhatsApp Business API | - | 1h |
| T36 | #86 | Backend: journey endpoints | T9 | 2h |
| T37 | #87 | n8n workflow: /comecou_jornada | T34, T35 | 1h |
| T38 | #88 | n8n workflow: 30min updates | T34, T35 | 2h |
| T39 | #89 | n8n workflow: goal alerts | T34, T35 | 1h |
| T40 | #90 | n8n workflow: /parou_jornada | T34, T35 | 1h |
| T41 | #91 | Message templates & i18n | T37-T40 | 1h |
| T42 | #92 | Bot tests (message validation) | T37-T40 | 2h |

---

## T34: Setup n8n Environment

**Issue**: #84  
**Status**: 🟡 Ready  
**Time**: 1h  

### Gate

- [ ] n8n instalado (Docker ou local)
- [ ] Database conectada
- [ ] Admin UI acessível
- [ ] CLI funciona

### Commit

```
feat(bot): setup n8n orchestration platform

- Install n8n via Docker
- Configure database connection
- Setup admin credentials
- Test CLI commands

Closes #84
```

---

## T35: Setup WhatsApp Business API

**Issue**: #85  
**Status**: 🟡 Ready  
**Time**: 1h  

### Gate

- [ ] WhatsApp Business Account criado
- [ ] Phone ID obtido
- [ ] Access token gerado
- [ ] Webhook URL configurada
- [ ] Test message enviado

### Commit

```
feat(bot): configure WhatsApp Business API

- Create Business Account
- Generate access token
- Configure webhook
- Setup phone number
- Test message sending

Closes #85
```

---

## T36: Backend Journey Endpoints

**Issue**: #86  
**Deps**: T9  
**Time**: 2h  

### Gate

- [ ] POST /journeys/start cria jornada
- [ ] POST /journeys/end finaliza
- [ ] GET /journeys/current retorna status
- [ ] Redis armazena journey_id
- [ ] Tests passam

### Commit

```
feat(backend): add journey management endpoints

- Add POST /drivers/:id/journeys/start
- Add POST /drivers/:id/journeys/end
- Add GET /drivers/:id/journeys/current
- Implement journey state in Redis
- Add journey tests

Closes #86
```

---

## T37: Workflow /comecou_jornada

**Issue**: #87  
**Deps**: T34, T35  
**Time**: 1h  

### Gate

- [ ] Webhook recebe mensagem
- [ ] Backend chamado
- [ ] Resposta enviada
- [ ] Journey salvo em Redis
- [ ] Timer iniciado

### Commit

```
feat(bot): add /comecou_jornada workflow

- Create n8n workflow for start journey
- Configure webhook
- Call backend endpoint
- Send confirmation message
- Schedule 30min updates

Closes #87
```

---

## T38: Workflow 30-Min Updates

**Issue**: #88  
**Deps**: T34, T35  
**Time**: 2h  

### Gate

- [ ] Timer executa a cada 30min
- [ ] Obtém stats do backend
- [ ] Obtém goal progress
- [ ] Mensagem formatada
- [ ] WhatsApp envia

### Commit

```
feat(bot): add 30-minute update workflow

- Create timer trigger (30min)
- Get active journeys from Redis
- Call stats and goals endpoints
- Format update message
- Send WhatsApp message

Closes #88
```

---

## T39: Workflow Goal Alerts

**Issue**: #89  
**Deps**: T34, T35  
**Time**: 1h  

### Gate

- [ ] Timer verifica meta a cada 5min
- [ ] Alerta em 50%, 80%, 100%
- [ ] Mensagens customizadas
- [ ] Stops after 100%

### Commit

```
feat(bot): add goal milestone alerts

- Create timer trigger (5min)
- Check goal progress
- Send 50% alert
- Send 80% alert
- Send 100% completion alert

Closes #89
```

---

## T40: Workflow /parou_jornada

**Issue**: #90  
**Deps**: T34, T35  
**Time**: 1h  

### Gate

- [ ] Webhook recebe mensagem
- [ ] Backend finaliza jornada
- [ ] Resumo calculado
- [ ] Mensagem enviada
- [ ] Redis limpado

### Commit

```
feat(bot): add /parou_jornada workflow

- Create webhook for stop journey
- Call backend endpoint
- Get final stats
- Send summary message
- Cleanup Redis

Closes #90
```

---

## T41: Message Templates & i18n

**Issue**: #91  
**Deps**: T37-T40  
**Time**: 1h  

### Gate

- [ ] Templates criados
- [ ] Variáveis dinâmicas funcionam
- [ ] PT-BR configurado
- [ ] Testes de renderização

### Commit

```
feat(bot): add message templates and i18n

- Create message template system
- Add Portuguese templates
- Implement variable substitution
- Test template rendering

Closes #91
```

---

## T42: Bot Tests

**Issue**: #92  
**Time**: 2h  

### Gate

- [ ] Message parsing tests
- [ ] Webhook validation tests
- [ ] Journey flow tests
- [ ] Error handling tests
- [ ] Coverage > 75%

### Commit

```
feat(bot): add comprehensive bot tests

- Add message parsing tests
- Add webhook signature validation tests
- Add end-to-end journey tests
- Add error scenario tests
- Reach 75%+ coverage

Closes #92
```

