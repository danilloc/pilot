package main

import (
	"database/sql"
	"fmt"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"pilot-backend/internal/config"
)

func TestSslModeToTLSParam(t *testing.T) {
	tests := map[string]string{
		"":          "false",
		"disable":   "false",
		"require":   "true",
		"verify-ca": "true",
	}
	for sslMode, want := range tests {
		if got := sslModeToTLSParam(sslMode); got != want {
			t.Errorf("sslModeToTLSParam(%q) = %q, want %q", sslMode, got, want)
		}
	}
}

// testMigrate builds a *migrate.Migrate against a throwaway database (not
// pilot_db, so it never touches the shared dev dataset), pointed at the
// real migrations directory two levels up.
func testMigrate(t *testing.T) *migrate.Migrate {
	t.Helper()

	cfg := config.Load()
	adminDSN := fmt.Sprintf("%s:%s@tcp(%s:%d)/", cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort)
	adminDB, err := sql.Open("mysql", adminDSN)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() { adminDB.Close() })
	if err := adminDB.Ping(); err != nil {
		t.Skipf("mysql not reachable, skipping cmd/migrate test: %v", err)
	}

	const testSchema = "pilot_db_cmd_migrate_test"
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

	dsn := fmt.Sprintf("mysql://%s:%s@tcp(%s:%d)/%s?multiStatements=true&tls=%s",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, testSchema, sslModeToTLSParam(cfg.DBSSLMode))
	m, err := migrate.New("file://../../migrations", dsn)
	if err != nil {
		t.Fatalf("migrate.New() error = %v", err)
	}
	t.Cleanup(func() { m.Close() })
	return m
}

func TestRun_UpDownVersion(t *testing.T) {
	m := testMigrate(t)

	if err := run(m, "up"); err != nil {
		t.Fatalf(`run(m, "up") error = %v`, err)
	}
	if err := run(m, "version"); err != nil {
		t.Fatalf(`run(m, "version") error = %v`, err)
	}
	if err := run(m, "down"); err != nil {
		t.Fatalf(`run(m, "down") error = %v`, err)
	}
	// "up" again after "down" must be idempotent-safe (ErrNoChange is
	// swallowed, not just a fresh migrate — but since down dropped
	// everything, this re-applies from scratch).
	if err := run(m, "up"); err != nil {
		t.Fatalf(`second run(m, "up") error = %v`, err)
	}
}

func TestRun_UnknownCommand(t *testing.T) {
	m := testMigrate(t)

	err := run(m, "sideways")
	if err == nil {
		t.Fatal(`run(m, "sideways") error = nil, want an error for an unknown command`)
	}
}
