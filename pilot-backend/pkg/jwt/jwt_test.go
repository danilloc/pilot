package jwt

import (
	"testing"
	"time"
)

const testSecret = "test-secret-do-not-use-in-prod"

func TestGenerateAndValidateToken_Valid(t *testing.T) {
	mgr := NewManager(testSecret)

	token, err := mgr.GenerateToken(42, "driver@example.com", time.Hour)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	claims, err := mgr.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken() error = %v, want nil", err)
	}
	if claims.DriverID != 42 {
		t.Errorf("DriverID = %d, want 42", claims.DriverID)
	}
	if claims.Email != "driver@example.com" {
		t.Errorf("Email = %q, want driver@example.com", claims.Email)
	}
}

func TestValidateToken_Expired(t *testing.T) {
	mgr := NewManager(testSecret)

	token, err := mgr.GenerateToken(1, "a@b.com", -time.Minute)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	_, err = mgr.ValidateToken(token)
	if err != ErrTokenExpired {
		t.Fatalf("ValidateToken() error = %v, want ErrTokenExpired", err)
	}
}

func TestValidateToken_InvalidSignature(t *testing.T) {
	mgr := NewManager(testSecret)
	other := NewManager("a-different-secret")

	token, err := mgr.GenerateToken(1, "a@b.com", time.Hour)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	_, err = other.ValidateToken(token)
	if err != ErrInvalidToken {
		t.Fatalf("ValidateToken() error = %v, want ErrInvalidToken", err)
	}
}

func TestValidateToken_Malformed(t *testing.T) {
	mgr := NewManager(testSecret)

	_, err := mgr.ValidateToken("not.a.jwt")
	if err != ErrInvalidToken {
		t.Fatalf("ValidateToken() error = %v, want ErrInvalidToken", err)
	}
}
