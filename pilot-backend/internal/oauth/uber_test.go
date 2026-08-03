package oauth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"golang.org/x/oauth2"
)

func testToken() *oauth2.Token {
	return &oauth2.Token{AccessToken: "test-access-token"}
}

func TestClient_Exchange_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm() error = %v", err)
		}
		if r.Form.Get("code") != "auth-code-123" {
			t.Errorf("code = %q, want auth-code-123", r.Form.Get("code"))
		}
		if r.Form.Get("grant_type") != "authorization_code" {
			t.Errorf("grant_type = %q, want authorization_code", r.Form.Get("grant_type"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token":  "issued-access-token",
			"refresh_token": "issued-refresh-token",
			"token_type":    "bearer",
			"expires_in":    3600,
		})
	}))
	defer srv.Close()

	c := NewClient("id", "secret", "redirect")
	c.cfg.Endpoint.TokenURL = srv.URL

	token, err := c.Exchange(t.Context(), "auth-code-123")
	if err != nil {
		t.Fatalf("Exchange() error = %v", err)
	}
	if token.AccessToken != "issued-access-token" {
		t.Errorf("AccessToken = %q, want issued-access-token", token.AccessToken)
	}
	if token.RefreshToken != "issued-refresh-token" {
		t.Errorf("RefreshToken = %q, want issued-refresh-token", token.RefreshToken)
	}
}

func TestClient_Exchange_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid_grant"})
	}))
	defer srv.Close()

	c := NewClient("id", "secret", "redirect")
	c.cfg.Endpoint.TokenURL = srv.URL

	_, err := c.Exchange(t.Context(), "bad-code")
	if err == nil {
		t.Fatal("expected an error for a rejected code, got nil")
	}
}

func TestClient_GetProfile_Success(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		json.NewEncoder(w).Encode(Profile{DriverID: "d1", FirstName: "Ana", Rating: 4.9})
	}))
	defer srv.Close()

	c := NewClient("id", "secret", "redirect")
	c.profileURL = srv.URL

	profile, err := c.GetProfile(t.Context(), testToken())
	if err != nil {
		t.Fatalf("GetProfile() error = %v", err)
	}
	if profile.DriverID != "d1" || profile.FirstName != "Ana" {
		t.Errorf("profile = %+v, want DriverID=d1 FirstName=Ana", profile)
	}
	if gotAuth != "Bearer test-access-token" {
		t.Errorf("Authorization header = %q, want %q", gotAuth, "Bearer test-access-token")
	}
}

func TestClient_GetProfile_RetriesOnServerErrorThenSucceeds(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(Profile{DriverID: "retried"})
	}))
	defer srv.Close()

	c := NewClient("id", "secret", "redirect")
	c.profileURL = srv.URL

	profile, err := c.GetProfile(t.Context(), testToken())
	if err != nil {
		t.Fatalf("GetProfile() error = %v", err)
	}
	if profile.DriverID != "retried" {
		t.Errorf("DriverID = %q, want retried", profile.DriverID)
	}
	if attempts != 2 {
		t.Errorf("attempts = %d, want 2 (one failure, one retry)", attempts)
	}
}

func TestClient_GetProfile_FailsAfterMaxAttempts(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewClient("id", "secret", "redirect")
	c.profileURL = srv.URL

	_, err := c.GetProfile(t.Context(), testToken())
	if err == nil {
		t.Fatal("expected an error after exhausting retries, got nil")
	}
	if attempts != maxAttempts {
		t.Errorf("attempts = %d, want %d (maxAttempts)", attempts, maxAttempts)
	}
}

// pagedTripsServer serves total canned trips, capping each response to
// perPage regardless of the client's requested limit — forcing the client
// to make multiple round trips to collect everything, exactly like a real
// paginated API that doesn't honor an overly large limit.
func pagedTripsServer(t *testing.T, total, perPage int) (*httptest.Server, *int) {
	t.Helper()
	requestCount := 0

	all := make([]TripRecord, total)
	for i := range all {
		all[i] = TripRecord{TripID: fmt.Sprintf("trip-%d", i)}
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		// A real paginated API honors the client's requested page size
		// (capped at its own perPage ceiling) — it must not hand back more
		// than what was asked for.
		reqLimit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		pageCap := perPage
		if reqLimit > 0 && reqLimit < pageCap {
			pageCap = reqLimit
		}

		end := offset + pageCap
		if end > len(all) {
			end = len(all)
		}
		var page []TripRecord
		if offset < len(all) {
			page = all[offset:end]
		}

		json.NewEncoder(w).Encode(tripsResponse{Count: total, Offset: offset, Trips: page})
	}))
	return srv, &requestCount
}

func TestClient_ListTrips_PaginatesUntilLimitReached(t *testing.T) {
	srv, requestCount := pagedTripsServer(t, 5, 2)
	defer srv.Close()

	c := NewClient("id", "secret", "redirect")
	c.tripsURL = srv.URL

	trips, err := c.ListTrips(t.Context(), testToken(), 5)
	if err != nil {
		t.Fatalf("ListTrips() error = %v", err)
	}
	if len(trips) != 5 {
		t.Fatalf("len(trips) = %d, want 5", len(trips))
	}
	// 2 per page server-side cap, 5 total => offsets 0,2,4 => 3 requests.
	if *requestCount != 3 {
		t.Errorf("requestCount = %d, want 3 (paginated in pages of 2)", *requestCount)
	}
	for i, trip := range trips {
		want := fmt.Sprintf("trip-%d", i)
		if trip.TripID != want {
			t.Errorf("trips[%d].TripID = %q, want %q (results out of order or duplicated)", i, trip.TripID, want)
		}
	}
}

func TestClient_ListTrips_StopsAtRequestedLimit(t *testing.T) {
	// Server has far more than the client asks for; ListTrips must not
	// over-fetch once it has collected `limit` trips.
	srv, requestCount := pagedTripsServer(t, 100, 50)
	defer srv.Close()

	c := NewClient("id", "secret", "redirect")
	c.tripsURL = srv.URL

	trips, err := c.ListTrips(t.Context(), testToken(), 10)
	if err != nil {
		t.Fatalf("ListTrips() error = %v", err)
	}
	if len(trips) != 10 {
		t.Fatalf("len(trips) = %d, want 10 (the requested limit, not the server's full 100)", len(trips))
	}
	if *requestCount != 1 {
		t.Errorf("requestCount = %d, want 1 (10 fits in a single page)", *requestCount)
	}
}

func TestClient_ListPayments_EmptyResponseIsNotAnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(paymentsResponse{Count: 0, Payments: nil})
	}))
	defer srv.Close()

	c := NewClient("id", "secret", "redirect")
	c.paymentsURL = srv.URL

	payments, err := c.ListPayments(t.Context(), testToken(), 50)
	if err != nil {
		t.Fatalf("ListPayments() error = %v, want nil (empty is valid — e.g. fleet-managed driver)", err)
	}
	if len(payments) != 0 {
		t.Errorf("len(payments) = %d, want 0", len(payments))
	}
}

func TestClient_ListPayments_ParsesConfirmedFieldShape(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(paymentsResponse{
			Count: 1,
			Payments: []PaymentRecord{
				{
					PaymentID: "pay-1", TripID: "trip-1", Category: "fare",
					EventTime: 1502842757, Amount: 3.12, CurrencyCode: "USD",
					Breakdown: PaymentBreakdown{Other: 4.16, ServiceFee: -1.04},
				},
			},
		})
	}))
	defer srv.Close()

	c := NewClient("id", "secret", "redirect")
	c.paymentsURL = srv.URL

	payments, err := c.ListPayments(t.Context(), testToken(), 50)
	if err != nil {
		t.Fatalf("ListPayments() error = %v", err)
	}
	if len(payments) != 1 {
		t.Fatalf("len(payments) = %d, want 1", len(payments))
	}
	if payments[0].Amount != 3.12 || payments[0].Breakdown.ServiceFee != -1.04 {
		t.Errorf("payments[0] = %+v, want Amount=3.12 Breakdown.ServiceFee=-1.04", payments[0])
	}
}

func TestTripRecord_KM_ConvertsMilesToKilometers(t *testing.T) {
	tr := TripRecord{DistanceMiles: 10}
	got := tr.KM()
	want := 16.0934
	if diff := got - want; diff > 0.0001 || diff < -0.0001 {
		t.Errorf("KM() = %v, want %v", got, want)
	}
}
