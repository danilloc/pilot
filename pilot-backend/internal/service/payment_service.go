package service

import (
	"context"
	"math"
	"time"

	"golang.org/x/oauth2"

	"pilot-backend/internal/models"
	"pilot-backend/internal/oauth"
	"pilot-backend/internal/repository"
	"pilot-backend/pkg/crypto"
	"pilot-backend/pkg/logger"
)

// PaymentService lists a driver's payments and syncs new ones from Uber.
type PaymentService struct {
	paymentRepo *repository.PaymentRepository
	driverRepo  *repository.DriverRepository
	encryptor   *crypto.Encryptor
	oauthClient oauth.UberClient
	log         *logger.Logger
}

// NewPaymentService builds a PaymentService from its collaborators.
func NewPaymentService(
	paymentRepo *repository.PaymentRepository,
	driverRepo *repository.DriverRepository,
	encryptor *crypto.Encryptor,
	oauthClient oauth.UberClient,
	log *logger.Logger,
) *PaymentService {
	return &PaymentService{
		paymentRepo: paymentRepo,
		driverRepo:  driverRepo,
		encryptor:   encryptor,
		oauthClient: oauthClient,
		log:         log,
	}
}

// ListPayments returns a page of the driver's payments within [start, end].
func (s *PaymentService) ListPayments(ctx context.Context, driverID int64, start, end time.Time, limit, offset int) ([]models.PaymentRecord, models.Pagination, error) {
	payments, total, err := s.paymentRepo.GetByDateRange(ctx, driverID, start, end, limit, offset)
	if err != nil {
		s.log.Errorw("payment.list_failed", "driver_id", driverID, "error", err.Error())
		return nil, models.Pagination{}, models.NewAPIError(models.ErrCodeDatabaseError, "database error")
	}
	return payments, models.NewPagination(limit, offset, total), nil
}

// SyncPayments fetches the driver's recent payments from Uber and persists
// any not already stored (deduped by uber_payment_id). An empty Uber
// response is not an error: per uber-integration.md, drivers who work for
// a fleet manager always get an empty payments list (payments go to the
// fleet, not the driver) — that's a valid outcome, synced_count is just 0.
func (s *PaymentService) SyncPayments(ctx context.Context, driverID int64, limit int) (int, time.Time, error) {
	if limit <= 0 {
		limit = defaultSyncLimit
	}

	driver, err := s.driverRepo.GetByID(ctx, driverID)
	if err != nil {
		return 0, time.Time{}, mapRepoErr(err)
	}

	accessToken, err := s.encryptor.Decrypt(driver.OAuthToken)
	if err != nil {
		s.log.Errorw("payment.token_decryption_failed", "driver_id", driverID)
		return 0, time.Time{}, models.NewAPIError(models.ErrCodeEncryption, "failed to read stored credentials")
	}

	expiry := time.Now()
	if driver.TokenExpiresAt != nil {
		expiry = *driver.TokenExpiresAt
	}

	records, err := s.oauthClient.ListPayments(ctx, &oauth2.Token{AccessToken: accessToken, Expiry: expiry}, limit)
	if err != nil {
		s.log.Errorw("payment.sync_fetch_failed", "driver_id", driverID, "error", err.Error())
		return 0, time.Time{}, models.NewAPIError(models.ErrCodeOAuthFailed, "failed to sync payments from Uber")
	}

	synced := 0
	for _, rec := range records {
		if _, err := s.paymentRepo.GetByUberPaymentID(ctx, rec.PaymentID); err == nil {
			continue // already synced
		}

		payment := &models.PaymentRecord{
			DriverID:      driverID,
			UberPaymentID: rec.PaymentID,
			Amount:        rec.Amount,
			Currency:      currencyOrDefault(rec.CurrencyCode),
			Tolls:         rec.Breakdown.Toll,
			// breakdown.service_fee is the closest documented field to a
			// tax-like deduction, and can be negative (a credit, not a
			// charge) — per uber-integration.md this mapping is an
			// unconfirmed approximation, revisit once real payments access
			// is granted. abs() keeps it compatible with our own
			// PaymentRecord.Validate(), which rejects negative taxes.
			Taxes:         math.Abs(rec.Breakdown.ServiceFee),
			PaymentMethod: rec.Category,
			PaymentDate:   time.Unix(rec.EventTime, 0),
		}

		if err := payment.Validate(); err != nil {
			s.log.Warnw("payment.sync_skipped_invalid", "driver_id", driverID, "uber_payment_id", rec.PaymentID, "error", err.Error())
			continue
		}
		if err := s.paymentRepo.Create(ctx, payment); err != nil {
			s.log.Errorw("payment.sync_create_failed", "driver_id", driverID, "uber_payment_id", rec.PaymentID, "error", err.Error())
			continue
		}
		synced++
	}

	return synced, time.Now(), nil
}
