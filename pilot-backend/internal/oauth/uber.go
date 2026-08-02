// Package oauth implements the OAuth2 authorization-code flow against
// Uber's driver API: exchanging a login code for tokens, then fetching the
// authenticated driver's profile, trips, and payments. Field shapes below
// match .specs/features/api-endpoints/uber-integration.md, confirmed
// against Uber's official tutorial (2026-08-02).
package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"golang.org/x/oauth2"
)

const (
	uberAuthURL     = "https://login.uber.com/oauth/v2/authorize"
	uberTokenURL    = "https://login.uber.com/oauth/v2/token"
	uberProfileURL  = "https://api.uber.com/v1/partners/me"
	uberTripsURL    = "https://api.uber.com/v1/partners/trips"
	uberPaymentsURL = "https://api.uber.com/v1/partners/payments"

	requestTimeout = 10 * time.Second
	maxAttempts    = 2

	// uberMaxPageSize is the Uber API's documented per-request cap; larger
	// result sets must be paginated with offset.
	uberMaxPageSize = 50
)

// milesToKM converts Uber's mile-denominated trip distance to kilometers.
const milesToKM = 1.60934

// Profile is the subset of Uber's partner profile response the app cares
// about.
type Profile struct {
	DriverID          string  `json:"driver_id"`
	FirstName         string  `json:"first_name"`
	LastName          string  `json:"last_name"`
	Email             string  `json:"email"`
	MobilePhoneNumber string  `json:"mobile_phone_number"`
	PictureURL        string  `json:"picture_url"`
	Rating            float64 `json:"rating"`
}

// TripEventTime is the {"timestamp": ...} shape shared by pickup/dropoff.
type TripEventTime struct {
	Timestamp int64 `json:"timestamp"`
}

// TripCity is the pickup location shape (only start_city is documented —
// no destination city exists in the payload).
type TripCity struct {
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	DisplayName string  `json:"display_name"`
}

// TripStatusChange is one entry in a trip's status history.
type TripStatusChange struct {
	Status    string `json:"status"`
	Timestamp int64  `json:"timestamp"`
}

// TripRecord is Uber's GET /v1/partners/trips item shape.
type TripRecord struct {
	TripID          string             `json:"trip_id"`
	DriverID        string             `json:"driver_id"`
	Fare            float64            `json:"fare"`
	CurrencyCode    string             `json:"currency_code"`
	DistanceMiles   float64            `json:"distance"`
	DurationSeconds int64              `json:"duration"`
	Status          string             `json:"status"`
	SurgeMultiplier float64            `json:"surge_multiplier"`
	VehicleID       string             `json:"vehicle_id"`
	Pickup          TripEventTime      `json:"pickup"`
	Dropoff         TripEventTime      `json:"dropoff"`
	StartCity       TripCity           `json:"start_city"`
	StatusChanges   []TripStatusChange `json:"status_changes"`
}

// KM returns the trip's distance converted from Uber's miles to
// kilometers, matching trips.distance_km's unit.
func (t TripRecord) KM() float64 { return t.DistanceMiles * milesToKM }

type tripsResponse struct {
	Count  int          `json:"count"`
	Limit  int          `json:"limit"`
	Offset int          `json:"offset"`
	Trips  []TripRecord `json:"trips"`
}

// PaymentBreakdown is the partial fare breakdown Uber attaches to a
// payment. Fields are optional per-payment — e.g. Toll only appears when
// the trip actually incurred one.
type PaymentBreakdown struct {
	Other       float64 `json:"other"`
	ServiceFee  float64 `json:"service_fee"`
	Toll        float64 `json:"toll"`
}

// PaymentRecord is Uber's GET /v1/partners/payments item shape.
type PaymentRecord struct {
	PaymentID     string           `json:"payment_id"`
	TripID        string           `json:"trip_id"`
	Category      string           `json:"category"`
	EventTime     int64            `json:"event_time"`
	CashCollected float64          `json:"cash_collected"`
	Amount        float64          `json:"amount"`
	CurrencyCode  string           `json:"currency_code"`
	DriverID      string           `json:"driver_id"`
	Breakdown     PaymentBreakdown `json:"breakdown"`
	PartnerID     string           `json:"partner_id"`
}

type paymentsResponse struct {
	Count    int             `json:"count"`
	Limit    int             `json:"limit"`
	Offset   int             `json:"offset"`
	Payments []PaymentRecord `json:"payments"`
}

// UberClient exchanges OAuth codes for tokens and fetches driver profile,
// trip, and payment data. It is an interface so callers (AuthService,
// TripService, PaymentService) can run against MockUberClient without
// making real network calls — necessary right now since partner.accounts/
// partner.trips/partner.payments scope access is still pending Uber's
// approval (see uber-integration.md).
type UberClient interface {
	Exchange(ctx context.Context, code string) (*oauth2.Token, error)
	GetProfile(ctx context.Context, token *oauth2.Token) (*Profile, error)
	// ListTrips returns up to limit trips, paginating internally against
	// Uber's own max-50-per-page cap.
	ListTrips(ctx context.Context, token *oauth2.Token, limit int) ([]TripRecord, error)
	// ListPayments returns up to limit payments. An empty result is not
	// necessarily an error: Uber returns an empty list for drivers who
	// work for a fleet manager (payments go to the fleet, not the driver).
	ListPayments(ctx context.Context, token *oauth2.Token, limit int) ([]PaymentRecord, error)
}

// Client is the real UberClient implementation, backed by
// golang.org/x/oauth2 and Uber's documented partner endpoints. It is
// complete and correct per the confirmed field mapping, but cannot be
// integration-tested until Uber approves scope access.
type Client struct {
	cfg        *oauth2.Config
	httpClient *http.Client
}

// NewUberClient returns MockUberClient when useMock is true, otherwise a
// real Client for the given OAuth app registration. useMock should come
// from UBER_USE_MOCK (default true) — see uber-integration.md for why: the
// partner.accounts/partner.trips/partner.payments scopes are still pending
// Uber's approval.
func NewUberClient(useMock bool, clientID, clientSecret, redirectURI string) UberClient {
	if useMock {
		return NewMockUberClient()
	}
	return NewClient(clientID, clientSecret, redirectURI)
}

// NewClient builds a Client for the given OAuth app registration.
func NewClient(clientID, clientSecret, redirectURI string) *Client {
	return &Client{
		cfg: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURI,
			Endpoint: oauth2.Endpoint{
				AuthURL:  uberAuthURL,
				TokenURL: uberTokenURL,
			},
			Scopes: []string{"partner.accounts", "partner.trips", "partner.payments"},
		},
		httpClient: &http.Client{Timeout: requestTimeout},
	}
}

// Exchange trades an authorization code for an access/refresh token pair.
func (c *Client) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	return c.cfg.Exchange(ctx, code)
}

// GetProfile fetches the authenticated driver's profile, retrying once with
// a short backoff on transport errors or a non-2xx response.
func (c *Client) GetProfile(ctx context.Context, token *oauth2.Token) (*Profile, error) {
	var profile Profile
	if err := c.getWithRetry(ctx, uberProfileURL, token, nil, &profile); err != nil {
		return nil, fmt.Errorf("uber profile fetch failed: %w", err)
	}
	return &profile, nil
}

// ListTrips fetches up to limit of the driver's most recent trips,
// paginating against Uber's max-50-per-page cap until limit is reached or
// the server reports no more results (count <= offset).
func (c *Client) ListTrips(ctx context.Context, token *oauth2.Token, limit int) ([]TripRecord, error) {
	all := make([]TripRecord, 0, limit)
	offset := 0

	for len(all) < limit {
		pageSize := limit - len(all)
		if pageSize > uberMaxPageSize {
			pageSize = uberMaxPageSize
		}

		var page tripsResponse
		params := url.Values{
			"offset": {strconv.Itoa(offset)},
			"limit":  {strconv.Itoa(pageSize)},
		}
		if err := c.getWithRetry(ctx, uberTripsURL, token, params, &page); err != nil {
			return nil, fmt.Errorf("uber trips fetch failed: %w", err)
		}

		all = append(all, page.Trips...)
		offset += len(page.Trips)
		if len(page.Trips) == 0 || offset >= page.Count {
			break
		}
	}

	return all, nil
}

// ListPayments fetches up to limit of the driver's most recent payments,
// paginating the same way ListTrips does. An empty (but error-free) result
// is expected for fleet-managed drivers, per uber-integration.md.
func (c *Client) ListPayments(ctx context.Context, token *oauth2.Token, limit int) ([]PaymentRecord, error) {
	all := make([]PaymentRecord, 0, limit)
	offset := 0

	for len(all) < limit {
		pageSize := limit - len(all)
		if pageSize > uberMaxPageSize {
			pageSize = uberMaxPageSize
		}

		var page paymentsResponse
		params := url.Values{
			"offset": {strconv.Itoa(offset)},
			"limit":  {strconv.Itoa(pageSize)},
		}
		if err := c.getWithRetry(ctx, uberPaymentsURL, token, params, &page); err != nil {
			return nil, fmt.Errorf("uber payments fetch failed: %w", err)
		}

		all = append(all, page.Payments...)
		offset += len(page.Payments)
		if len(page.Payments) == 0 || offset >= page.Count {
			break
		}
	}

	return all, nil
}

// getWithRetry performs an authenticated GET against url (plus optional
// query params), decoding a 200 JSON response into dest. It retries once
// with a short backoff on transport errors or a non-2xx response.
func (c *Client) getWithRetry(ctx context.Context, endpoint string, token *oauth2.Token, params url.Values, dest interface{}) error {
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			select {
			case <-time.After(time.Duration(attempt) * 250 * time.Millisecond):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		if err := c.doGet(ctx, endpoint, token, params, dest); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	return lastErr
}

func (c *Client) doGet(ctx context.Context, endpoint string, token *oauth2.Token, params url.Values, dest interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	if len(params) > 0 {
		req.URL.RawQuery = params.Encode()
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(dest)
}
