package service

import (
	"context"
	"time"

	"golang.org/x/oauth2"

	"pilot-backend/internal/models"
	"pilot-backend/internal/oauth"
	"pilot-backend/internal/repository"
	"pilot-backend/pkg/crypto"
	"pilot-backend/pkg/logger"
)

const defaultSyncLimit = 50

// TripService lists a driver's trips and syncs new ones from Uber.
type TripService struct {
	tripRepo    *repository.TripRepository
	driverRepo  *repository.DriverRepository
	encryptor   *crypto.Encryptor
	oauthClient oauth.UberClient
	log         *logger.Logger
}

// NewTripService builds a TripService from its collaborators.
func NewTripService(
	tripRepo *repository.TripRepository,
	driverRepo *repository.DriverRepository,
	encryptor *crypto.Encryptor,
	oauthClient oauth.UberClient,
	log *logger.Logger,
) *TripService {
	return &TripService{
		tripRepo:    tripRepo,
		driverRepo:  driverRepo,
		encryptor:   encryptor,
		oauthClient: oauthClient,
		log:         log,
	}
}

// ListTrips returns a page of the driver's trips ended within [start, end],
// optionally filtered by status.
func (s *TripService) ListTrips(ctx context.Context, driverID int64, start, end time.Time, status string, limit, offset int) ([]models.Trip, models.Pagination, error) {
	trips, total, err := s.tripRepo.GetByDateRange(ctx, driverID, start, end, status, limit, offset)
	if err != nil {
		s.log.Errorw("trip.list_failed", "driver_id", driverID, "error", err.Error())
		return nil, models.Pagination{}, models.NewAPIError(models.ErrCodeDatabaseError, "database error")
	}
	return trips, models.NewPagination(limit, offset, total), nil
}

// GetTrip returns a single trip, scoped to driverID so a driver can never
// fetch another driver's trip by guessing an ID.
func (s *TripService) GetTrip(ctx context.Context, driverID, tripID int64) (*models.Trip, error) {
	trip, err := s.tripRepo.GetByID(ctx, driverID, tripID)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	return trip, nil
}

// SyncTrips fetches the driver's recent trips from Uber and persists any
// not already stored (deduped by uber_trip_id), returning how many were
// newly synced.
func (s *TripService) SyncTrips(ctx context.Context, driverID int64, limit int) (int, time.Time, error) {
	if limit <= 0 {
		limit = defaultSyncLimit
	}

	driver, err := s.driverRepo.GetByID(ctx, driverID)
	if err != nil {
		return 0, time.Time{}, mapRepoErr(err)
	}

	accessToken, err := s.encryptor.Decrypt(driver.OAuthToken)
	if err != nil {
		s.log.Errorw("trip.token_decryption_failed", "driver_id", driverID)
		return 0, time.Time{}, models.NewAPIError(models.ErrCodeEncryption, "failed to read stored credentials")
	}

	expiry := time.Now()
	if driver.TokenExpiresAt != nil {
		expiry = *driver.TokenExpiresAt
	}

	records, err := s.oauthClient.ListTrips(ctx, &oauth2.Token{AccessToken: accessToken, Expiry: expiry}, limit)
	if err != nil {
		s.log.Errorw("trip.sync_fetch_failed", "driver_id", driverID, "error", err.Error())
		return 0, time.Time{}, models.NewAPIError(models.ErrCodeOAuthFailed, "failed to sync trips from Uber")
	}

	synced := 0
	for _, rec := range records {
		if _, err := s.tripRepo.GetByUberTripID(ctx, rec.TripID); err == nil {
			continue // already synced
		}

		trip := &models.Trip{
			DriverID:   driverID,
			UberTripID: rec.TripID,
			StartedAt:  time.Unix(rec.StartTime, 0),
			EndedAt:    time.Unix(rec.EndTime, 0),
			DistanceKM: rec.DistanceKM,
			FareValue:  rec.Fare,
			Currency:   currencyOrDefault(rec.Currency),
			City:       rec.City,
			Status:     statusOrDefault(rec.Status),
		}

		if err := trip.Validate(); err != nil {
			s.log.Warnw("trip.sync_skipped_invalid", "driver_id", driverID, "uber_trip_id", rec.TripID, "error", err.Error())
			continue
		}
		if err := s.tripRepo.Create(ctx, trip); err != nil {
			s.log.Errorw("trip.sync_create_failed", "driver_id", driverID, "uber_trip_id", rec.TripID, "error", err.Error())
			continue
		}
		synced++
	}

	return synced, time.Now(), nil
}

func currencyOrDefault(currency string) string {
	if currency == "" {
		return "BRL"
	}
	return currency
}

func statusOrDefault(status string) string {
	if status == "" {
		return "COMPLETED"
	}
	return status
}
