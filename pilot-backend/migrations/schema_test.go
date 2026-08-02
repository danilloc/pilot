// Package migrations verifies the schema applied by the SQL files in this
// directory: indexes exist, foreign keys and CHECK constraints actually
// reject bad data, FULLTEXT search works, and the query planner picks the
// indexes we expect. These tests assume `migrate up` has already been run
// against the target database (see cmd/migrate) and are skipped if MySQL
// isn't reachable.
package migrations

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"

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
		{"trips", "ft_city"},
		{"daily_goals", "unique_driver_date"},
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

	var plan string
	err := db.QueryRow(
		"EXPLAIN FORMAT=JSON SELECT * FROM trips WHERE driver_id = 1 AND ended_at BETWEEN '2026-01-01' AND '2026-12-31'",
	).Scan(&plan)
	if err != nil {
		t.Fatalf("EXPLAIN query error = %v", err)
	}
	if !strings.Contains(plan, "idx_driver_ended") {
		t.Errorf("EXPLAIN plan does not use idx_driver_ended:\n%s", plan)
	}
}
