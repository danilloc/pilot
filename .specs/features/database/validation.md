# Database Feature Validation

**Date**: 2026-08-02
**Spec**: No `spec.md` for this feature — AC source of truth is `.specs/features/database/design.md` ("Migrations", "Índices Críticos", "Queries Críticas") and the Gate checklists in `.specs/features/database/tasks.md` (T11–T14)
**Diff range**: `develop..fatia/database` (commits `b5ac4e2`, `1f566af`, `d694ce9`, `f3a94f0`; a fifth commit `1a2f8dd` on the branch is unrelated docs and out of scope)
**Verifier**: independent sub-agent (author ≠ verifier)

---

## Task Completion

| Task | Status  | Notes |
| ---- | ------- | ----- |
| T11  | ✅ Done | golang-migrate wired into `cmd/migrate/main.go`, confirmed working live (`version=6 dirty=false`) |
| T12  | ⚠️ Partial | All 6 up/down migration pairs exist and up-migrations are applied and schema-verified; **no automated test exercises `migrate down`** (evidence-or-zero gap, see below) |
| T13  | ⚠️ Partial | Indexes/constraints tests exist and pass, but coverage is uneven — 3 of 6 tables (`payment_records`, `work_sessions`, `stats_cache`) have zero test coverage, and the design-doc "critical index" `idx_driver_date` (payment_records) is never asserted |
| T14  | ✅ Done | All 3 critical queries tested for correctness + <100ms threshold against 1,025,800-row `trips`; slow-query-log assertion present |

---

## Spec-Anchored Acceptance Criteria

### T11: Setup Migrate Tool

| Gate item | Expected outcome | `file:line` + evidence | Result |
| --- | --- | --- | --- |
| golang-migrate instalado | `golang-migrate/migrate/v4` a direct dependency | `pilot-backend/go.mod:9` — `github.com/golang-migrate/migrate/v4 v4.19.1` | ✅ PASS |
| migrations directory criado | `migrations/` dir with SQL files under source control | `pilot-backend/migrations/.gitkeep` + 12 `.sql` files present | ✅ PASS |
| migrate up/down funciona | `up`/`down`/`version` subcommands work against real DB | `pilot-backend/cmd/migrate/main.go:50-74` (`run()`) — manually verified: `go run ./cmd/migrate version` → `migrate: version=6 dirty=false` | ✅ PASS |
| Database criado e conecta | DSN built from `internal/config`, connects successfully | `pilot-backend/cmd/migrate/main.go:31-41` — same manual run above proves a live connection | ✅ PASS |

### T12: Create Migrations

| Gate item | Expected outcome | `file:line` + evidence | Result |
| --- | --- | --- | --- |
| 000001–000006 `*.up.sql` executável | Each table created exactly per design.md's "Migrations" section | `pilot-backend/migrations/000001_create_drivers.up.sql` … `000006_create_stats_cache.up.sql` — byte-for-byte diffed against design.md §Migrations, identical (ignoring whitespace); confirmed live: `go run ./cmd/migrate version` → `version=6 dirty=false` | ✅ PASS |
| Rollback funciona em cada uma | `migrate down` succeeds for each step | `pilot-backend/migrations/00000N_*.down.sql` — all 6 are single `DROP TABLE IF EXISTS <table>;` statements, syntactically correct and FK-order-safe if run in sequence (6→1) | ⚠️ Evidence gap — no test in `migrations/*_test.go` or CI invokes `migrate down`; only manual/visual verification possible. Down SQL is trivial enough that risk is low, but per evidence-or-zero this checkbox has no executed-evidence citation. **Not independently re-run by this Verifier** because doing so would drop the shared 1,025,800-row `trips` table seeded for T14, which the task brief explicitly said not to delete. |
| Foreign keys criadas | All 5 FK relationships exist (`trips→drivers`, `daily_goals→drivers`, `payment_records→drivers`, `work_sessions→drivers`, `work_sessions→daily_goals`) | `pilot-backend/migrations/000002_create_trips.up.sql:26`, `000003_create_daily_goals.up.sql:12`, `000004_create_payment_records.up.sql:16`, `000005_create_work_sessions.up.sql:12-13` (DDL present for all 5); **only `trips→drivers` is behaviorally tested** — `pilot-backend/migrations/schema_test.go:110-148` | ⚠️ Partial — DDL present for all 5 FKs (✅), but only 1 of 5 has a passing behavioral test. `daily_goals→drivers`, `payment_records→drivers`, `work_sessions→drivers`, and `work_sessions→daily_goals` (`ON DELETE SET NULL`, the only non-CASCADE FK in the schema) are completely untested. |
| Constraints CHECK validam corretamente | Both CHECK constraints (`check_rating`, `check_goal_positive`) reject bad data | `schema_test.go:150-163` — `TestCheckConstraint_RejectsInvalidRating` (rating=6.0 → MySQL error containing `check_rating`); `schema_test.go:165-179` — `TestCheckConstraint_RejectsNonPositiveGoalAmount` (goal_amount=0 → error containing `check_goal_positive`) | ✅ PASS — both of the schema's 2 CHECK constraints are covered |

### T13: Create Indexes & Constraints (DDL from T12, tests added here — documented scoping decision, not a gap)

| Gate item | Expected outcome | `file:line` + evidence | Result |
| --- | --- | --- | --- |
| idx_driver_ended criado em trips | Index exists | `schema_test.go:70-71` (`TestIndexes_Exist/trips.idx_driver_ended`) — pass | ✅ PASS |
| idx_status criado em trips | Index exists | `schema_test.go:72` (`TestIndexes_Exist/trips.idx_status`) — pass | ✅ PASS |
| unique_driver_date criado em daily_goals | Unique constraint exists and rejects duplicates | `schema_test.go:73` (existence) + `schema_test.go:181-194` (`TestUniqueDriverDate_RejectsDuplicateGoal`, behavioral) — both pass | ✅ PASS |
| Foreign keys funcionam | All FKs enforce referential integrity | Same partial coverage as T12's FK row above | ⚠️ Partial (see T12 FK row) |
| Constraints CHECK validam | Both CHECK constraints work | Same as T12 CHECK row | ✅ PASS |
| FULLTEXT search em city funciona | `MATCH...AGAINST` returns expected row | `schema_test.go:196-218` (`TestFulltextSearch_MatchesCity`) — inserts city='São Paulo', searches 'Paulo', asserts count==1 | ✅ PASS |
| Index stats mostram uso | EXPLAIN plan shows index selected | `schema_test.go:220-233` (`TestExplain_UsesDriverEndedIndex`) — pass, **but see Discrimination Sensor mutation 3**: the assertion (`strings.Contains(plan, "idx_driver_ended")`) matches on `possible_keys` too, not just the chosen `key`, so it does not reliably prove the index was actually *used* | ⚠️ Spec-precision gap — test passes but the assertion is weaker than the gate item's intent ("mostram uso" = shows *usage*, not just consideration) |

**Índices Críticos cross-check against design.md**: of the 5 indexes design.md flags as critical (`idx_driver_ended`, `idx_driver_id` on trips, `idx_driver_date` on payment_records, `unique_driver_date`, `idx_status`), only 3 (`idx_driver_ended`, `idx_status`, `unique_driver_date`) have any test asserting their existence. `idx_driver_id` (trips) and `idx_driver_date` (payment_records) are never asserted by name anywhere in the test suite.

### T14: Test Queries & Optimization

| Gate item | Expected outcome | `file:line` + evidence | Result |
| --- | --- | --- | --- |
| Daily earnings query executável, <100ms @ 1M rows | Query from design.md §Queries Críticas runs and returns >0 trips in <100ms | `query_bench_test.go:40-64` (`t.Run("daily_earnings", …)`) — re-run by Verifier: 0.06s (60ms) elapsed, `totalTrips > 0` | ✅ PASS |
| Weekly stats query executável, <100ms | Same | `query_bench_test.go:66-103` — re-run: 0.10s (100ms) elapsed, `dayCount > 0` | ✅ PASS |
| Goal progress query executável, <100ms | Same | `query_bench_test.go:105-124` — re-run: 0.05s (50ms) elapsed, `goalAmount > 0` | ✅ PASS |
| Index explain plan otimizado | EXPLAIN shows `idx_driver_ended` used for daily_earnings and weekly_stats | `query_bench_test.go:61-63`, `query_bench_test.go:100-102` (`assertExplainUsesIndex`) — pass | ⚠️ Spec-precision gap — same weak substring-match issue as the T13 EXPLAIN test (confirmed by mutation sensor #3); also, **no EXPLAIN assertion exists for the `goal_progress` query** (only correctness+timing are checked for it) |
| Slow query log vazio | `mysql.slow_log` has 0 entries for `trips` queries above the 100ms threshold | `query_bench_test.go:126`, `:349-358` (`assertSlowLogEmpty`) — pass | ✅ PASS |

**Status**: ⚠️ Gaps present — all core functional behavior is correctly implemented and passing, but test-coverage gaps exist for 3 of 6 tables' FK/constraint behavior, and the EXPLAIN-based "index usage" assertions are weaker than their gate wording implies (confirmed via sensor mutation).

---

## Discrimination Sensor

Sensor ran directly against the live, already-migrated MySQL database (`pilot_db`) via targeted `ALTER TABLE` statements simulating each SQL-level mutation, rather than a full `migrate down`/`up` cycle — because a full down would cascade-drop the shared 1,025,800-row `trips` table seeded for T14, which the task brief explicitly said not to delete, and re-seeding takes minutes. Each mutation was verified applied (via `information_schema` / `SHOW INDEX`), the affected test(s) were re-run, then the mutation was reverted and re-verified back to the original state. Final state confirmed: `migrate version` = 6, not dirty; `idx_driver_ended` = `(driver_id, ended_at)`; `trips_ibfk_1` = `CASCADE`; `check_rating` constraint present; `trips` row count unchanged at 1,025,800.

| # | Mutation | Applied via | Test(s) re-run | Killed? |
| - | -------- | ----------- | --------------- | ------- |
| 1 | Dropped `CHECK (rating >= 0 AND rating <= 5)` on `drivers` (mirrors `000001_create_drivers.up.sql:26`) | `ALTER TABLE drivers DROP CHECK check_rating` | `TestCheckConstraint_RejectsInvalidRating` (`schema_test.go:150`) | ✅ Killed — `expected check_rating violation for rating=6.0, got nil error` |
| 2 | Removed `ON DELETE CASCADE` from `trips.driver_id` FK, changed to default `NO ACTION` (mirrors `000002_create_trips.up.sql:26`) | `ALTER TABLE trips DROP FOREIGN KEY trips_ibfk_1; ALTER TABLE trips ADD CONSTRAINT trips_ibfk_1 FOREIGN KEY (driver_id) REFERENCES drivers(id)` | `TestForeignKey_CascadeDeletesTrips` (`schema_test.go:126`) | ✅ Killed — delete errored with MySQL 1451 (parent row referenced) |
| 3 | Reversed `idx_driver_ended` column order from `(driver_id, ended_at)` to `(ended_at, driver_id)` (mirrors `000002_create_trips.up.sql:29`) | `ALTER TABLE trips DROP INDEX idx_driver_ended, ADD INDEX idx_driver_ended (ended_at, driver_id)` | `TestExplain_UsesDriverEndedIndex` (`schema_test.go:220`) — **PASSED (survived)**; `TestQueryPerformance_Under1MRows/daily_earnings` and `/weekly_stats` (`query_bench_test.go:40`, `:66`, gated behind `RUN_PERF_TESTS=1`) — **FAILED (killed)** | ⚠️ Mixed — survived the default, always-run gate (`schema_test.go`'s check); only killed by the expensive perf test, which is not part of the normal `go test ./...` gate. Root cause: `TestExplain_UsesDriverEndedIndex` asserts `strings.Contains(planJSON, "idx_driver_ended")`, which matches the index name appearing in `possible_keys` even when MySQL's actual chosen `key` is a different index — it never checks the `"key"` field specifically. |

**Sensor depth**: lightweight (3 mutations, default tier)
**Result**: 2/3 fully killed by the default gate; 1/3 only killed by the perf-gated test and via a coincidental match, not a targeted assertion — **effectively a partial survival** → flagged as a fix task below.

---

## Code Quality

| Principle | Status |
| --- | --- |
| No features beyond what was asked | ✅ — migration tool, 6 tables, index/constraint tests, perf tests; nothing extraneous |
| No abstractions for single-use code | ✅ |
| No unnecessary "flexibility" added | ✅ |
| Only touched files required for task | ✅ — diff is exactly `cmd/migrate/main.go`, `migrations/*`, `go.mod`/`go.sum` (dependency additions are golang-migrate + its transitive graph, expected) |
| Didn't "improve" unrelated code | ✅ |
| Matches existing patterns/style | ✅ — test style (table-driven where useful, `t.Helper()`, `t.Cleanup()`) matches `internal/config/config_test.go`; `gofmt -l` reports no issues |
| Would senior engineer approve? | ✅ overall, with the noted test-coverage caveats |
| Tests map to acceptance criteria and are non-shallow | ⚠️ — mostly yes (behavioral assertions, not just "no error"), but 3 of 6 tables have zero tests, so those ACs are effectively unclaimed |
| Spec-anchored outcome check | ⚠️ — EXPLAIN-based assertions check substring presence rather than the actually-chosen index (see sensor #3) |
| Per-layer coverage: DDL has 1:1 mapping to design.md tables; tests do not | ⚠️ — `payment_records`, `work_sessions`, `stats_cache` have DDL but no tests |
| Every test in scope maps to a spec AC / Gate item | ✅ — no unclaimed/extraneous tests found |
| Documented project guidelines followed | `pilot-backend` has no dedicated Go testing guideline doc — "none — strong defaults applied", and applied defaults are consistent with `internal/config/config_test.go` |

---

## Edge Cases

- [x] FK violation on insert (orphan trip) — handled, tested
- [x] FK cascade delete — handled, tested
- [x] CHECK constraint violations (both) — handled, tested
- [x] Duplicate unique key (`daily_goals` one-per-day) — handled, tested
- [x] FULLTEXT search — handled, tested
- [ ] FK `ON DELETE SET NULL` behavior (`work_sessions.daily_goal_id`) — NOT tested (only FK type in the schema that isn't CASCADE)
- [ ] `payment_records` / `work_sessions` / `stats_cache` any constraint or index behavior — NOT tested
- [ ] `migrate down` execution — NOT tested (no automated evidence)

---

## Gate Check

- **Gate command**: `go build ./...`, `go vet ./...`, `gofmt -l migrations/ cmd/migrate/`, `go test ./migrations/... -v -count=1`, and `RUN_PERF_TESTS=1 go test ./migrations/... -v -count=1 -run TestQueryPerformance_Under1MRows`
- **Result**:
  - `go build ./...` — clean, no output
  - `go vet ./...` — clean, no output
  - `gofmt -l migrations/ cmd/migrate/` — clean, no files need formatting
  - `go test ./migrations/...` (default) — 9 passed, 0 failed, 1 skipped (`TestQueryPerformance_Under1MRows`, justified — gated behind `RUN_PERF_TESTS=1` by design since it seeds/queries 1M+ rows)
  - `go test ./migrations/... -run TestQueryPerformance_Under1MRows` with `RUN_PERF_TESTS=1` — 1 passed (3 sub-tests: daily_earnings 60ms, weekly_stats 100ms, goal_progress 50ms — all < 100ms threshold, consistent with the 70ms/~100ms/50ms figures recorded in tasks.md)
- **Test count before feature**: 0 (no `migrations` package existed on `develop`)
- **Test count after feature**: 10 top-level test functions (9 always-run + 1 perf-gated), 4 sub-tests under `TestIndexes_Exist`, 3 sub-tests under `TestQueryPerformance_Under1MRows`
- **Delta**: +10 new top-level tests
- **Skipped tests**: `TestQueryPerformance_Under1MRows` under default `go test` — justified, documented in-code (`query_bench_test.go:13-18`), explicitly re-run separately by this Verifier with `RUN_PERF_TESTS=1` and passed
- **Failures**: none in the clean (unmutated) tree

---

## Fix Plans (if issues found)

### Fix 1: EXPLAIN-based "index used" assertions match on index name anywhere in the plan, not the actually-chosen index

- **Root cause**: `schema_test.go:230` (`TestExplain_UsesDriverEndedIndex`) and `query_bench_test.go:143-152` (`assertExplainUsesIndex`) both do `strings.Contains(planJSON, wantIndex)`. MySQL's `EXPLAIN FORMAT=JSON` always lists every index the optimizer *considered* under `possible_keys`, regardless of which one it picked under `"key"`. A column-order regression on `idx_driver_ended` (verified via sensor mutation #3) survives `TestExplain_UsesDriverEndedIndex` entirely, and is only caught by the perf test — a test that is gated behind `RUN_PERF_TESTS=1` and thus not part of the default `go test ./...` gate.
- **Fix task**: Change both assertions to parse the JSON and check `query_block.table.key == wantIndex` (the actually-chosen access path), not a raw substring match against the whole plan.
- **Priority**: Major — a real schema regression (bad index column order) can land and pass CI silently today.

### Fix 2: `payment_records`, `work_sessions`, `stats_cache` have zero test coverage

- **Root cause**: `schema_test.go` only exercises `drivers`, `trips`, `daily_goals`. The other 3 migrated tables (half of T12's scope) have DDL but no test proving their FKs, indexes, generated columns, or unique constraints actually work — including `idx_driver_date` on `payment_records`, a table design.md explicitly calls out in "Índices Críticos", and `work_sessions.daily_goal_id`'s `ON DELETE SET NULL`, the only non-CASCADE FK behavior in the whole schema.
- **Fix task**: Add tests analogous to the existing `trips`/`daily_goals` ones: FK rejection + cascade/set-null behavior for `payment_records.driver_id`, `work_sessions.driver_id`, `work_sessions.daily_goal_id`; existence check for `idx_driver_date`, `idx_driver_started`, `idx_driver_type_date`, `idx_expires`, `unique_stat`.
- **Priority**: Major — T12/T13 gate items "Foreign keys criadas/funcionam" and "Index stats mostram uso" are marked done but are only partially evidenced.

### Fix 3: No automated evidence for `migrate down`

- **Root cause**: No test in the diff invokes `migrate down`. T12's gate item and commit message both claim "Test all migrations up and down," but there is no `file:line` this Verifier could cite as executed evidence, and re-running it live was declined here specifically to avoid destroying the shared 1M-row perf dataset.
- **Fix task**: Low-cost option — add a narrow test/script that runs `migrate down` + `migrate up` against a throwaway database/schema (not `pilot_db`) in CI, so the rollback path gets exercised without touching shared dev data.
- **Priority**: Minor — the down SQL is trivially correct by inspection (`DROP TABLE IF EXISTS`), so functional risk is low; this is purely an evidence/process gap.

### Fix 4: No EXPLAIN/index-usage assertion for the `goal_progress` query

- **Root cause**: `query_bench_test.go:105-124` checks correctness and timing for `goal_progress` but never asserts which index MySQL used for the `daily_goals`/`trips` join, unlike the other two critical queries.
- **Fix task**: Add an `assertExplainUsesIndex`-style check (once Fix 1 lands) for the `goal_progress` query.
- **Priority**: Minor — timing already passes comfortably (50ms), so this is a completeness gap, not a known regression.

---

## Requirement Traceability Update

| Gate item | Previous Status | New Status |
| --- | --- | --- |
| T11 all items | ✅ Done | ✅ Verified |
| T12: migrations executáveis | ✅ Done | ✅ Verified |
| T12: rollback funciona | ✅ Done | ⚠️ Needs evidence (Fix 3) |
| T12/T13: foreign keys criadas/funcionam | ✅ Done | ⚠️ Needs fix (Fix 2) |
| T13: idx_driver_ended, idx_status, unique_driver_date | ✅ Done | ✅ Verified |
| T13: FULLTEXT search | ✅ Done | ✅ Verified |
| T13: CHECK constraints | ✅ Done | ✅ Verified |
| T13: index stats mostram uso | ✅ Done | ⚠️ Needs fix (Fix 1) |
| T14: all 3 queries executable, <100ms | ✅ Done | ✅ Verified |
| T14: index explain plan otimizado | ✅ Done | ⚠️ Needs fix (Fix 1, Fix 4) |
| T14: slow query log vazio | ✅ Done | ✅ Verified |

---

## Summary

**Overall**: ⚠️ Issues — the implemented schema, migration tool, and tested behaviors are all correct; the gaps are entirely in test *coverage breadth* (3 of 6 tables untested) and test *assertion precision* (EXPLAIN checks that don't verify the actually-chosen index), not in the SQL/Go implementation itself. Nothing here indicates broken production behavior.

**Spec-anchored check**: 14/18 gate items fully matched with strong evidence; 4 flagged ⚠️ (partial FK coverage, weak EXPLAIN assertions ×2, missing rollback-execution evidence)
**Sensor**: 2/3 mutations cleanly killed by the default gate; 1/3 (index column order) survived the default gate and was only killed by the perf-gated test via a coincidental match, not a targeted assertion
**Gate**: `go build`, `go vet`, `gofmt` all clean; `go test ./migrations/...` 9/9 passed (1 justified skip) + perf test 1/1 passed (3/3 sub-tests) when run explicitly

**What works**: All 6 tables migrate up cleanly and match design.md exactly; both CHECK constraints reject bad data; the one tested FK (`trips→drivers`) correctly rejects orphans and cascades deletes; `unique_driver_date` rejects duplicates; FULLTEXT search works; all 3 critical queries return correct data and run well under the 100ms/1M-row budget; slow query log stays empty.

**Issues found**: See Fix Plans 1–4 above (2 Major, 2 Minor). None indicate the shipped schema is wrong — they indicate the safety net around 3 of 6 tables and around index-usage assertions is thinner than the tasks.md gate checkboxes claim.

**Next steps**: Route Fix 1 and Fix 2 back as fix tasks (Major); Fix 3 and Fix 4 are optional low-risk follow-ups.
