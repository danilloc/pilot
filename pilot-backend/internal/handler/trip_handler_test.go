package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"pilot-backend/internal/oauth"
)

func TestTripHandler_List_RequiresAuth(t *testing.T) {
	env := newTestEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/trips", nil)
	rec := httptest.NewRecorder()
	env.engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestTripHandler_SyncThenList(t *testing.T) {
	env := newTestEnv(t)
	token, driver := env.login(t, "code-"+t.Name())

	now := time.Now()
	env.uber.trips = []oauth.TripRecord{
		{
			TripID: "handler-" + t.Name(), Status: "completed",
			Pickup: oauth.TripEventTime{Timestamp: now.Add(-time.Hour).Unix()}, Dropoff: oauth.TripEventTime{Timestamp: now.Unix()},
			DistanceMiles: 12, Fare: 45, CurrencyCode: "BRL",
			StartCity: oauth.TripCity{DisplayName: "São Paulo"},
		},
	}

	syncReq := httptest.NewRequest(http.MethodPost, "/api/trips/sync", nil)
	syncReq.Header.Set("Authorization", "Bearer "+token)
	syncRec := httptest.NewRecorder()
	env.engine.ServeHTTP(syncRec, syncReq)

	if syncRec.Code != http.StatusOK {
		t.Fatalf("sync status = %d, body = %s", syncRec.Code, syncRec.Body.String())
	}
	var syncResp struct {
		Success     bool   `json:"success"`
		SyncedCount int    `json:"synced_count"`
		SyncedAt    string `json:"synced_at"`
	}
	if err := json.Unmarshal(syncRec.Body.Bytes(), &syncResp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !syncResp.Success || syncResp.SyncedCount != 1 {
		t.Fatalf("sync body = %+v, want success=true, synced_count=1", syncResp)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/trips?start_date="+now.AddDate(0, 0, -1).Format(dateLayout)+"&end_date="+now.AddDate(0, 0, 1).Format(dateLayout), nil)
	listReq.Header.Set("Authorization", "Bearer "+token)
	listRec := httptest.NewRecorder()
	env.engine.ServeHTTP(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", listRec.Code, listRec.Body.String())
	}
	var listResp struct {
		Data []struct {
			ID       int64   `json:"id"`
			DriverID int64   `json:"driver_id"`
			City     string  `json:"city"`
			FareVal  float64 `json:"fare_value"`
		} `json:"data"`
		Pagination struct {
			Total int64 `json:"total"`
		} `json:"pagination"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if listResp.Pagination.Total != 1 || len(listResp.Data) != 1 {
		t.Fatalf("list body = %+v, want 1 trip", listResp)
	}
	if listResp.Data[0].DriverID != driver.ID {
		t.Errorf("DriverID = %d, want %d", listResp.Data[0].DriverID, driver.ID)
	}
}

func TestTripHandler_Get_NeverLeaksCoordinates(t *testing.T) {
	env := newTestEnv(t)
	token, _ := env.login(t, "code-"+t.Name())

	now := time.Now()
	env.uber.trips = []oauth.TripRecord{
		{
			TripID: "coords-" + t.Name(), Status: "completed",
			Pickup: oauth.TripEventTime{Timestamp: now.Add(-time.Hour).Unix()}, Dropoff: oauth.TripEventTime{Timestamp: now.Unix()},
			DistanceMiles: 3, Fare: 12,
		},
	}
	syncReq := httptest.NewRequest(http.MethodPost, "/api/trips/sync", nil)
	syncReq.Header.Set("Authorization", "Bearer "+token)
	syncRec := httptest.NewRecorder()
	env.engine.ServeHTTP(syncRec, syncReq)
	if syncRec.Code != http.StatusOK {
		t.Fatalf("sync status = %d", syncRec.Code)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/trips", nil)
	listReq.Header.Set("Authorization", "Bearer "+token)
	listRec := httptest.NewRecorder()
	env.engine.ServeHTTP(listRec, listReq)

	var listResp struct {
		Data []struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(listResp.Data) != 1 {
		t.Fatalf("expected 1 trip, got %d", len(listResp.Data))
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/trips/"+strconv.FormatInt(listResp.Data[0].ID, 10), nil)
	getReq.Header.Set("Authorization", "Bearer "+token)
	getRec := httptest.NewRecorder()
	env.engine.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("get status = %d, body = %s", getRec.Code, getRec.Body.String())
	}
	for _, field := range []string{"start_latitude", "start_longitude", "end_latitude", "end_longitude"} {
		if bytes.Contains(getRec.Body.Bytes(), []byte(field)) {
			t.Errorf("GET /trips/:id leaked coordinate field %q", field)
		}
	}
}

func TestTripHandler_Get_NotFound(t *testing.T) {
	env := newTestEnv(t)
	token, _ := env.login(t, "code-"+t.Name())

	req := httptest.NewRequest(http.MethodGet, "/api/trips/999999999", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	env.engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestTripHandler_List_InvalidDateFormat(t *testing.T) {
	env := newTestEnv(t)
	token, _ := env.login(t, "code-"+t.Name())

	req := httptest.NewRequest(http.MethodGet, "/api/trips?start_date=garbage", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	env.engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body = %s", rec.Code, rec.Body.String())
	}
}

func TestTripHandler_Get_NonNumericID(t *testing.T) {
	env := newTestEnv(t)
	token, _ := env.login(t, "code-"+t.Name())

	req := httptest.NewRequest(http.MethodGet, "/api/trips/not-a-number", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	env.engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404, body = %s", rec.Code, rec.Body.String())
	}
}

func TestTripHandler_Sync_InvalidBody(t *testing.T) {
	env := newTestEnv(t)
	token, _ := env.login(t, "code-"+t.Name())

	req := httptest.NewRequest(http.MethodPost, "/api/trips/sync", bytes.NewReader([]byte(`not-json`)))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	env.engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body = %s", rec.Code, rec.Body.String())
	}
}

func TestTripHandler_Sync_DriverNotFound(t *testing.T) {
	env := newTestEnv(t)

	req := httptest.NewRequest(http.MethodPost, "/api/trips/sync", nil)
	req.Header.Set("Authorization", "Bearer "+ghostToken(t, env))
	rec := httptest.NewRecorder()
	env.engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404, body = %s", rec.Code, rec.Body.String())
	}
}

func TestTripHandler_List_DriverNotFound(t *testing.T) {
	env := newTestEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/trips", nil)
	req.Header.Set("Authorization", "Bearer "+ghostToken(t, env))
	rec := httptest.NewRecorder()
	env.engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK && rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 200 (empty list) or 404, body = %s", rec.Code, rec.Body.String())
	}
}
