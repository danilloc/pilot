package service

import (
	"context"
	"errors"
	"time"

	"golang.org/x/oauth2"

	"pilot-backend/internal/models"
	"pilot-backend/internal/oauth"
	"pilot-backend/internal/repository"
	"pilot-backend/pkg/crypto"
	"pilot-backend/pkg/logger"
)

// DriverService implements profile read/update and re-sync against Uber
// for the authenticated driver.
type DriverService struct {
	driverRepo  *repository.DriverRepository
	encryptor   *crypto.Encryptor
	oauthClient oauth.UberClient
	log         *logger.Logger
}

// NewDriverService builds a DriverService from its collaborators.
func NewDriverService(
	driverRepo *repository.DriverRepository,
	encryptor *crypto.Encryptor,
	oauthClient oauth.UberClient,
	log *logger.Logger,
) *DriverService {
	return &DriverService{
		driverRepo:  driverRepo,
		encryptor:   encryptor,
		oauthClient: oauthClient,
		log:         log,
	}
}

// GetProfile returns the driver's redacted profile.
func (s *DriverService) GetProfile(ctx context.Context, driverID int64) (*models.Driver, error) {
	driver, err := s.driverRepo.GetByID(ctx, driverID)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	driver.Redact()
	return driver, nil
}

// UpdateProfile applies client-editable fields (currently just phone — name,
// email, and rating are Uber-sourced and only change via SyncProfile) and
// persists the result.
func (s *DriverService) UpdateProfile(ctx context.Context, driverID int64, phone string) (*models.Driver, error) {
	driver, err := s.driverRepo.GetByID(ctx, driverID)
	if err != nil {
		return nil, mapRepoErr(err)
	}

	if phone != "" {
		driver.Phone = phone
	}

	if err := driver.Validate(); err != nil {
		return nil, err
	}

	if err := s.driverRepo.Update(ctx, driver); err != nil {
		s.log.Errorw("driver.update_failed", "driver_id", driverID, "error", err.Error())
		return nil, models.NewAPIError(models.ErrCodeDatabaseError, "failed to update driver")
	}

	driver.Redact()
	return driver, nil
}

// SyncProfile re-fetches the driver's profile from Uber using their stored
// (encrypted) access token and updates the locally cached fields.
func (s *DriverService) SyncProfile(ctx context.Context, driverID int64) (time.Time, error) {
	driver, err := s.driverRepo.GetByID(ctx, driverID)
	if err != nil {
		return time.Time{}, mapRepoErr(err)
	}

	accessToken, err := s.encryptor.Decrypt(driver.OAuthToken)
	if err != nil {
		s.log.Errorw("driver.token_decryption_failed", "driver_id", driverID)
		return time.Time{}, models.NewAPIError(models.ErrCodeEncryption, "failed to read stored credentials")
	}

	expiry := time.Now()
	if driver.TokenExpiresAt != nil {
		expiry = *driver.TokenExpiresAt
	}

	profile, err := s.oauthClient.GetProfile(ctx, &oauth2.Token{AccessToken: accessToken, Expiry: expiry})
	if err != nil {
		s.log.Errorw("driver.sync_profile_failed", "driver_id", driverID, "error", err.Error())
		return time.Time{}, models.NewAPIError(models.ErrCodeOAuthFailed, "failed to sync profile from Uber")
	}

	driver.Name = profile.FirstName + " " + profile.LastName
	driver.Email = profile.Email
	driver.Phone = profile.MobilePhoneNumber
	driver.ProfilePictureURL = profile.PictureURL
	driver.Rating = profile.Rating

	now := time.Now()
	driver.LastSyncedAt = &now

	if err := driver.Validate(); err != nil {
		return time.Time{}, err
	}

	if err := s.driverRepo.Update(ctx, driver); err != nil {
		s.log.Errorw("driver.sync_persist_failed", "driver_id", driverID, "error", err.Error())
		return time.Time{}, models.NewAPIError(models.ErrCodeDatabaseError, "failed to persist synced profile")
	}

	return now, nil
}

func mapRepoErr(err error) error {
	if errors.Is(err, repository.ErrNotFound) {
		return models.NewAPIError(models.ErrCodeNotFound, "driver not found")
	}
	return models.NewAPIError(models.ErrCodeDatabaseError, "database error")
}
