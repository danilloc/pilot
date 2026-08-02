package models

import "testing"

func TestIsValidEmail(t *testing.T) {
	tests := map[string]bool{
		"driver@example.com":    true,
		"a.b+c@sub.example.com": true,
		"":                      false,
		"not-an-email":          false,
		"missing-domain@":       false,
		"@missing-local.com":    false,
		"Name <driver@ex.com>":  false,
		"driver@example":        true, // valid per RFC 5322 addr-spec, no TLD requirement
	}

	for email, want := range tests {
		if got := isValidEmail(email); got != want {
			t.Errorf("isValidEmail(%q) = %v, want %v", email, got, want)
		}
	}
}

func TestIsValidE164(t *testing.T) {
	tests := map[string]bool{
		"+5511999999999": true,
		"+1234567890":    true,
		"5511999999999":  false, // missing +
		"+0123456789":    false, // leading zero after +
		"+55 11999999":   false, // spaces not allowed
		"":               false,
	}

	for phone, want := range tests {
		if got := isValidE164(phone); got != want {
			t.Errorf("isValidE164(%q) = %v, want %v", phone, got, want)
		}
	}
}
