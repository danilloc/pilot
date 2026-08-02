// Package logger provides structured JSON logging built on zap. Log level is
// the only configurable dimension — encoding is always JSON so log output is
// safe to ship to log aggregators without a separate parser per environment.
package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger wraps zap's SugaredLogger for structured, leveled logging.
type Logger struct {
	*zap.SugaredLogger
}

// New builds a Logger emitting JSON at the given level (debug, info, warn,
// error). Unrecognized levels fall back to info.
func New(level string) *Logger {
	config := zap.NewProductionConfig()

	switch level {
	case "debug":
		config.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	case "info":
		config.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	case "warn":
		config.Level = zap.NewAtomicLevelAt(zapcore.WarnLevel)
	case "error":
		config.Level = zap.NewAtomicLevelAt(zapcore.ErrorLevel)
	default:
		config.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	}

	config.Encoding = "json"

	built, err := config.Build()
	if err != nil {
		// Falls back to a no-op logger rather than crashing the process on a
		// logging misconfiguration — logging must never be why the server fails to start.
		built = zap.NewNop()
	}

	return &Logger{built.Sugar()}
}
