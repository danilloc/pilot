package migrations

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"pilot-backend/internal/config"
)

// migratedTables lists every table created across the six migrations, in
// their up-migration (creation) order.
var migratedTables = []string{"drivers", "trips", "daily_goals", "payment_records", "work_sessions", "stats_cache"}

// TestMigrateUpDown_RoundTrip proves T12's "Rollback funciona em cada uma"
// gate item with an automated, re-runnable test rather than a one-off
// manual check: it runs the full migrate Up/Down cycle against a throwaway
// database (not pilot_db), so the shared dev data seeded for other tests
// is never touched.
func TestMigrateUpDown_RoundTrip(t *testing.T) {
	cfg := config.Load()

	adminDSN := fmt.Sprintf("%s:%s@tcp(%s:%d)/", cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort)
	adminDB, err := sql.Open("mysql", adminDSN)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	// t.Cleanup runs LIFO, so registering the Close cleanup first (runs
	// last) guarantees the drop-database cleanup below (registered later,
	// runs first) executes on a still-open connection.
	t.Cleanup(func() { adminDB.Close() })
	if err := adminDB.Ping(); err != nil {
		t.Skipf("mysql not reachable, skipping rollback test: %v", err)
	}

	const testSchema = "pilot_db_migrate_rollback_test"
	if _, err := adminDB.Exec("DROP DATABASE IF EXISTS " + testSchema); err != nil {
		t.Fatalf("drop test schema error = %v", err)
	}
	if _, err := adminDB.Exec("CREATE DATABASE " + testSchema); err != nil {
		t.Fatalf("create test schema error = %v", err)
	}
	t.Cleanup(func() {
		if _, err := adminDB.Exec("DROP DATABASE IF EXISTS " + testSchema); err != nil {
			t.Errorf("cleanup: drop test schema error = %v", err)
		}
	})

	dsn := fmt.Sprintf("mysql://%s:%s@tcp(%s:%d)/%s?multiStatements=true", cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, testSchema)
	// "file://." resolves against this test binary's working directory,
	// which `go test` sets to this package's own source directory — the
	// same one the .sql migration files live in.
	m, err := migrate.New("file://.", dsn)
	if err != nil {
		t.Fatalf("migrate.New() error = %v", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil {
		t.Fatalf("Up() error = %v", err)
	}
	assertTablesExist(t, adminDB, testSchema, migratedTables, true)

	if err := m.Down(); err != nil {
		t.Fatalf("Down() error = %v", err)
	}
	assertTablesExist(t, adminDB, testSchema, migratedTables, false)
}

func assertTablesExist(t *testing.T, adminDB *sql.DB, schema string, tables []string, wantExist bool) {
	t.Helper()
	for _, table := range tables {
		var count int
		if err := adminDB.QueryRow(
			"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = ? AND table_name = ?",
			schema, table,
		).Scan(&count); err != nil {
			t.Fatalf("checking table %s error = %v", table, err)
		}
		exists := count > 0
		if exists != wantExist {
			t.Errorf("table %q exists = %v, want %v", table, exists, wantExist)
		}
	}
}
