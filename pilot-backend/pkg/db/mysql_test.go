package db

import (
	"testing"

	"pilot-backend/internal/config"
	"pilot-backend/pkg/logger"
)

// TestConnectAndHealthCheck is an integration test: it needs a reachable
// MySQL instance (see docker-compose.yml). It skips instead of failing when
// none is available, e.g. in environments without Docker.
func TestConnectAndHealthCheck(t *testing.T) {
	cfg := config.Load()
	log := logger.New("error", "test")

	gormDB, err := Connect(cfg, log)
	if err != nil {
		t.Skipf("mysql not reachable, skipping integration test: %v", err)
	}

	if err := HealthCheck(gormDB); err != nil {
		t.Fatalf("HealthCheck() error = %v, want nil", err)
	}

	sqlDB, _ := gormDB.DB()
	defer sqlDB.Close()
}

func TestHealthCheckNilConnection(t *testing.T) {
	if err := HealthCheck(nil); err == nil {
		t.Fatal("HealthCheck(nil) error = nil, want error")
	}
}
