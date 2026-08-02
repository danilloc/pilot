package config

import (
	"os"
	"testing"
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
