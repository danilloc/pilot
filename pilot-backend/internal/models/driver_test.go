package models

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDriver_Validate(t *testing.T) {
	base := func() Driver {
		return Driver{Email: "driver@example.com", Phone: "+5511999999999", Rating: 4.8}
	}

	tests := []struct {
		name    string
		mutate  func(*Driver)
		wantErr error
	}{
		{"valid", func(d *Driver) {}, nil},
		{"invalid email", func(d *Driver) { d.Email = "not-an-email" }, ErrInvalidEmail},
		{"empty email", func(d *Driver) { d.Email = "" }, ErrInvalidEmail},
		{"invalid phone", func(d *Driver) { d.Phone = "011999999999" }, ErrInvalidPhone},
		{"empty phone allowed", func(d *Driver) { d.Phone = "" }, nil},
		{"rating too high", func(d *Driver) { d.Rating = 5.5 }, ErrInvalidRating},
		{"rating negative", func(d *Driver) { d.Rating = -1 }, ErrInvalidRating},
		{"banned", func(d *Driver) { d.IsBanned = true }, ErrDriverBanned},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := base()
			tt.mutate(&d)
			err := d.Validate()
			if err != tt.wantErr {
				t.Errorf("Validate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestDriver_Redact(t *testing.T) {
	d := Driver{
		OAuthToken:          "secret-token",
		OAuthRefreshToken:   "secret-refresh",
		FailedLoginAttempts: 3,
		LastLoginIP:         "1.2.3.4",
		BanReason:           "fraud",
	}
	d.Redact()

	if d.OAuthToken != "" || d.OAuthRefreshToken != "" || d.FailedLoginAttempts != 0 || d.LastLoginIP != "" || d.BanReason != "" {
		t.Errorf("Redact() left sensitive fields populated: %+v", d)
	}
}

func TestDriver_JSONMarshal_OmitsSensitiveFields(t *testing.T) {
	d := Driver{
		Name:       "João",
		Email:      "joao@example.com",
		OAuthToken: "should-not-appear",
		UberID:     "should-not-appear-either",
	}

	out, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	if strings.Contains(string(out), "should-not-appear") {
		t.Errorf("JSON output leaked a sensitive field: %s", out)
	}
	if !strings.Contains(string(out), `"email":"joao@example.com"`) {
		t.Errorf("JSON output missing expected email field: %s", out)
	}
}
