# Pilot — Uber Drivers API Integration Design

**Feature**: `uber-integration`
**Status**: 🟢 Mapeamento de campos 100% confirmado — 🔴 acesso real ainda bloqueado (aprovação pendente da Uber)
**Criado após**: pesquisa na documentação oficial (developer.uber.com/docs/drivers)
**Atualizado após**: teste real de OAuth + tutorial curl completo oficial (02/08/2026)

---

## 🔴 Confirmado: acesso aos escopos ainda não aprovado pela Uber

Testamos o fluxo OAuth de verdade (tela de autorização da Uber) e o painel
do desenvolvedor confirma:

> "Seu aplicativo atualmente não tem acesso aos escopos de Código de
> Autorização. Entre em contato com seu representante de desenvolvimento
> de negócios da Uber ou com seu ponto de contato da Uber para solicitar
> acesso."

Isso vale para **todos os 3 escopos** (`partner.accounts`, `partner.trips`,
`partner.payments`) — não é um problema de configuração, é aprovação
pendente do lado da Uber, que exige contato com o time de parcerias/dev
relations deles. Prazo fora do nosso controle.

**Decisão**: seguir o desenvolvimento da fatia `api-endpoints` (T16, T17,
T19) com **dados mockados/fixtures** baseados nos formatos reais
documentados abaixo (JSON de exemplo confirmado na documentação oficial),
e trocar pela chamada real assim que a aprovação da Uber sair. O client
Go deve ser escrito de forma que trocar de mock para chamada real seja
só uma troca de implementação por trás da mesma interface (ex: uma
interface `UberClient` com um `MockUberClient` e um `RealUberClient`).

---

## ⚠️ Outras pendências (menores, não bloqueantes)

Estas não bloqueiam desenvolvimento/testes, mas precisam ser resolvidas
antes de considerar a integração "pronta para motoristas reais":

1. ~~Confirmar scopes ativos no app~~ — **resolvido acima**: nenhum scope
   de Authorization Code está ativo ainda.
2. **App está em modo "TEST APP" (sandbox)**: precisa solicitar "Criar
   aplicativo de produção" no painel — mas isso só faz sentido depois que
   os escopos forem aprovados. Dados de sandbox

   podem ser fictícios/limitados.
3. **Rate limits não documentados publicamente**: a doc oficial não lista
   um número exato. Implementar backoff exponencial (já previsto no
   design do backend) e monitorar respostas 429.

---

## Os 3 Endpoints Reais

### 1. `GET /v1/partners/me` — Perfil

Usado na T8 (login) e T9 (sync de perfil). Formato de resposta não detalhado
publicamente com o mesmo nível de profundidade dos outros dois, mas inclui:
rating, picture, status da conta, contagem de trips.

**Scope necessário**: `partner.accounts`

### 2. `GET /v1/partners/trips` — Corridas

Usado na T10/T16/T17 (sync e listagem de trips).

**Scope necessário**: `partner.trips`

**Query params**:
| Param | Tipo | Descrição |
|---|---|---|
| `offset` | int | Paginação, default 0 |
| `limit` | int | Default 10, **máximo 50** |
| `from_time` | int | Unix timestamp, início do período |
| `to_time` | int | Unix timestamp, fim do período |

**Resposta real (confirmada — tutorial oficial completo, 02/08/2026)**:
```json
{
  "count": 1200,
  "limit": 1,
  "offset": 0,
  "trips": [
    {
      "trip_id": "b5613b6a-fe74-4704-a637-50f8d51a8bb1",
      "driver_id": "8LvWuRAq2511gmr8EMkovekFNa2848ly...",
      "fare": 6.2,
      "currency_code": "USD",
      "distance": 0.37,
      "duration": 475,
      "status": "completed",
      "surge_multiplier": 1,
      "vehicle_id": "0082b54a-6a5e-4f6b-b999-b0649f286381",
      "pickup": { "timestamp": 1502843903 },
      "dropoff": { "timestamp": 1502844378 },
      "start_city": {
        "latitude": 38.3498,
        "longitude": -81.6326,
        "display_name": "Charleston, WV"
      },
      "status_changes": [
        { "status": "accepted", "timestamp": 1502843899 },
        { "status": "driver_arrived", "timestamp": 1502843900 },
        { "status": "trip_began", "timestamp": 1502843903 },
        { "status": "completed", "timestamp": 1502844378 }
      ]
    }
  ]
}
```

> `count` pode ser maior que `limit` — **é obrigatório paginar** com
> `offset` até coletar tudo. `status` pode ser: `accepted`, `arriving`,
> `in_progress`, `rider_canceled`, `driver_canceled`, `completed`.

### 3. `GET /v1/partners/payments` — Pagamentos

Usado na T19 (sync de pagamentos).

**Scope necessário**: `partner.payments`

**Nota importante da doc oficial**: pagamentos aparecem "near real-time".
Corridas canceladas **não geram entrada de payment**. Se o motorista
trabalha para um "fleet manager" (frota terceirizada), a resposta vem
**sempre vazia** — os pagamentos vão pro gestor da frota, não pro
motorista. Isso é um caso real que pode acontecer com nossos usuários e
precisa de tratamento (ex: avisar o motorista que não conseguimos
sincronizar pagamentos automaticamente nesse caso).

---

## Mapeamento: Resposta da Uber → Nossa Tabela `trips`

| Campo em `trips` (nosso schema) | Origem no JSON da Uber | Observação |
|---|---|---|
| `uber_trip_id` | ✅ **`trip_id`** — campo direto, confirmado no tutorial oficial completo | Resolvido |
| `started_at` | ✅ **`pickup.timestamp`** — campo direto, mais simples que derivar de `status_changes` | Resolvido |
| `ended_at` | `dropoff.timestamp` | direto |
| `distance_km` | `distance * 1.60934` | a Uber devolve em **milhas**, precisa converter |
| `fare_value` | `fare` | direto — é um valor único, **sem quebra** |
| `fare_base`, `fare_distance`, `fare_time`, `toll_charge`, `service_fee` | **não existem na resposta da Uber** | ficam `0`/`NULL` — schema já suporta isso, não precisa migração nova |
| `city` | `start_city.display_name` | só cidade de **origem**, não existe cidade de destino no payload |
| `start_latitude`/`start_longitude` | `start_city.latitude`/`start_city.longitude` | direto |
| `end_latitude`/`end_longitude` | **não disponível no payload documentado** | ficam `NULL` |
| `status` | ✅ **`status`** — campo direto (`completed`, `driver_canceled`, `rider_canceled`, etc), não precisa mais derivar de `status_changes` | mapear pros nossos valores (`COMPLETED`, `CANCELLED`) |
| (extra, não mapeado ainda) | `duration` (segundos) | pode usar pra **validar** que `duration_minutes` calculado bate com o que a Uber informa — não precisa persistir, é redundante com `started_at`/`ended_at` |
| (extra, não mapeado ainda) | `currency_code` | mapear pra `trips.currency` (já existe no schema) |

### ✅ Ponto que estava em aberto — agora resolvido

O tutorial oficial completo (`docs/drivers/tutorials/api/curl`) mostra o
JSON de resposta **sem truncar**, confirmando `trip_id` como campo direto
e único. Idempotência garantida usando `uber_trip_id = trip_id` — não
precisa mais do fallback de hash.

---

## Mapeamento: Resposta da Uber → Nossa Tabela `payment_records`

**Formato real confirmado (tutorial oficial completo, 02/08/2026)**:

```json
{
  "count": 1200,
  "limit": 1,
  "offset": 0,
  "payments": [
    {
      "payment_id": "5cb8304c-f3f0-4a46-b6e3-b55e020750d7",
      "trip_id": "5cb8304c-f3f0-4a46-b6e3-b55e020750d7",
      "category": "fare",
      "event_time": 1502842757,
      "cash_collected": 0,
      "amount": 3.12,
      "currency_code": "USD",
      "driver_id": "8LvWuRAq2511gmr8EMkovekFNa2848ly...",
      "breakdown": {
        "other": 4.16,
        "service_fee": -1.04
      },
      "rider_fees": {},
      "partner_id": "8LvWuRAq2511gmr8EMkovekFNa2848ly..."
    }
  ]
}
```

| Campo em `payment_records` (nosso schema) | Origem no JSON da Uber | Observação |
|---|---|---|
| `uber_payment_id` | ✅ **`payment_id`** — string UUID direta, confirmado | Resolvido |
| `amount` | `amount` | direto |
| `payment_date` | `event_time` (Unix timestamp) | converter pra `DATE` |
| `tolls` | `breakdown.toll` (quando presente — não aparece em todo pagamento) | direto |
| `taxes` | **ainda não confirmado** — `breakdown` tem `other`/`service_fee`/`toll`, nenhum chamado literalmente "tax" | mapear `service_fee` como aproximação, revisar quando tiver acesso real |
| `currency` | `currency_code` | direto |
| (referência) | `trip_id` | usar pra vincular `payment_records` → `trips` via `trips.uber_trip_id = payments.trip_id` |

**Caso especial documentado**: se o motorista trabalha para um "fleet
manager", a resposta de payments vem **sempre vazia** (ver nota acima) —
tratar como cenário válido, não como erro.


---

## Fluxo de Sincronização (proposto)

```
Motorista clica "Sincronizar" (ou timer automático)
        ↓
POST /api/trips/sync (nosso backend)
        ↓
Backend chama GET /v1/partners/trips?from_time=<ultimo_sync>&limit=50
        ↓
Para cada trip retornada:
  - Deriva started_at/ended_at do status_changes
  - Converte distance milhas→km
  - Monta chave de idempotência
  - INSERT ... ON DUPLICATE KEY UPDATE (idempotente)
        ↓
Se count > limit: repete com offset += 50, até coletar tudo
        ↓
Atualiza drivers.last_synced_at
        ↓
Retorna { success: true, synced_count: N, synced_at: ... }
```

## Idempotência

Como já vimos na fatia `core-backend` (bug #5 documentado pelo agent —
"ID do driver não populava após upsert em conflito"), o padrão de
`ON DUPLICATE KEY UPDATE` do MySQL já é usado no projeto. Pra `trips`,
precisamos de uma constraint `UNIQUE` na coluna de idempotência (seja
`uber_trip_id` de verdade ou o hash calculado) — isso já existe no schema
atual (`uber_trip_id VARCHAR(100) UNIQUE NOT NULL`), só precisa confirmar
que sempre teremos um valor válido pra popular esse campo.

## Rate Limiting e Retry

Já coberto no design do backend (T1/T7):
- Timeout de 30s por chamada
- Retry com backoff exponencial em erro 5xx ou timeout
- Respeitar 429 (Too Many Requests) com espera antes de tentar de novo
- Nunca sincronizar mais de 1x por minuto por motorista (evita abuso
  mesmo sem limite documentado publicamente)

---

## Antes de mandar a fatia `api-endpoints` pro agent

- [x] Confirmado: `partner.accounts`, `partner.trips` e `partner.payments`
      **não estão liberados** — app precisa de aprovação da Uber
      (contato com dev relations/parcerias, processo deles)
- [x] Formato real de `trips` **100% confirmado** via tutorial oficial
      completo — incluindo `trip_id`, `pickup.timestamp`, `status`,
      `duration`, `currency_code`
- [x] Formato real de `payments` **100% confirmado** via tutorial oficial
      completo — incluindo `payment_id` como string, `trip_id` de
      vinculação
- [x] `uber_trip_id` = `trip_id` — idempotência garantida, sem precisar de
      fallback de hash
- [x] `UBER_CLIENT_ID`/`UBER_CLIENT_SECRET`/`UBER_REDIRECT_URI` conferidos
      no `.env` (redirect URI confirmado ✅)
- [ ] **Ação de negócio pendente, fora do dev**: Danillo buscar contato de
      dev relations da Uber pra solicitar aprovação dos 3 escopos
- [ ] Testar OAuth com o domínio correto `auth.uber.com` (em vez de
      `login.uber.com`, usado nos testes anteriores) — mesmo que a
      aprovação ainda esteja pendente, vale confirmar se o domínio
      influenciava o erro observado

## Estratégia de implementação (decidida em 02/08/2026)

A fatia `api-endpoints` **segue normalmente**, com o client Uber implementado
atrás de uma interface (`UberClient`), com duas implementações:

- `MockUberClient` — retorna fixtures baseados nos formatos documentados
  aqui (inclusive o JSON real de `payments`). **Usado agora.**
- `RealUberClient` — chama a API de verdade. **Implementado, mas não
  testável até a aprovação sair.** Fica pronto pra troca de configuração
  (ex: uma flag `UBER_USE_MOCK=true` no `.env`) assim que o acesso for
  liberado.

Isso permite terminar o desenvolvimento e os testes da fatia inteira sem
depender do cronograma da Uber, e trocar pra dados reais só mudando uma
variável de ambiente quando chegar a hora.
