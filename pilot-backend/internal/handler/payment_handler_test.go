package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"pilot-backend/internal/oauth"
)

func TestPaymentHandler_List_RequiresAuth(t *testing.T) {
	env := newTestEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/payments", nil)
	rec := httptest.NewRecorder()
	env.engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestPaymentHandler_SyncThenList(t *testing.T) {
	env := newTestEnv(t)
	token, _ := env.login(t, "code-"+t.Name())

	env.uber.payments = []oauth.PaymentRecord{
		{PaymentID: "handler-pay-" + t.Name(), Amount: 55.75, CurrencyCode: "BRL", Category: "fare", EventTime: time.Now().Unix()},
	}

	syncReq := httptest.NewRequest(http.MethodPost, "/api/payments/sync", nil)
	syncReq.Header.Set("Authorization", "Bearer "+token)
	syncRec := httptest.NewRecorder()
	env.engine.ServeHTTP(syncRec, syncReq)

	if syncRec.Code != http.StatusOK {
		t.Fatalf("sync status = %d, body = %s", syncRec.Code, syncRec.Body.String())
	}
	var syncResp struct {
		Success     bool `json:"success"`
		SyncedCount int  `json:"synced_count"`
	}
	if err := json.Unmarshal(syncRec.Body.Bytes(), &syncResp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !syncResp.Success || syncResp.SyncedCount != 1 {
		t.Fatalf("sync body = %+v, want success=true synced_count=1", syncResp)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/payments", nil)
	listReq.Header.Set("Authorization", "Bearer "+token)
	listRec := httptest.NewRecorder()
	env.engine.ServeHTTP(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", listRec.Code, listRec.Body.String())
	}
	var listResp struct {
		Data []struct {
			Amount float64 `json:"amount"`
		} `json:"data"`
		Pagination struct {
			Total int64 `json:"total"`
		} `json:"pagination"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if listResp.Pagination.Total != 1 || len(listResp.Data) != 1 || listResp.Data[0].Amount != 55.75 {
		t.Fatalf("list resp = %+v, want 1 payment with amount=55.75", listResp)
	}
}

func TestPaymentHandler_Sync_EmptyResultSucceeds(t *testing.T) {
	env := newTestEnv(t)
	token, _ := env.login(t, "code-"+t.Name())
	env.uber.payments = nil

	req := httptest.NewRequest(http.MethodPost, "/api/payments/sync", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	env.engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s (empty Uber response must not be an error — e.g. fleet-managed driver)", rec.Code, rec.Body.String())
	}
	var resp struct {
		Success     bool `json:"success"`
		SyncedCount int  `json:"synced_count"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !resp.Success || resp.SyncedCount != 0 {
		t.Errorf("resp = %+v, want success=true synced_count=0", resp)
	}
}
