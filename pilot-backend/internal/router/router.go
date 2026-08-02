// Package router assembles the gin engine: global middleware chain, the
// /health endpoint, and the /api route group that feature handlers
// (auth, drivers, trips, ...) register themselves onto.
package router

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"pilot-backend/internal/config"
	"pilot-backend/internal/middleware"
	"pilot-backend/pkg/cache"
	"pilot-backend/pkg/db"
	"pilot-backend/pkg/jwt"
	"pilot-backend/pkg/logger"
)

// RateLimiters holds one rate-limit middleware per category from
// api-endpoints/design.md's Rate Limiting table (Auth/Trips/Stats/Goals,
// each "req/min"), plus Default for routes the table doesn't call out
// (drivers, payments) — those keep the general RATE_LIMIT_REQUESTS/WINDOW
// config. All are no-op pass-throughs when RATE_LIMIT_ENABLED=false.
type RateLimiters struct {
	Default gin.HandlerFunc
	Auth    gin.HandlerFunc
	Trips   gin.HandlerFunc
	Stats   gin.HandlerFunc
	Goals   gin.HandlerFunc
}

// Router bundles the gin engine with the pieces handler registration needs.
type Router struct {
	Engine *gin.Engine
	// API is the "/api" group: CORS + logging + error handling already
	// apply (from Engine.Use). It carries no auth or rate-limit
	// requirement — handlers apply Auth and the relevant RateLimit
	// themselves, since limits differ by route category.
	API *gin.RouterGroup
	// Auth is the JWT authentication middleware, applied by handler
	// registration code to whichever routes require a logged-in driver.
	Auth gin.HandlerFunc
	// RateLimit is applied by handler registration code, one field per
	// design.md category.
	RateLimit RateLimiters
}

const rateLimitPerMinuteWindow = time.Minute

// New builds the engine and its global middleware chain.
func New(cfg *config.Config, log *logger.Logger, jwtMgr *jwt.Manager, redisCache *cache.Redis, gormDB *gorm.DB) *Router {
	gin.SetMode(cfg.GinMode)

	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(middleware.RequestLogger(log))
	engine.Use(middleware.CORS(cfg.CORSOrigins))
	engine.Use(middleware.ErrorHandler(log))

	engine.GET("/health", healthHandler(gormDB, redisCache))

	api := engine.Group("/api")

	return &Router{
		Engine:    engine,
		API:       api,
		Auth:      middleware.JWTAuth(jwtMgr, redisCache),
		RateLimit: buildRateLimiters(cfg, redisCache),
	}
}

func buildRateLimiters(cfg *config.Config, redisCache *cache.Redis) RateLimiters {
	if !cfg.RateLimitEnabled {
		passthrough := func(c *gin.Context) { c.Next() }
		return RateLimiters{Default: passthrough, Auth: passthrough, Trips: passthrough, Stats: passthrough, Goals: passthrough}
	}

	defaultWindow := time.Duration(cfg.RateLimitWindow) * time.Second
	return RateLimiters{
		Default: middleware.RateLimit(redisCache, cfg.RateLimitRequests, defaultWindow),
		Auth:    middleware.RateLimit(redisCache, 5, rateLimitPerMinuteWindow),
		Trips:   middleware.RateLimit(redisCache, 30, rateLimitPerMinuteWindow),
		Stats:   middleware.RateLimit(redisCache, 60, rateLimitPerMinuteWindow),
		Goals:   middleware.RateLimit(redisCache, 30, rateLimitPerMinuteWindow),
	}
}

func healthHandler(gormDB *gorm.DB, redisCache *cache.Redis) gin.HandlerFunc {
	return func(c *gin.Context) {
		dbStatus := "connected"
		if err := db.HealthCheck(gormDB); err != nil {
			dbStatus = "disconnected"
		}

		cacheStatus := "connected"
		if err := redisCache.Ping(c.Request.Context()); err != nil {
			cacheStatus = "disconnected"
		}

		status := http.StatusOK
		overall := "healthy"
		if dbStatus != "connected" || cacheStatus != "connected" {
			status = http.StatusServiceUnavailable
			overall = "unhealthy"
		}

		c.JSON(status, gin.H{
			"status":   overall,
			"database": dbStatus,
			"cache":    cacheStatus,
		})
	}
}
