# Pilot — Database Tasks

**Feature**: `database-schema`  
**Branch**: `features/database-schema`  
**Assignee**: eliezersilva-spec (DB Lead)  

---

## Task List

| T# | Issue | Task | Deps | Time |
|----|-------|------|------|------|
| T11 | #61 | ✅ Setup migrate tool | - | 1h |
| T12 | #62 | ✅ Create migration 001-006 | T11 | 2h |
| T13 | #63 | Create indexes & constraints | T12 | 1h |
| T14 | #64 | Test queries & optimization | T13 | 2h |

---

## T11: Setup Migrate Tool

**Issue**: #61  
**Status**: ✅ Done  
**Time**: 1h  

### Gate

- [x] golang-migrate instalado
- [x] migrations directory criado
- [x] migrate up/down funciona
- [x] Database criado e conecta

### Commit

```
feat(database): setup golang-migrate for schema management

- Install golang-migrate CLI
- Create migrations directory structure
- Add migration runner in Go
- Test up/down migrations

Closes #61
```

---

## T12: Create Migrations

**Issue**: #62  
**Deps**: T11  
**Status**: ✅ Done  
**Time**: 2h  

### Gate

- [x] 001_create_drivers.sql executável
- [x] 002_create_trips.sql executável
- [x] 003_create_daily_goals.sql executável
- [x] 004_create_payment_records.sql executável
- [x] 005_create_work_sessions.sql executável
- [x] 006_create_stats_cache.sql executável
- [x] Rollback funciona em cada uma
- [x] Foreign keys criadas
- [x] Constraints validam corretamente

### Commit

```
feat(database): add schema migrations

- Add 001_create_drivers migration
- Add 002_create_trips migration
- Add 003_create_daily_goals migration
- Add 004_create_payment_records migration
- Add 005_create_work_sessions migration
- Add 006_create_stats_cache migration
- Test all migrations up and down

Closes #62
```

---

## T13: Create Indexes & Constraints

**Issue**: #63  
**Deps**: T12  
**Status**: 🟡 Ready  
**Time**: 1h  

### Gate

- [ ] idx_driver_ended criado em trips
- [ ] idx_status criado em trips
- [ ] unique_driver_date criado em daily_goals
- [ ] Foreign keys funcionam
- [ ] Constraints CHECK validam
- [ ] FULLTEXT search em city funciona
- [ ] Index stats mostram uso

### Commit

```
feat(database): add indexes and constraints

- Create compound index idx_driver_ended
- Create FULLTEXT index on city
- Add UNIQUE constraint on daily goals
- Test index usage with EXPLAIN
- Verify foreign key constraints

Closes #63
```

---

## T14: Test Queries & Optimization

**Issue**: #64  
**Deps**: T13  
**Status**: ⏳ Blocked  
**Time**: 2h  

### Gate

- [ ] Daily earnings query executável
- [ ] Weekly stats query executável
- [ ] Goal progress query executável
- [ ] Queries < 100ms com 1M rows
- [ ] Index explain plan otimizado
- [ ] Slow query log vazio

### Commit

```
feat(database): add query tests and optimization

- Test daily earnings query performance
- Test weekly stats aggregation
- Test goal progress calculation
- Add query performance benchmarks
- Verify indexes used in EXPLAIN plans

Closes #64
```

