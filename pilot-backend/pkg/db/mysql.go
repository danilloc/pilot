// Package db manages the MySQL connection used by the application, wrapping
// GORM with the pooling and TLS settings the deployment environment requires.
package db

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"pilot-backend/internal/config"
	applogger "pilot-backend/pkg/logger"
)

const (
	maxIdleConns    = 10
	maxOpenConns    = 50
	connMaxLifetime = time.Hour
)

// Connect opens a pooled connection to MySQL using the given configuration.
// Query logging is only enabled outside production, and the connection uses
// cfg.DBSSLMode to control the TLS mode negotiated with the server.
func Connect(cfg *config.Config, log *applogger.Logger) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?parseTime=true&loc=Local&tls=%s",
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBName,
		sslModeToTLSParam(cfg.DBSSLMode),
	)

	logLevel := gormlogger.Info
	if cfg.Environment == "production" {
		logLevel = gormlogger.Silent
	}

	gormDB, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(logLevel),
	})
	if err != nil {
		log.Errorw("db.connection_failed", "error", err.Error())
		return nil, err
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		log.Errorw("db.pool_unavailable", "error", err.Error())
		return nil, err
	}

	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetConnMaxLifetime(connMaxLifetime)

	log.Infow("db.connected", "host", cfg.DBHost, "database", cfg.DBName)
	return gormDB, nil
}

// HealthCheck pings the underlying connection, returning an error if the
// database is unreachable.
func HealthCheck(gormDB *gorm.DB) error {
	if gormDB == nil {
		return errors.New("db: nil connection")
	}
	sqlDB, err := gormDB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

// sslModeToTLSParam maps our DB_SSLMODE values onto the go-sql-driver tls
// query param: "disable" leaves TLS off, anything else ("require", etc.)
// enables it via the driver's built-in "true" config.
func sslModeToTLSParam(sslMode string) string {
	if sslMode == "" || sslMode == "disable" {
		return "false"
	}
	return "true"
}
