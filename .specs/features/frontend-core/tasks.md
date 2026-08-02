# Pilot — Frontend Tasks

**Feature**: `frontend-ui`  
**Branch**: `features/frontend-ui`  
**Assignee**: danilloc  

---

## Task List

| T# | Issue | Task | Deps | Time |
|----|-------|------|------|------|
| T24 | #74 | Setup Nuxt project | - | 2h |
| T25 | #75 | Create composables (auth/stats/trips) | T24 | 2h |
| T26 | #76 | Create Pinia stores | T24 | 1h |
| T27 | #77 | Build dashboard page | T25 | 2h |
| T28 | #78 | Build analytics page | T25 | 2h |
| T29 | #79 | Build trips page | T25 | 2h |
| T30 | #80 | Build profile page | T25 | 1h |
| T31 | #81 | Build common components | T26 | 2h |
| T32 | #82 | PWA setup (offline/manifest) | T24 | 1h |
| T33 | #83 | Frontend tests | All | 2h |

---

## T24: Setup Nuxt Project

**Issue**: #74  
**Status**: 🟡 Ready  
**Time**: 2h  

### Gate

- [ ] Nuxt 3 projeto criado
- [ ] Tailwind CSS configurado
- [ ] TypeScript pronto
- [ ] .env.example criado
- [ ] npm run dev funciona
- [ ] npm run build compila

### Commit

```
feat(frontend): initialize Nuxt 3 project with Tailwind

- Create Nuxt 3 project structure
- Configure Tailwind CSS
- Setup TypeScript
- Configure runtime config
- Add .env.example

Closes #74
```

---

## T25: Create Composables

**Issue**: #75  
**Deps**: T24  
**Time**: 2h  

### Gate

- [ ] useAuth composable funciona
- [ ] useStats composable funciona
- [ ] useTrips composable funciona
- [ ] useGoals composable funciona
- [ ] API integration funciona

### Commit

```
feat(frontend): add composables for logic reuse

- Add useAuth composable
- Add useStats composable
- Add useTrips composable
- Add useGoals composable
- Test composable functionality

Closes #75
```

---

## T26: Create Pinia Stores

**Issue**: #76  
**Deps**: T24  
**Time**: 1h  

### Gate

- [ ] Auth store funciona
- [ ] Driver store funciona
- [ ] Trips store funciona
- [ ] Stats store funciona
- [ ] UI store funciona

### Commit

```
feat(frontend): add Pinia stores for state management

- Create auth store
- Create driver store
- Create trips store
- Create stats store
- Create ui store

Closes #76
```

---

## T27: Dashboard Page

**Issue**: #77  
**Deps**: T25  
**Time**: 2h  

### Gate

- [ ] EarningsCard mostra ganho do dia
- [ ] GoalProgress mostra meta
- [ ] RecentTrips lista últimas 5
- [ ] Stats grid exibe métricas
- [ ] Refresh a cada 30s

### Commit

```
feat(frontend): build dashboard page

- Create Dashboard page
- Add EarningsCard component
- Add GoalProgress component
- Add RecentTrips component
- Add auto-refresh logic

Closes #77
```

---

## T28: Analytics Page

**Issue**: #78  
**Deps**: T25  
**Time**: 2h  

### Gate

- [ ] Abas week/month funcionam
- [ ] Gráficos mostram dados
- [ ] Filtros funcionam
- [ ] Performance OK (< 1s)

### Commit

```
feat(frontend): build analytics page

- Create Analytics page
- Add EarningsChart component
- Add weekly/monthly tabs
- Add performance charts

Closes #78
```

---

## T29: Trips Page

**Issue**: #79  
**Deps**: T25  
**Time**: 2h  

### Gate

- [ ] Tabela lista trips
- [ ] Paginação funciona
- [ ] Filtros por data funcionam
- [ ] Clique abre detalhes

### Commit

```
feat(frontend): build trips history page

- Create Trips page
- Add TripsTable component
- Add pagination
- Add date filtering
- Add trip details modal

Closes #79
```

---

## T30: Profile Page

**Issue**: #80  
**Deps**: T25  
**Time**: 1h  

### Gate

- [ ] Mostra dados pessoais
- [ ] Edição funciona
- [ ] Logout funciona
- [ ] Stats do motorista

### Commit

```
feat(frontend): build profile page

- Create Profile page
- Add ProfileForm component
- Add logout button
- Add profile stats

Closes #80
```

---

## T31: Common Components

**Issue**: #81  
**Deps**: T26  
**Time**: 2h  

### Gate

- [ ] Header com logo/logout
- [ ] Sidebar com navegação
- [ ] LoadingSpinner funciona
- [ ] ErrorAlert mostra erros
- [ ] Footer presente

### Commit

```
feat(frontend): add common components

- Create Header component
- Create Sidebar component
- Create LoadingSpinner component
- Create ErrorAlert component
- Add component tests

Closes #81
```

---

## T32: PWA Setup

**Issue**: #82  
**Deps**: T24  
**Time**: 1h  

### Gate

- [ ] Service Worker instalado
- [ ] Offline funcionando
- [ ] Manifest.json criado
- [ ] App instalável

### Commit

```
feat(frontend): setup PWA capabilities

- Add service worker
- Create manifest.json
- Enable offline mode
- Add install prompt

Closes #82
```

---

## T33: Frontend Tests

**Issue**: #83  
**Time**: 2h  

### Gate

- [ ] Unit tests componentes
- [ ] Composable tests
- [ ] Store tests
- [ ] Coverage > 70%

### Commit

```
feat(frontend): add component and integration tests

- Add Vitest config
- Add component tests
- Add composable tests
- Add store tests
- Reach 70%+ coverage

Closes #83
```

