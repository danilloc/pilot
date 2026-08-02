# Pilot Fleet Companion

Monorepo Go + Vue para um app de suporte a motoristas, com foco em análise de ganhos, jornada de trabalho e performance em tempo real.

## Abordagem

Este projeto usa a ideia de Spec-Driven Development (SDD): criar especificações claras antes de implementar e guiar o desenvolvimento com essas specs.

## Estrutura do repositório

- `backend/` - API REST em Go usando Gin
- `frontend/` - App web em Vue 3 + Vite
- `specs/` - especificações e requisitos iniciais

## Como usar

### Backend

1. Acesse `backend/`
2. Execute `go mod tidy`
3. Execute `go run .`

O backend roda em `http://localhost:8080`.

### Frontend

1. Acesse `frontend/`
2. Execute `npm install`
3. Execute `npm run dev`

O frontend roda em `http://localhost:5173`.

## SDD e desenvolvimento

Comece usando `specs/overview.md` para definir requisitos de produto, endpoints esperados e comportamento da UI. Depois, implemente backend e frontend com base nessas specs.
