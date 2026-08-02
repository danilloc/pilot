package main

import (
	"fmt"

	"github.com/joho/godotenv"

	"pilot-backend/internal/config"
	"pilot-backend/internal/handler"
	"pilot-backend/internal/oauth"
	"pilot-backend/internal/repository"
	"pilot-backend/internal/router"
	"pilot-backend/internal/service"
	"pilot-backend/pkg/cache"
	"pilot-backend/pkg/crypto"
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

	encryptor, err := crypto.NewEncryptor(cfg.EncryptionKey)
	if err != nil {
		log.Fatalw("server.encryptor_init_failed", "error", err.Error())
	}

	driverRepo := repository.NewDriverRepository(gormDB)
	uberClient := oauth.NewClient(cfg.UberClientID, cfg.UberClientSecret, cfg.UberRedirectURI)
	authService := service.NewAuthService(driverRepo, jwtMgr, cfg.JWTExpiry, encryptor, uberClient, redisCache, log)
	authHandler := handler.NewAuthHandler(authService, driverRepo)
	driverService := service.NewDriverService(driverRepo, encryptor, uberClient, log)
	driverHandler := handler.NewDriverHandler(driverService)
	tripRepo := repository.NewTripRepository(gormDB)
	tripService := service.NewTripService(tripRepo, driverRepo, encryptor, uberClient, log)
	tripHandler := handler.NewTripHandler(tripService)
	statsService := service.NewStatsService(gormDB, redisCache, log)
	statsHandler := handler.NewStatsHandler(statsService)

	r := router.New(cfg, log, jwtMgr, redisCache, gormDB)
	authHandler.Register(r.API, r.Auth)
	driverHandler.Register(r.API, r.Auth)
	tripHandler.Register(r.API, r.Auth)
	statsHandler.Register(r.API, r.Auth)

	log.Infow("server.ready")
	if err := r.Engine.Run(fmt.Sprintf(":%d", cfg.Port)); err != nil {
		log.Fatalw("server.stopped", "error", err.Error())
	}
}
