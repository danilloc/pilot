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
//
// environment controls stack traces on Error-level logs: zap's production
// config attaches one to every Error+ entry by default, which is useful
// while developing but noisy (and a minor info-exposure surface) once logs
// ship to an aggregator — so it's only enabled when environment is
// "development". Any other value (including "" or "production") disables it.
func New(level, environment string) *Logger {
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

	var opts []zap.Option
	if environment != "development" {
		// zapcore.InvalidLevel sorts above every real level, so no entry
		// ever qualifies — this disables stacktrace capture entirely.
		opts = append(opts, zap.AddStacktrace(zapcore.InvalidLevel))
	}

	built, err := config.Build(opts...)
	if err != nil {
		// Falls back to a no-op logger rather than crashing the process on a
		// logging misconfiguration — logging must never be why the server fails to start.
		built = zap.NewNop()
	}

	return &Logger{built.Sugar()}
}
