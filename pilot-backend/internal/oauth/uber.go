// Package oauth implements the OAuth2 authorization-code flow against
// Uber's driver API: exchanging a login code for tokens, then fetching the
// authenticated driver's profile.
package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"
)

const (
	uberAuthURL    = "https://login.uber.com/oauth/v2/authorize"
	uberTokenURL   = "https://login.uber.com/oauth/v2/token"
	uberProfileURL = "https://api.uber.com/v1.2/partners/me"

	requestTimeout = 10 * time.Second
	maxAttempts    = 2
)

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

// UberClient exchanges OAuth codes for tokens and fetches driver profiles.
// It is an interface so callers (AuthService) can be tested against a fake
// without making real network calls to Uber.
type UberClient interface {
	Exchange(ctx context.Context, code string) (*oauth2.Token, error)
	GetProfile(ctx context.Context, token *oauth2.Token) (*Profile, error)
}

// Client is the real UberClient implementation, backed by
// golang.org/x/oauth2 and Uber's documented partner endpoints.
type Client struct {
	cfg        *oauth2.Config
	httpClient *http.Client
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
			Scopes: []string{"profile", "partner.trips"},
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
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			select {
			case <-time.After(time.Duration(attempt) * 250 * time.Millisecond):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		profile, err := c.fetchProfile(ctx, token)
		if err == nil {
			return profile, nil
		}
		lastErr = err
	}
	return nil, fmt.Errorf("uber profile fetch failed: %w", lastErr)
}

func (c *Client) fetchProfile(ctx context.Context, token *oauth2.Token) (*Profile, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uberProfileURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var profile Profile
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return nil, err
	}
	return &profile, nil
}
