package oauth

import (
	"context"
	"time"

	"golang.org/x/oauth2"
)

// MockUberClient stands in for Client while partner.accounts/partner.trips/
// partner.payments scope access is pending Uber's approval. Its default
// fixtures are the exact example payloads from
// .specs/features/api-endpoints/uber-integration.md (confirmed against
// Uber's official tutorial), so code written against MockUberClient
// exercises the real field shapes. Trips/Payments/Profile are exported so
// callers (tests, or a future admin tool) can override them.
type MockUberClient struct {
	Profile  *Profile
	Trips    []TripRecord
	Payments []PaymentRecord
}

// NewMockUberClient builds a MockUberClient with the confirmed example
// fixtures from uber-integration.md.
func NewMockUberClient() *MockUberClient {
	return &MockUberClient{
		Profile:  defaultMockProfile(),
		Trips:    []TripRecord{defaultMockTrip()},
		Payments: []PaymentRecord{defaultMockPayment()},
	}
}

// Exchange returns a static mock token; MockUberClient never contacts Uber.
func (m *MockUberClient) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	return &oauth2.Token{
		AccessToken:  "mock-access-token",
		RefreshToken: "mock-refresh-token",
		Expiry:       time.Now().Add(time.Hour),
	}, nil
}

// GetProfile returns m.Profile.
func (m *MockUberClient) GetProfile(ctx context.Context, token *oauth2.Token) (*Profile, error) {
	p := *m.Profile
	return &p, nil
}

// ListTrips returns up to limit of m.Trips.
func (m *MockUberClient) ListTrips(ctx context.Context, token *oauth2.Token, limit int) ([]TripRecord, error) {
	if limit >= 0 && limit < len(m.Trips) {
		return m.Trips[:limit], nil
	}
	return m.Trips, nil
}

// ListPayments returns up to limit of m.Payments.
func (m *MockUberClient) ListPayments(ctx context.Context, token *oauth2.Token, limit int) ([]PaymentRecord, error) {
	if limit >= 0 && limit < len(m.Payments) {
		return m.Payments[:limit], nil
	}
	return m.Payments, nil
}

func defaultMockProfile() *Profile {
	return &Profile{
		DriverID:          "8LvWuRAq2511gmr8EMkovekFNa2848ly",
		FirstName:         "Mock",
		LastName:          "Driver",
		Email:             "mock.driver@example.com",
		MobilePhoneNumber: "+15555550100",
		PictureURL:        "https://example.com/mock-driver.jpg",
		Rating:            4.8,
	}
}

// defaultMockTrip is the exact GET /v1/partners/trips example from
// uber-integration.md.
func defaultMockTrip() TripRecord {
	return TripRecord{
		TripID:          "b5613b6a-fe74-4704-a637-50f8d51a8bb1",
		DriverID:        "8LvWuRAq2511gmr8EMkovekFNa2848ly",
		Fare:            6.2,
		CurrencyCode:    "USD",
		DistanceMiles:   0.37,
		DurationSeconds: 475,
		Status:          "completed",
		SurgeMultiplier: 1,
		VehicleID:       "0082b54a-6a5e-4f6b-b999-b0649f286381",
		Pickup:          TripEventTime{Timestamp: 1502843903},
		Dropoff:         TripEventTime{Timestamp: 1502844378},
		StartCity: TripCity{
			Latitude:    38.3498,
			Longitude:   -81.6326,
			DisplayName: "Charleston, WV",
		},
		StatusChanges: []TripStatusChange{
			{Status: "accepted", Timestamp: 1502843899},
			{Status: "driver_arrived", Timestamp: 1502843900},
			{Status: "trip_began", Timestamp: 1502843903},
			{Status: "completed", Timestamp: 1502844378},
		},
	}
}

// defaultMockPayment is the exact GET /v1/partners/payments example from
// uber-integration.md.
func defaultMockPayment() PaymentRecord {
	return PaymentRecord{
		PaymentID:     "5cb8304c-f3f0-4a46-b6e3-b55e020750d7",
		TripID:        "5cb8304c-f3f0-4a46-b6e3-b55e020750d7",
		Category:      "fare",
		EventTime:     1502842757,
		CashCollected: 0,
		Amount:        3.12,
		CurrencyCode:  "USD",
		DriverID:      "8LvWuRAq2511gmr8EMkovekFNa2848ly",
		Breakdown: PaymentBreakdown{
			Other:      4.16,
			ServiceFee: -1.04,
		},
		PartnerID: "8LvWuRAq2511gmr8EMkovekFNa2848ly",
	}
}
