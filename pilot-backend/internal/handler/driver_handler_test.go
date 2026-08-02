package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"pilot-backend/internal/models"
)

func TestDriverHandler_Me_RequiresAuth(t *testing.T) {
	env := newTestEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/drivers/me", nil)
	rec := httptest.NewRecorder()
	env.engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestDriverHandler_Me_ReturnsProfile(t *testing.T) {
	env := newTestEnv(t)
	token, driver := env.login(t, "code-"+t.Name())

	req := httptest.NewRequest(http.MethodGet, "/api/drivers/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	env.engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var got models.Driver
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got.ID != driver.ID {
		t.Errorf("ID = %d, want %d", got.ID, driver.ID)
	}
}

func TestDriverHandler_UpdateMe(t *testing.T) {
	env := newTestEnv(t)
	token, _ := env.login(t, "code-"+t.Name())

	body, _ := json.Marshal(map[string]string{"phone": "+5511988888888"})
	req := httptest.NewRequest(http.MethodPut, "/api/drivers/me", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	env.engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var got models.Driver
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got.Phone != "+5511988888888" {
		t.Errorf("Phone = %q, want +5511988888888", got.Phone)
	}
}

func TestDriverHandler_UpdateMe_InvalidPhone(t *testing.T) {
	env := newTestEnv(t)
	token, _ := env.login(t, "code-"+t.Name())

	body, _ := json.Marshal(map[string]string{"phone": "not-a-phone"})
	req := httptest.NewRequest(http.MethodPut, "/api/drivers/me", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	env.engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body = %s", rec.Code, rec.Body.String())
	}
}

func TestDriverHandler_SyncProfile(t *testing.T) {
	env := newTestEnv(t)
	token, _ := env.login(t, "code-"+t.Name())

	req := httptest.NewRequest(http.MethodPost, "/api/drivers/sync-profile", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	env.engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var got struct {
		Success  bool   `json:"success"`
		SyncedAt string `json:"synced_at"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !got.Success || got.SyncedAt == "" {
		t.Errorf("body = %+v, want success=true and a synced_at timestamp", got)
	}
}
