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

// Router bundles the gin engine with the pieces handler registration needs.
type Router struct {
	Engine *gin.Engine
	// API is the "/api" group: CORS + logging + error handling already
	// apply (from Engine.Use), plus rate limiting when enabled. It carries
	// no auth requirement — protected routes must additionally use Auth.
	API *gin.RouterGroup
	// Auth is the JWT authentication middleware, applied by handler
	// registration code to whichever routes require a logged-in driver.
	Auth gin.HandlerFunc
}

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
	if cfg.RateLimitEnabled {
		window := time.Duration(cfg.RateLimitWindow) * time.Second
		api.Use(middleware.RateLimit(redisCache, cfg.RateLimitRequests, window))
	}

	return &Router{
		Engine: engine,
		API:    api,
		Auth:   middleware.JWTAuth(jwtMgr, redisCache),
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
