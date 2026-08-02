package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGoalHandler_Create_RequiresAuth(t *testing.T) {
	env := newTestEnv(t)

	req := httptest.NewRequest(http.MethodPost, "/api/goals", bytes.NewReader([]byte(`{}`)))
	rec := httptest.NewRecorder()
	env.engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestGoalHandler_CreateThenGet(t *testing.T) {
	env := newTestEnv(t)
	token, _ := env.login(t, "code-"+t.Name())

	body, _ := json.Marshal(map[string]interface{}{"goal_date": "2026-06-01", "goal_amount": 350.00})
	createReq := httptest.NewRequest(http.MethodPost, "/api/goals", bytes.NewReader(body))
	createReq.Header.Set("Authorization", "Bearer "+token)
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	env.engine.ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", createRec.Code, createRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/goals?date=2026-06-01", nil)
	getReq.Header.Set("Authorization", "Bearer "+token)
	getRec := httptest.NewRecorder()
	env.engine.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("get status = %d, body = %s", getRec.Code, getRec.Body.String())
	}
	var resp struct {
		GoalAmount   float64 `json:"goal_amount"`
		ActualAmount float64 `json:"actual_amount"`
	}
	if err := json.Unmarshal(getRec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if resp.GoalAmount != 350.00 {
		t.Errorf("GoalAmount = %v, want 350.00", resp.GoalAmount)
	}
	if resp.ActualAmount != 0 {
		t.Errorf("ActualAmount = %v, want 0 (no trips yet)", resp.ActualAmount)
	}
}

func TestGoalHandler_Progress(t *testing.T) {
	env := newTestEnv(t)
	token, _ := env.login(t, "code-"+t.Name())

	body, _ := json.Marshal(map[string]interface{}{"goal_date": time.Now().Format("2006-01-02"), "goal_amount": 100.00})
	createReq := httptest.NewRequest(http.MethodPost, "/api/goals", bytes.NewReader(body))
	createReq.Header.Set("Authorization", "Bearer "+token)
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	env.engine.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", createRec.Code, createRec.Body.String())
	}

	req := httptest.NewRequest(http.MethodGet, "/api/goals/progress", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	env.engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		GoalAmount float64 `json:"goal_amount"`
		Remaining  float64 `json:"remaining"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if resp.GoalAmount != 100.00 || resp.Remaining != 100.00 {
		t.Errorf("resp = %+v, want GoalAmount=100 Remaining=100 (no trips synced yet)", resp)
	}
}

func TestGoalHandler_UpdateAndDelete(t *testing.T) {
	env := newTestEnv(t)
	token, _ := env.login(t, "code-"+t.Name())

	body, _ := json.Marshal(map[string]interface{}{"goal_date": "2026-07-01", "goal_amount": 100.00})
	createReq := httptest.NewRequest(http.MethodPost, "/api/goals", bytes.NewReader(body))
	createReq.Header.Set("Authorization", "Bearer "+token)
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	env.engine.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", createRec.Code, createRec.Body.String())
	}
	var created struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	updateBody, _ := json.Marshal(map[string]interface{}{"goal_amount": 400.00})
	updateReq := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/goals/%d", created.ID), bytes.NewReader(updateBody))
	updateReq.Header.Set("Authorization", "Bearer "+token)
	updateReq.Header.Set("Content-Type", "application/json")
	updateRec := httptest.NewRecorder()
	env.engine.ServeHTTP(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("update status = %d, body = %s", updateRec.Code, updateRec.Body.String())
	}
	var updated struct {
		GoalAmount float64 `json:"goal_amount"`
	}
	if err := json.Unmarshal(updateRec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if updated.GoalAmount != 400.00 {
		t.Errorf("GoalAmount = %v, want 400.00", updated.GoalAmount)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/goals/%d", created.ID), nil)
	deleteReq.Header.Set("Authorization", "Bearer "+token)
	deleteRec := httptest.NewRecorder()
	env.engine.ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("delete status = %d, body = %s", deleteRec.Code, deleteRec.Body.String())
	}
	var deleted struct {
		Success bool   `json:"success"`
		Status  string `json:"status"`
	}
	if err := json.Unmarshal(deleteRec.Body.Bytes(), &deleted); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !deleted.Success || deleted.Status != "ABANDONED" {
		t.Errorf("delete resp = %+v, want success=true status=ABANDONED", deleted)
	}
}
