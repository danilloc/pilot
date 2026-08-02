# LESSONS — auto-maintained by scripts/lessons.py

> Machine-owned. Do NOT hand-edit. Changes are overwritten on the next `lessons.py` write.
> Canonical state lives in `.specs/lessons.json`. Edit lessons only via the script.
> promote_threshold=2 distinct features · window_days=45 · quarantine_threshold=2

## Confirmed (load these at Specify/Design)

Corroborated across multiple features. Safe to apply as guidance.

_none_

## Candidates (under observation — do NOT load as guidance yet)

Seen once or not yet corroborated. Tracked, not trusted.

### L-001 — Assert EXPLAIN FORMAT=JSON index usage by checking the query_block.table.key field for the exact index name, not substring presence anywhere in the plan, since possible_keys always lists every considered index even when unused
- signal: `surviving_mutant` · recurrence: 1 feature(s) · scope: `migrations` · harmful: 0
- features: database
- evidence: validation.md mutation #3 / schema_test.go:220 TestExplain_UsesDriverEndedIndex (migrations)
- last seen: 2026-08-02T15:53:26Z

### L-002 — When a migration task creates multiple tables, add at least one FK-rejection test and one index-existence test per table, not just the first few, so coverage does not cluster on early tables while later ones ship untested
- signal: `ac_gap` · recurrence: 1 feature(s) · scope: `migrations` · harmful: 0
- features: database
- evidence: validation.md Fix 2 / tasks.md T12-T13 Foreign keys criadas/funcionam gate (migrations)
- last seen: 2026-08-02T15:53:26Z

## Quarantined (failed when applied — ignore)

A confirmed lesson that recurred alongside failure. Kept for the maintainer to review.

_none_
