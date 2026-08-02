package main

import (
	"github.com/joho/godotenv"

	"pilot-backend/internal/config"
	"pilot-backend/pkg/logger"
)

func main() {
	// .env is optional: in production/CI, config comes from real environment
	// variables, so a missing file here must not stop the server from starting.
	_ = godotenv.Load(".env", ".env.local")

	cfg := config.Load()

	log := logger.New(cfg.LogLevel)
	defer log.Sync()

	log.Infow("server.starting", "port", cfg.Port, "env", cfg.Environment)
	log.Infow("server.ready")
}
