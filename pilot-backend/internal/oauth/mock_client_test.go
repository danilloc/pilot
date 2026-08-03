package oauth

import (
	"testing"

	"golang.org/x/oauth2"
)

func TestNewMockUberClient_DefaultFixtures(t *testing.T) {
	m := NewMockUberClient()
	if m.Profile == nil {
		t.Fatal("Profile is nil")
	}
	if len(m.Trips) != 1 {
		t.Fatalf("len(Trips) = %d, want 1", len(m.Trips))
	}
	if len(m.Payments) != 1 {
		t.Fatalf("len(Payments) = %d, want 1", len(m.Payments))
	}

	// The fixtures must be the exact confirmed example payloads from
	// uber-integration.md, not placeholder values — pin the fields that
	// matter to our field mapping (trip_service.go / payment_service.go).
	trip := m.Trips[0]
	if trip.TripID != "b5613b6a-fe74-4704-a637-50f8d51a8bb1" {
		t.Errorf("Trips[0].TripID = %q, want the documented example ID", trip.TripID)
	}
	if trip.Status != "completed" {
		t.Errorf("Trips[0].Status = %q, want completed", trip.Status)
	}
	if trip.DistanceMiles != 0.37 {
		t.Errorf("Trips[0].DistanceMiles = %v, want 0.37", trip.DistanceMiles)
	}
	if trip.Pickup.Timestamp != 1502843903 || trip.Dropoff.Timestamp != 1502844378 {
		t.Errorf("Trips[0] pickup/dropoff = %d/%d, want 1502843903/1502844378", trip.Pickup.Timestamp, trip.Dropoff.Timestamp)
	}

	payment := m.Payments[0]
	if payment.PaymentID != "5cb8304c-f3f0-4a46-b6e3-b55e020750d7" {
		t.Errorf("Payments[0].PaymentID = %q, want the documented example ID", payment.PaymentID)
	}
	if payment.Amount != 3.12 {
		t.Errorf("Payments[0].Amount = %v, want 3.12", payment.Amount)
	}
	if payment.Breakdown.ServiceFee != -1.04 {
		t.Errorf("Payments[0].Breakdown.ServiceFee = %v, want -1.04 (the documented example is negative)", payment.Breakdown.ServiceFee)
	}
}

func TestMockUberClient_Exchange(t *testing.T) {
	m := NewMockUberClient()
	token, err := m.Exchange(t.Context(), "any-code")
	if err != nil {
		t.Fatalf("Exchange() error = %v", err)
	}
	if token.AccessToken == "" {
		t.Error("AccessToken is empty")
	}
	if token.Expiry.IsZero() {
		t.Error("Expiry is zero")
	}
}

func TestMockUberClient_GetProfile_ReturnsACopy(t *testing.T) {
	m := NewMockUberClient()
	token, _ := m.Exchange(t.Context(), "code")

	p1, err := m.GetProfile(t.Context(), token)
	if err != nil {
		t.Fatalf("GetProfile() error = %v", err)
	}
	// Mutating the returned profile must not affect m.Profile or future
	// calls — GetProfile is documented to return a copy.
	p1.FirstName = "Mutated"

	p2, err := m.GetProfile(t.Context(), token)
	if err != nil {
		t.Fatalf("GetProfile() error = %v", err)
	}
	if p2.FirstName == "Mutated" {
		t.Error("mutating a returned Profile leaked into the mock's internal state")
	}
}

func TestMockUberClient_ListTrips_RespectsLimit(t *testing.T) {
	m := NewMockUberClient()
	m.Trips = []TripRecord{{TripID: "a"}, {TripID: "b"}, {TripID: "c"}}

	got, err := m.ListTrips(t.Context(), &oauth2.Token{}, 2)
	if err != nil {
		t.Fatalf("ListTrips() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}

	all, err := m.ListTrips(t.Context(), &oauth2.Token{}, 10)
	if err != nil {
		t.Fatalf("ListTrips() error = %v", err)
	}
	if len(all) != 3 {
		t.Errorf("len(all) = %d, want 3 (limit larger than available)", len(all))
	}
}

func TestMockUberClient_ListPayments_RespectsLimit(t *testing.T) {
	m := NewMockUberClient()
	m.Payments = []PaymentRecord{{PaymentID: "a"}, {PaymentID: "b"}}

	got, err := m.ListPayments(t.Context(), &oauth2.Token{}, 1)
	if err != nil {
		t.Fatalf("ListPayments() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
}

func TestMockUberClient_ListPayments_EmptyIsValid(t *testing.T) {
	m := NewMockUberClient()
	m.Payments = nil

	got, err := m.ListPayments(t.Context(), &oauth2.Token{}, 50)
	if err != nil {
		t.Fatalf("ListPayments() error = %v, want nil (empty is a valid fleet-manager response)", err)
	}
	if len(got) != 0 {
		t.Errorf("len(got) = %d, want 0", len(got))
	}
}

func TestNewUberClient_MockVsReal(t *testing.T) {
	mock := NewUberClient(true, "id", "secret", "redirect")
	if _, ok := mock.(*MockUberClient); !ok {
		t.Errorf("NewUberClient(true, ...) = %T, want *MockUberClient", mock)
	}

	real := NewUberClient(false, "id", "secret", "redirect")
	if _, ok := real.(*Client); !ok {
		t.Errorf("NewUberClient(false, ...) = %T, want *Client", real)
	}
}
