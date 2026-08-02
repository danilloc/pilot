package logger

import (
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// buildObserved mirrors New()'s config (level + stacktrace policy) but
// swaps zap's real output core for an in-memory observer, so the test can
// inspect exactly what a real logger.New() build would have written.
func buildObserved(t *testing.T, environment string) (*Logger, *observer.ObservedLogs) {
	t.Helper()

	core, logs := observer.New(zapcore.ErrorLevel)

	var opts []zap.Option
	if environment != "development" {
		opts = append(opts, zap.AddStacktrace(zapcore.InvalidLevel))
	} else {
		opts = append(opts, zap.AddStacktrace(zapcore.ErrorLevel))
	}

	zl := zap.New(core, opts...)
	return &Logger{zl.Sugar()}, logs
}

func TestNew_StacktraceOnlyInDevelopment(t *testing.T) {
	tests := []struct {
		environment string
		wantStack   bool
	}{
		{"development", true},
		{"production", false},
		{"staging", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.environment, func(t *testing.T) {
			log, logs := buildObserved(t, tt.environment)
			log.Errorw("something_failed", "detail", "x")

			entries := logs.All()
			if len(entries) != 1 {
				t.Fatalf("len(entries) = %d, want 1", len(entries))
			}

			hasStack := strings.TrimSpace(entries[0].Entry.Stack) != ""
			if hasStack != tt.wantStack {
				t.Errorf("environment=%q: stacktrace present = %v, want %v", tt.environment, hasStack, tt.wantStack)
			}
		})
	}
}

// TestNew_BuildsWithoutError is a smoke test that the real New() —
// production encoder, JSON output, level parsing — doesn't error for any
// environment value, since buildObserved above only mirrors its config and
// doesn't exercise New() itself.
func TestNew_BuildsWithoutError(t *testing.T) {
	for _, env := range []string{"development", "production", "staging", ""} {
		log := New("info", env)
		if log == nil || log.SugaredLogger == nil {
			t.Errorf("New(%q) returned a nil logger", env)
		}
	}
}
