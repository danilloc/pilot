package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad_GinModeDefaultsFollowEnvironment(t *testing.T) {
	tests := []struct {
		environment string
		wantGinMode string
	}{
		{"development", "debug"},
		{"staging", "release"},
		{"production", "release"},
	}

	for _, tt := range tests {
		t.Run(tt.environment, func(t *testing.T) {
			os.Unsetenv("GIN_MODE")
			t.Setenv("ENVIRONMENT", tt.environment)

			cfg := Load()
			if cfg.GinMode != tt.wantGinMode {
				t.Errorf("GinMode = %q, want %q for ENVIRONMENT=%s", cfg.GinMode, tt.wantGinMode, tt.environment)
			}
		})
	}
}

func TestLoad_GinModeExplicitOverridesDefault(t *testing.T) {
	t.Setenv("ENVIRONMENT", "production")
	t.Setenv("GIN_MODE", "debug")

	cfg := Load()
	if cfg.GinMode != "debug" {
		t.Errorf("GinMode = %q, want explicit GIN_MODE=debug to win over the production default", cfg.GinMode)
	}
}

func TestGetEnvBool(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{"unset uses default", "", true},
		{"valid true", "true", true},
		{"valid false", "false", false},
		{"invalid falls back to default", "not-a-bool", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value == "" {
				os.Unsetenv("TEST_BOOL_VAR")
			} else {
				t.Setenv("TEST_BOOL_VAR", tt.value)
			}
			if got := getEnvBool("TEST_BOOL_VAR", true); got != tt.want {
				t.Errorf("getEnvBool() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetEnvDuration(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{"unset uses default", "", "1h0m0s"},
		{"valid duration", "30m", "30m0s"},
		{"invalid falls back to default", "not-a-duration", "1h0m0s"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value == "" {
				os.Unsetenv("TEST_DURATION_VAR")
			} else {
				t.Setenv("TEST_DURATION_VAR", tt.value)
			}
			if got := getEnvDuration("TEST_DURATION_VAR", time.Hour); got.String() != tt.want {
				t.Errorf("getEnvDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetEnvInt_InvalidFallsBackToDefault(t *testing.T) {
	t.Setenv("TEST_INT_VAR", "not-an-int")
	if got := getEnvInt("TEST_INT_VAR", 42); got != 42 {
		t.Errorf("getEnvInt() = %d, want 42 (default, invalid value ignored)", got)
	}
}

func TestGetEnvList(t *testing.T) {
	t.Setenv("TEST_LIST_VAR", "a, b ,, c")
	got := getEnvList("TEST_LIST_VAR", nil)
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("getEnvList() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("getEnvList()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
