// Command migrate applies or rolls back the database schema using
// golang-migrate, reading SQL files from ./migrations and the database
// connection from the same environment/.env config the server uses.
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"

	"pilot-backend/internal/config"
)

const migrationsPath = "file://migrations"

func main() {
	// .env is optional: in production/CI, config comes from real environment
	// variables, so a missing file here must not stop the tool from running.
	_ = godotenv.Load(".env", ".env.local")

	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: migrate <up|down|version>")
		os.Exit(1)
	}

	cfg := config.Load()
	dsn := fmt.Sprintf(
		"mysql://%s:%s@tcp(%s:%d)/%s?multiStatements=true&tls=%s",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName, sslModeToTLSParam(cfg.DBSSLMode),
	)

	m, err := migrate.New(migrationsPath, dsn)
	if err != nil {
		fmt.Fprintln(os.Stderr, "migrate: init failed:", err)
		os.Exit(1)
	}
	defer m.Close()

	if err := run(m, os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, "migrate:", err)
		os.Exit(1)
	}
}

func run(m *migrate.Migrate, command string) error {
	switch command {
	case "up":
		err := m.Up()
		if err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return err
		}
		fmt.Println("migrate: up complete")
	case "down":
		err := m.Down()
		if err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return err
		}
		fmt.Println("migrate: down complete")
	case "version":
		version, dirty, err := m.Version()
		if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
			return err
		}
		fmt.Printf("migrate: version=%d dirty=%v\n", version, dirty)
	default:
		return fmt.Errorf("unknown command %q (want up, down, or version)", command)
	}
	return nil
}

// sslModeToTLSParam maps DB_SSLMODE onto the go-sql-driver tls query param,
// mirroring pkg/db.Connect's DSN construction so the migration tool and the
// server negotiate the connection identically.
func sslModeToTLSParam(sslMode string) string {
	if sslMode == "" || sslMode == "disable" {
		return "false"
	}
	return "true"
}
