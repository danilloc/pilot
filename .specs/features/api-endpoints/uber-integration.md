# Pilot — Uber Drivers API Integration Design

**Feature**: `uber-integration`
**Status**: 🟡 Precisa de validação prática antes da implementação final
**Criado após**: pesquisa na documentação oficial (developer.uber.com/docs/drivers)

---

## ⚠️ Pendências que bloqueiam a implementação 100% de produção

Estas não bloqueiam desenvolvimento/testes, mas precisam ser resolvidas
antes de considerar a integração "pronta para motoristas reais":

1. **Confirmar scopes ativos no app**: não foi possível confirmar no painel
   do developer.uber.com se `partner.trips` e `partner.payments` estão
   liberados (só `partner.accounts`/perfil é garantido). **Ação**: fazer uma
   chamada de teste real assim que a T16/T17 estiverem implementadas e
   observar se retorna 200 com dados ou 403/401 de permissão.
2. **App está em modo "TEST APP" (sandbox)**: precisa solicitar "Create Prod
   App" no painel antes de motoristas reais poderem logar. Dados de sandbox
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

**Resposta real**:
```json
{
  "count": 1200,
  "limit": 50,
  "trips": [
    {
      "fare": 6.2,
      "distance": 0.37,
      "vehicle_id": "0082b54a-6a5e-4f6b-b999-b0649f286381",
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
> `offset` até coletar tudo.

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
| `uber_trip_id` | **não confirmado no payload público** — precisa validar no teste real se existe um campo tipo `trip_id`/`request_id`, ou se é preciso gerar hash de `dropoff.timestamp + vehicle_id` como fallback de idempotência | 🔴 Ponto de atenção |
| `started_at` | `status_changes[].timestamp` onde `status == "trip_began"` | derivado, não é campo direto |
| `ended_at` | `dropoff.timestamp` | direto |
| `distance_km` | `distance * 1.60934` | a Uber devolve em **milhas**, precisa converter |
| `fare_value` | `fare` | direto — é um valor único, **sem quebra** |
| `fare_base`, `fare_distance`, `fare_time`, `toll_charge`, `service_fee` | **não existem na resposta da Uber** | ficam `0`/`NULL` — schema já suporta isso, não precisa migração nova |
| `city` | `start_city.display_name` | só cidade de **origem**, não existe cidade de destino no payload |
| `start_latitude`/`start_longitude` | `start_city.latitude`/`start_city.longitude` | direto |
| `end_latitude`/`end_longitude` | **não disponível no payload documentado** | ficam `NULL` |
| `status` | derivar do último item de `status_changes` (`completed`, `driver_canceled`, `rider_canceled`) | mapear pros nossos valores (`COMPLETED`, `CANCELLED`) |

### 🔴 Ponto crítico a resolver na implementação (T16/T17)

O payload de exemplo da documentação **não mostra claramente um ID único
de trip**. Antes de implementar o sync de verdade, é obrigatório fazer uma
chamada real (mesmo em sandbox) e inspecionar o JSON completo pra confirmar
o nome exato do campo de ID — sem isso, não dá pra garantir idempotência
(evitar duplicar a mesma corrida se o sync rodar duas vezes).

**Se não existir campo de ID único**: usar como chave de idempotência um
hash de `vehicle_id + dropoff.timestamp` (uma corrida não devia repetir
esses dois valores juntos).

---

## Mapeamento: Resposta da Uber → Nossa Tabela `payment_records`

Menos detalhado publicamente que trips. A doc confirma que a resposta
inclui: moeda, valor, horário do pagamento, e tipo (`device_subscription`
aparece como exemplo de tipo periódico). O mapeamento exato campo-a-campo
só pode ser confirmado com uma chamada real — **não adivinhar campos aqui**.

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

Checklist do que precisa estar resolvido:

- [ ] Confirmado se `partner.trips` e `partner.payments` estão liberados
      (via teste real ou painel)
- [ ] Feita 1 chamada de teste em sandbox pra `GET /partners/trips` e
      inspecionado o JSON completo — confirmar nome do campo de ID único
- [ ] Feita 1 chamada de teste pra `GET /partners/payments` — confirmar
      estrutura completa dos campos
- [ ] Decidido o fallback de idempotência caso não exista ID único
- [ ] `UBER_CLIENT_ID`/`UBER_CLIENT_SECRET`/`UBER_REDIRECT_URI` conferidos
      no `.env` (redirect URI já confirmado ✅)

Se algum item não puder ser resolvido antes (ex: scope de trips ainda não
aprovado pela Uber), a fatia `api-endpoints` pode seguir implementando os
handlers com dados mockados/fixtures baseados neste documento, e o sync
real fica marcado como pendente até o acesso ser confirmado.
