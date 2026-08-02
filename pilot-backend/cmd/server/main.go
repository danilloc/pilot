package main

import (
	"fmt"

	"github.com/joho/godotenv"

	"pilot-backend/internal/config"
	"pilot-backend/internal/router"
	"pilot-backend/pkg/cache"
	"pilot-backend/pkg/db"
	"pilot-backend/pkg/jwt"
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

	gormDB, err := db.Connect(cfg, log)
	if err != nil {
		log.Fatalw("server.startup_failed", "error", err.Error())
	}
	sqlDB, _ := gormDB.DB()
	defer sqlDB.Close()

	redisCache := cache.NewRedis(cfg)
	defer redisCache.Close()

	jwtMgr := jwt.NewManager(cfg.JWTSecret)

	r := router.New(cfg, log, jwtMgr, redisCache, gormDB)

	log.Infow("server.ready")
	if err := r.Engine.Run(fmt.Sprintf(":%d", cfg.Port)); err != nil {
		log.Fatalw("server.stopped", "error", err.Error())
	}
}
