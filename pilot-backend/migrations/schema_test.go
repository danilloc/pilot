// Package migrations verifies the schema applied by the SQL files in this
// directory: indexes exist, foreign keys and CHECK constraints actually
// reject bad data, FULLTEXT search works, and the query planner picks the
// indexes we expect. These tests assume `migrate up` has already been run
// against the target database (see cmd/migrate) and are skipped if MySQL
// isn't reachable.
package migrations

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"

	"pilot-backend/internal/config"
)

func testDB(t *testing.T) *sql.DB {
	t.Helper()

	cfg := config.Load()
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?parseTime=true&loc=Local",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Skipf("mysql not reachable, skipping schema test: %v", err)
	}

	var tableCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'drivers'").Scan(&tableCount); err != nil {
		t.Fatalf("checking schema state error = %v", err)
	}
	if tableCount == 0 {
		t.Skip("schema not migrated (run `go run ./cmd/migrate up` first), skipping schema test")
	}

	t.Cleanup(func() { db.Close() })
	return db
}

func indexExists(t *testing.T, db *sql.DB, table, index string) bool {
	t.Helper()
	var count int
	err := db.QueryRow(
		"SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = ? AND index_name = ?",
		table, index,
	).Scan(&count)
	if err != nil {
		t.Fatalf("querying information_schema.statistics error = %v", err)
	}
	return count > 0
}

func TestIndexes_Exist(t *testing.T) {
	db := testDB(t)

	tests := []struct {
		table string
		index string
	}{
		{"trips", "idx_driver_ended"},
		{"trips", "idx_status"},
		{"trips", "idx_driver_id"},
		{"trips", "ft_city"},
		{"daily_goals", "unique_driver_date"},
		{"payment_records", "idx_driver_date"},
		{"work_sessions", "idx_driver_started"},
		{"stats_cache", "unique_stat"},
		{"stats_cache", "idx_driver_type_date"},
		{"stats_cache", "idx_expires"},
	}

	for _, tt := range tests {
		t.Run(tt.table+"."+tt.index, func(t *testing.T) {
			if !indexExists(t, db, tt.table, tt.index) {
				t.Errorf("index %q not found on table %q", tt.index, tt.table)
			}
		})
	}
}

// withTestDriver inserts a throwaway driver (FK target for trips/daily_goals
// tests) and returns its ID, cleaning up after the test.
func withTestDriver(t *testing.T, db *sql.DB, uberID string) int64 {
	t.Helper()
	res, err := db.Exec(
		"INSERT INTO drivers (uuid, uber_id, name, email, rating) VALUES (UUID(), ?, 'Schema Test Driver', ?, 4.5)",
		uberID, uberID+"@example.com",
	)
	if err != nil {
		t.Fatalf("insert test driver error = %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId() error = %v", err)
	}
	t.Cleanup(func() {
		db.Exec("SET FOREIGN_KEY_CHECKS=0")
		db.Exec("DELETE FROM trips WHERE driver_id = ?", id)
		db.Exec("DELETE FROM daily_goals WHERE driver_id = ?", id)
		db.Exec("DELETE FROM payment_records WHERE driver_id = ?", id)
		db.Exec("DELETE FROM work_sessions WHERE driver_id = ?", id)
		db.Exec("DELETE FROM stats_cache WHERE driver_id = ?", id)
		db.Exec("DELETE FROM drivers WHERE id = ?", id)
		db.Exec("SET FOREIGN_KEY_CHECKS=1")
	})
	return id
}

func TestForeignKey_RejectsOrphanTrip(t *testing.T) {
	db := testDB(t)

	_, err := db.Exec(
		"INSERT INTO trips (uuid, driver_id, uber_trip_id, started_at, ended_at, distance_km, fare_value) VALUES (UUID(), 999999999, ?, NOW(), NOW(), 1, 1)",
		"fk-test-"+t.Name(),
	)
	if err == nil {
		t.Fatal("expected a foreign key violation inserting a trip with a nonexistent driver_id, got nil error")
	}
	var mysqlErr *mysql.MySQLError
	if !errors.As(err, &mysqlErr) || mysqlErr.Number != 1452 {
		t.Fatalf("error = %v, want a MySQL foreign key constraint error (1452)", err)
	}
}

func TestForeignKey_CascadeDeletesTrips(t *testing.T) {
	db := testDB(t)
	driverID := withTestDriver(t, db, "fk-cascade-"+t.Name())

	if _, err := db.Exec(
		"INSERT INTO trips (uuid, driver_id, uber_trip_id, started_at, ended_at, distance_km, fare_value) VALUES (UUID(), ?, ?, NOW(), NOW(), 1, 1)",
		driverID, "cascade-"+t.Name(),
	); err != nil {
		t.Fatalf("insert trip error = %v", err)
	}

	if _, err := db.Exec("DELETE FROM drivers WHERE id = ?", driverID); err != nil {
		t.Fatalf("delete driver error = %v", err)
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM trips WHERE driver_id = ?", driverID).Scan(&count); err != nil {
		t.Fatalf("count trips error = %v", err)
	}
	if count != 0 {
		t.Errorf("trips remaining after driver delete = %d, want 0 (ON DELETE CASCADE)", count)
	}
}

func TestCheckConstraint_RejectsInvalidRating(t *testing.T) {
	db := testDB(t)

	_, err := db.Exec(
		"INSERT INTO drivers (uuid, uber_id, name, email, rating) VALUES (UUID(), ?, 'Bad Rating', ?, 6.0)",
		"check-rating-"+t.Name(), "check-rating-"+t.Name()+"@example.com",
	)
	if err == nil {
		t.Fatal("expected check_rating violation for rating=6.0, got nil error")
	}
	if !strings.Contains(err.Error(), "check_rating") {
		t.Fatalf("error = %v, want it to reference check_rating", err)
	}
}

func TestCheckConstraint_RejectsNonPositiveGoalAmount(t *testing.T) {
	db := testDB(t)
	driverID := withTestDriver(t, db, "check-goal-"+t.Name())

	_, err := db.Exec(
		"INSERT INTO daily_goals (uuid, driver_id, goal_date, goal_amount) VALUES (UUID(), ?, CURDATE(), 0)",
		driverID,
	)
	if err == nil {
		t.Fatal("expected check_goal_positive violation for goal_amount=0, got nil error")
	}
	if !strings.Contains(err.Error(), "check_goal_positive") {
		t.Fatalf("error = %v, want it to reference check_goal_positive", err)
	}
}

func TestUniqueDriverDate_RejectsDuplicateGoal(t *testing.T) {
	db := testDB(t)
	driverID := withTestDriver(t, db, "unique-goal-"+t.Name())

	insert := "INSERT INTO daily_goals (uuid, driver_id, goal_date, goal_amount) VALUES (UUID(), ?, '2026-06-01', 100)"
	if _, err := db.Exec(insert, driverID); err != nil {
		t.Fatalf("first insert error = %v", err)
	}

	_, err := db.Exec(insert, driverID)
	if err == nil {
		t.Fatal("expected unique_driver_date violation on duplicate (driver_id, goal_date), got nil error")
	}
}

func TestForeignKey_RejectsOrphanPaymentRecord(t *testing.T) {
	db := testDB(t)

	_, err := db.Exec(
		"INSERT INTO payment_records (uuid, driver_id, amount, payment_date) VALUES (UUID(), 999999999, 100, CURDATE())",
	)
	if err == nil {
		t.Fatal("expected a foreign key violation inserting a payment_record with a nonexistent driver_id, got nil error")
	}
	var mysqlErr *mysql.MySQLError
	if !errors.As(err, &mysqlErr) || mysqlErr.Number != 1452 {
		t.Fatalf("error = %v, want a MySQL foreign key constraint error (1452)", err)
	}
}

func TestForeignKey_CascadeDeletesPaymentRecords(t *testing.T) {
	db := testDB(t)
	driverID := withTestDriver(t, db, "fk-payment-cascade-"+t.Name())

	if _, err := db.Exec(
		"INSERT INTO payment_records (uuid, driver_id, amount, payment_date) VALUES (UUID(), ?, 100, CURDATE())",
		driverID,
	); err != nil {
		t.Fatalf("insert payment_record error = %v", err)
	}

	if _, err := db.Exec("DELETE FROM drivers WHERE id = ?", driverID); err != nil {
		t.Fatalf("delete driver error = %v", err)
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM payment_records WHERE driver_id = ?", driverID).Scan(&count); err != nil {
		t.Fatalf("count payment_records error = %v", err)
	}
	if count != 0 {
		t.Errorf("payment_records remaining after driver delete = %d, want 0 (ON DELETE CASCADE)", count)
	}
}

func TestForeignKey_RejectsOrphanWorkSession(t *testing.T) {
	db := testDB(t)

	_, err := db.Exec(
		"INSERT INTO work_sessions (uuid, driver_id, started_at) VALUES (UUID(), 999999999, NOW())",
	)
	if err == nil {
		t.Fatal("expected a foreign key violation inserting a work_session with a nonexistent driver_id, got nil error")
	}
	var mysqlErr *mysql.MySQLError
	if !errors.As(err, &mysqlErr) || mysqlErr.Number != 1452 {
		t.Fatalf("error = %v, want a MySQL foreign key constraint error (1452)", err)
	}
}

func TestForeignKey_CascadeDeletesWorkSessions(t *testing.T) {
	db := testDB(t)
	driverID := withTestDriver(t, db, "fk-worksession-cascade-"+t.Name())

	if _, err := db.Exec(
		"INSERT INTO work_sessions (uuid, driver_id, started_at) VALUES (UUID(), ?, NOW())",
		driverID,
	); err != nil {
		t.Fatalf("insert work_session error = %v", err)
	}

	if _, err := db.Exec("DELETE FROM drivers WHERE id = ?", driverID); err != nil {
		t.Fatalf("delete driver error = %v", err)
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM work_sessions WHERE driver_id = ?", driverID).Scan(&count); err != nil {
		t.Fatalf("count work_sessions error = %v", err)
	}
	if count != 0 {
		t.Errorf("work_sessions remaining after driver delete = %d, want 0 (ON DELETE CASCADE)", count)
	}
}

// TestForeignKey_WorkSessionDailyGoalSetsNull covers the schema's only
// non-CASCADE FK: deleting a daily_goal must null out
// work_sessions.daily_goal_id, not delete or block-delete the session.
func TestForeignKey_WorkSessionDailyGoalSetsNull(t *testing.T) {
	db := testDB(t)
	driverID := withTestDriver(t, db, "fk-setnull-"+t.Name())

	goalRes, err := db.Exec(
		"INSERT INTO daily_goals (uuid, driver_id, goal_date, goal_amount) VALUES (UUID(), ?, CURDATE(), 200)",
		driverID,
	)
	if err != nil {
		t.Fatalf("insert daily_goal error = %v", err)
	}
	goalID, err := goalRes.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId() error = %v", err)
	}

	var sessionID int64
	res, err := db.Exec(
		"INSERT INTO work_sessions (uuid, driver_id, started_at, daily_goal_id) VALUES (UUID(), ?, NOW(), ?)",
		driverID, goalID,
	)
	if err != nil {
		t.Fatalf("insert work_session error = %v", err)
	}
	sessionID, err = res.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId() error = %v", err)
	}

	if _, err := db.Exec("DELETE FROM daily_goals WHERE id = ?", goalID); err != nil {
		t.Fatalf("delete daily_goal error = %v", err)
	}

	var dailyGoalID sql.NullInt64
	if err := db.QueryRow("SELECT daily_goal_id FROM work_sessions WHERE id = ?", sessionID).Scan(&dailyGoalID); err != nil {
		t.Fatalf("query work_session error = %v (session should still exist, not cascade-deleted)", err)
	}
	if dailyGoalID.Valid {
		t.Errorf("daily_goal_id = %v, want NULL (ON DELETE SET NULL)", dailyGoalID.Int64)
	}
}

func TestUniqueStat_RejectsDuplicateStatsCache(t *testing.T) {
	db := testDB(t)
	driverID := withTestDriver(t, db, "unique-stat-"+t.Name())

	insert := "INSERT INTO stats_cache (driver_id, stat_type, stat_date) VALUES (?, 'daily', CURDATE())"
	if _, err := db.Exec(insert, driverID); err != nil {
		t.Fatalf("first insert error = %v", err)
	}

	_, err := db.Exec(insert, driverID)
	if err == nil {
		t.Fatal("expected unique_stat violation on duplicate (driver_id, stat_type, stat_date), got nil error")
	}
}

func TestFulltextSearch_MatchesCity(t *testing.T) {
	db := testDB(t)
	driverID := withTestDriver(t, db, "fulltext-"+t.Name())

	if _, err := db.Exec(
		"INSERT INTO trips (uuid, driver_id, uber_trip_id, started_at, ended_at, distance_km, fare_value, city) VALUES (UUID(), ?, ?, NOW(), NOW(), 1, 1, 'São Paulo')",
		driverID, "fulltext-"+t.Name(),
	); err != nil {
		t.Fatalf("insert trip error = %v", err)
	}

	var count int
	err := db.QueryRow(
		"SELECT COUNT(*) FROM trips WHERE driver_id = ? AND MATCH(city) AGAINST (? IN NATURAL LANGUAGE MODE)",
		driverID, "Paulo",
	).Scan(&count)
	if err != nil {
		t.Fatalf("fulltext query error = %v", err)
	}
	if count != 1 {
		t.Errorf("MATCH...AGAINST('Paulo') matched %d rows, want 1", count)
	}
}

func TestExplain_UsesDriverEndedIndex(t *testing.T) {
	db := testDB(t)
	driverID := withTestDriver(t, db, "explain-"+t.Name())

	// The composite index's second column only helps when the range
	// predicate is both sargable (no function wrapping the column — a raw
	// BETWEEN, unlike design.md's DATE(ended_at) = ... queries tested in
	// query_bench_test.go, which are NOT sargable on ended_at and
	// correctly use idx_driver_id instead) AND selective relative to the
	// driver's total rows. Spread 100 trips across 2 years so a 30-day
	// query window excludes most of them.
	now := time.Now()
	for i := 0; i < 100; i++ {
		started := now.AddDate(0, 0, -i*7) // one every 7 days, ~2 years back
		_, err := db.Exec(
			"INSERT INTO trips (uuid, driver_id, uber_trip_id, started_at, ended_at, distance_km, fare_value) VALUES (UUID(), ?, ?, ?, ?, 5, 20)",
			driverID, fmt.Sprintf("explain-seed-%d", i), started, started.Add(20*time.Minute),
		)
		if err != nil {
			t.Fatalf("seed trip %d error = %v", i, err)
		}
	}

	rangeStart := now.AddDate(0, 0, -30).Format("2006-01-02 15:04:05")
	rangeEnd := now.Format("2006-01-02 15:04:05")
	assertExplainUsesIndex(t, db,
		"SELECT * FROM trips WHERE driver_id = ? AND ended_at BETWEEN ? AND ?",
		[]interface{}{driverID, rangeStart, rangeEnd}, "trips", "idx_driver_ended")
}

// explainKeyForTable parses EXPLAIN FORMAT=JSON output and returns the
// index actually chosen (the "key" field, MySQL's real access path) for
// the given table, or "" if that table's node has no "key" (e.g. a full
// scan). Unlike a raw substring search, this only matches the index MySQL
// picked — not every index merely listed under "possible_keys".
func explainKeyForTable(t *testing.T, planJSON string, tableName string) string {
	t.Helper()

	var root map[string]interface{}
	if err := json.Unmarshal([]byte(planJSON), &root); err != nil {
		t.Fatalf("parsing EXPLAIN JSON error = %v\nplan:\n%s", err, planJSON)
	}

	var found string
	var walk func(v interface{})
	walk = func(v interface{}) {
		switch val := v.(type) {
		case map[string]interface{}:
			if tn, ok := val["table_name"].(string); ok && tn == tableName {
				if key, ok := val["key"].(string); ok {
					found = key
				}
			}
			for _, child := range val {
				walk(child)
			}
		case []interface{}:
			for _, child := range val {
				walk(child)
			}
		}
	}
	walk(root)
	return found
}

// assertExplainUsesIndex runs "EXPLAIN FORMAT=JSON <query>" and asserts
// that wantIndex is the index MySQL actually chose (the "key" field) for
// tableName — not merely a candidate under "possible_keys".
func assertExplainUsesIndex(t *testing.T, db *sql.DB, query string, args []interface{}, tableName, wantIndex string) {
	t.Helper()
	var plan string
	if err := db.QueryRow("EXPLAIN FORMAT=JSON "+query, args...).Scan(&plan); err != nil {
		t.Fatalf("EXPLAIN error = %v", err)
	}
	got := explainKeyForTable(t, plan, tableName)
	if got != wantIndex {
		t.Errorf("EXPLAIN chose key=%q for table %q, want %q\nplan:\n%s", got, tableName, wantIndex, plan)
	}
}
