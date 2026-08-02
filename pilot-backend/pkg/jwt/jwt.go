// Package jwt issues and validates the HS256 access tokens drivers use to
// authenticate against the API.
package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ErrInvalidToken is returned for malformed tokens, bad signatures, or any
// other validation failure not covered by a more specific error.
var ErrInvalidToken = errors.New("invalid token")

// ErrTokenExpired is returned when a token's exp claim is in the past.
var ErrTokenExpired = errors.New("token expired")

const (
	issuer   = "pilot-app"
	audience = "pilot-drivers"
)

// Claims are the JWT claims embedded in every access token.
type Claims struct {
	DriverID int64  `json:"driver_id"`
	Email    string `json:"email"`
	jwt.RegisteredClaims
}

// Manager generates and validates tokens signed with a single HS256 secret.
type Manager struct {
	secret string
}

// NewManager builds a Manager using secret to sign and verify tokens.
func NewManager(secret string) *Manager {
	return &Manager{secret: secret}
}

// GenerateToken issues a signed token for driverID/email, valid for expiry.
func (m *Manager) GenerateToken(driverID int64, email string, expiry time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		DriverID: driverID,
		Email:    email,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
			Issuer:    issuer,
			Audience:  jwt.ClaimStrings{audience},
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.secret))
}

// ValidateToken parses and verifies tokenString, returning its claims if the
// signature, issuer/audience, and expiry all check out.
func (m *Manager) ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, ErrInvalidToken
			}
			return []byte(m.secret), nil
		},
		jwt.WithIssuer(issuer),
		jwt.WithAudience(audience),
	)

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
