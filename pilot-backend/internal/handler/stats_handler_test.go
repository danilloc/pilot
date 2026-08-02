package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestStatsHandler_Today_RequiresAuth(t *testing.T) {
	env := newTestEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/stats/today", nil)
	rec := httptest.NewRecorder()
	env.engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestStatsHandler_Today_ReturnsComputedStats(t *testing.T) {
	env := newTestEnv(t)
	token, driver := env.login(t, "code-"+t.Name())

	env.uber.trips = nil // no sync needed; seed a trip directly instead
	seedHandlerTrip(t, env, driver.ID, "stats-today-"+t.Name(), time.Now().Add(-time.Hour), 42.5)

	req := httptest.NewRequest(http.MethodGet, "/api/stats/today", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	env.engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		TotalTrips  int     `json:"total_trips"`
		TotalEarned float64 `json:"total_earned"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if resp.TotalTrips != 1 || resp.TotalEarned != 42.5 {
		t.Errorf("resp = %+v, want TotalTrips=1 TotalEarned=42.5", resp)
	}
}

func TestStatsHandler_Week(t *testing.T) {
	env := newTestEnv(t)
	token, driver := env.login(t, "code-"+t.Name())
	seedHandlerTrip(t, env, driver.ID, "stats-week-"+t.Name(), time.Now().Add(-24*time.Hour), 30)

	req := httptest.NewRequest(http.MethodGet, "/api/stats/week", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	env.engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Days []struct {
			Trips int `json:"trips"`
		} `json:"days"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(resp.Days) == 0 {
		t.Error("expected at least one day in the weekly breakdown")
	}
}

func TestStatsHandler_Month_InvalidFormat(t *testing.T) {
	env := newTestEnv(t)
	token, _ := env.login(t, "code-"+t.Name())

	req := httptest.NewRequest(http.MethodGet, "/api/stats/month?month=garbage", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	env.engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body = %s", rec.Code, rec.Body.String())
	}
}

// seedHandlerTrip inserts a COMPLETED trip directly via the shared test
// engine's driver repo connection, for stats tests that don't need a full
// sync round trip.
func seedHandlerTrip(t *testing.T, env *testEnv, driverID int64, uberTripID string, endedAt time.Time, fare float64) {
	t.Helper()
	if err := env.gormDB.Exec(
		"INSERT INTO trips (uuid, driver_id, uber_trip_id, started_at, ended_at, distance_km, fare_value, status) VALUES (UUID(), ?, ?, ?, ?, 5, ?, 'COMPLETED')",
		driverID, uberTripID, endedAt.Add(-20*time.Minute), endedAt, fare,
	).Error; err != nil {
		t.Fatalf("seed trip error = %v", err)
	}
}
