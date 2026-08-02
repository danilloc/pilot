// Package service holds business logic that sits between HTTP handlers and
// the repository/cache layers.
package service

import (
	"context"
	"errors"
	"time"

	"pilot-backend/internal/models"
	"pilot-backend/internal/oauth"
	"pilot-backend/internal/repository"
	"pilot-backend/pkg/cache"
	"pilot-backend/pkg/crypto"
	"pilot-backend/pkg/jwt"
	"pilot-backend/pkg/logger"
)

// AuthService implements the Uber OAuth login flow, session validation, and
// logout. It never returns a raw internal error to callers — always an
// *models.APIError safe to serialize straight to the client.
type AuthService struct {
	driverRepo  *repository.DriverRepository
	jwtManager  *jwt.Manager
	jwtExpiry   time.Duration
	encryptor   *crypto.Encryptor
	oauthClient oauth.UberClient
	redis       *cache.Redis
	log         *logger.Logger
}

// NewAuthService builds an AuthService from its collaborators.
func NewAuthService(
	driverRepo *repository.DriverRepository,
	jwtManager *jwt.Manager,
	jwtExpiry time.Duration,
	encryptor *crypto.Encryptor,
	oauthClient oauth.UberClient,
	redis *cache.Redis,
	log *logger.Logger,
) *AuthService {
	return &AuthService{
		driverRepo:  driverRepo,
		jwtManager:  jwtManager,
		jwtExpiry:   jwtExpiry,
		encryptor:   encryptor,
		oauthClient: oauthClient,
		redis:       redis,
		log:         log,
	}
}

// LoginWithUberCode exchanges an Uber OAuth code for tokens, fetches the
// driver's profile, upserts the driver record (tokens encrypted at rest),
// and returns a JWT the client uses for subsequent requests.
func (s *AuthService) LoginWithUberCode(ctx context.Context, code, clientIP string) (string, *models.Driver, error) {
	if code == "" {
		s.log.Warnw("auth.empty_code")
		return "", nil, models.NewAPIError(models.ErrCodeOAuthFailed, "authorization code is required")
	}

	token, err := s.oauthClient.Exchange(ctx, code)
	if err != nil {
		s.log.Errorw("auth.oauth_exchange_failed", "error", err.Error())
		return "", nil, models.NewAPIError(models.ErrCodeOAuthFailed, "authentication failed")
	}

	profile, err := s.oauthClient.GetProfile(ctx, token)
	if err != nil {
		s.log.Errorw("auth.profile_fetch_failed", "error", err.Error())
		return "", nil, models.NewAPIError(models.ErrCodeOAuthFailed, "failed to fetch driver profile")
	}

	// A banned driver must not get a fresh session just by re-authenticating
	// with Uber — check the existing record (if any) before upserting a new
	// one, since the struct we're about to build always starts unbanned.
	if existing, err := s.driverRepo.GetByUberID(ctx, profile.DriverID); err == nil && existing.IsBanned {
		s.log.Warnw("auth.banned_driver_login_attempt", "uber_id", profile.DriverID)
		return "", nil, models.ErrDriverBanned
	}

	encryptedAccessToken, err := s.encryptor.Encrypt(token.AccessToken)
	if err != nil {
		s.log.Errorw("auth.token_encryption_failed")
		return "", nil, models.NewAPIError(models.ErrCodeEncryption, "failed to secure credentials")
	}
	encryptedRefreshToken, err := s.encryptor.Encrypt(token.RefreshToken)
	if err != nil {
		s.log.Errorw("auth.refresh_token_encryption_failed")
		return "", nil, models.NewAPIError(models.ErrCodeEncryption, "failed to secure credentials")
	}

	now := time.Now()
	driver := &models.Driver{
		UberID:            profile.DriverID,
		Name:              profile.FirstName + " " + profile.LastName,
		Email:             profile.Email,
		Phone:             profile.MobilePhoneNumber,
		ProfilePictureURL: profile.PictureURL,
		Rating:            profile.Rating,
		AccountStatus:     "ACTIVE",
		OAuthToken:        encryptedAccessToken,
		OAuthRefreshToken: encryptedRefreshToken,
		TokenExpiresAt:    &token.Expiry,
		LastLoginIP:       clientIP,
		LastLoginTime:     &now,
		LastSyncedAt:      &now,
	}

	if err := driver.Validate(); err != nil {
		return "", nil, err
	}

	if err := s.driverRepo.UpsertDriver(ctx, driver); err != nil {
		s.log.Errorw("auth.driver_upsert_failed", "error", err.Error())
		return "", nil, models.NewAPIError(models.ErrCodeDatabaseError, "failed to persist driver")
	}

	// MySQL's ON DUPLICATE KEY UPDATE path doesn't populate driver.ID on
	// GORM's struct when the row already existed, only on a fresh insert —
	// so re-fetch to get the real, persisted ID for the JWT claims.
	persisted, err := s.driverRepo.GetByUberID(ctx, driver.UberID)
	if err != nil {
		s.log.Errorw("auth.driver_refetch_failed", "error", err.Error())
		return "", nil, models.NewAPIError(models.ErrCodeDatabaseError, "failed to load driver")
	}

	jwtToken, err := s.jwtManager.GenerateToken(persisted.ID, persisted.Email, s.jwtExpiry)
	if err != nil {
		s.log.Errorw("auth.jwt_generation_failed", "error", err.Error())
		return "", nil, models.NewAPIError(models.ErrCodeInternalError, "failed to generate session token")
	}

	s.log.Infow("auth.login_success", "driver_id", persisted.ID)

	persisted.Redact()
	return jwtToken, persisted, nil
}

// Logout blacklists token so it's rejected immediately, even though its
// signature and expiry remain otherwise valid.
func (s *AuthService) Logout(ctx context.Context, token string) error {
	ttl := s.jwtExpiry
	if claims, err := s.jwtManager.ValidateToken(token); err == nil {
		if remaining := time.Until(claims.ExpiresAt.Time); remaining > 0 {
			ttl = remaining
		}
	} else if !errors.Is(err, jwt.ErrTokenExpired) {
		// Malformed token: nothing meaningful to blacklist.
		return models.NewAPIError(models.ErrCodeInvalidToken, "invalid token")
	}

	if err := s.redis.Blacklist(ctx, token, ttl); err != nil {
		s.log.Errorw("auth.logout_failed", "error", err.Error())
		return models.NewAPIError(models.ErrCodeInternalError, "logout failed")
	}

	s.log.Infow("auth.logout_success")
	return nil
}
