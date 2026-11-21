package journal

import (
	"accord/services/webhook-gateway/internal/signatures"
	"context"
	"errors"
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"log/slog"
	"log"
	"gorm.io/gorm/logger"
	"os"
	"time"
)

func NewRepository(dsn string) (*GORMWebhookReceiptRepository, error) {
db, err := gorm.Open(
    postgres.Open(dsn),
    &gorm.Config{
        Logger: logger.New(
            log.New(os.Stderr, "", log.LstdFlags),
            logger.Config{
                LogLevel:                  logger.Warn,
                SlowThreshold:             200 * time.Millisecond,
                IgnoreRecordNotFoundError: true,
            },
        ),
    },
)
	if err != nil {
		return nil, fmt.Errorf("unable to establish connection to DB: %w", err)
	}
	return &GORMWebhookReceiptRepository{DB: db}, nil
}

func NewRejectionLog(dsn string) (*GORMWebhookRejectionRepository, error) {
db, err := gorm.Open(
    postgres.Open(dsn),
    &gorm.Config{
        Logger: logger.New(
            log.New(os.Stderr, "", log.LstdFlags),
            logger.Config{
                LogLevel:                  logger.Warn,
                SlowThreshold:             200 * time.Millisecond,
                IgnoreRecordNotFoundError: true,
            },
        ),
    },
)
	if err != nil {
		return nil, fmt.Errorf("unable to establish connection to DB: %w", err)
	}
	return &GORMWebhookRejectionRepository{DB: db}, nil
}

// GORMWebhookReceiptRepository persists webhook receipts using GORM.
type GORMWebhookReceiptRepository struct {
	DB *gorm.DB
}

type GORMWebhookRejectionRepository struct {
	DB *gorm.DB
}

// RecordEvent persists a webhook receipt in the configured database.
func (r *GORMWebhookReceiptRepository) RecordEvent(
	ctx context.Context,
	receipt *signatures.WebhookReceipt,
) error {
	return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing signatures.WebhookReceipt
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("provider = ? AND provider_event_id = ?", receipt.Provider, receipt.ProviderEventID).
			First(&existing).Error

		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {

			} else {
				return err
			}
		} else {
			slog.Warn(fmt.Sprintf("Event %v is a duplicated event. recording as duplicate of Provider ID: %v\n", receipt.ID, receipt.ProviderEventID))
			receipt.IsDuplicate = true
			receipt.Status = signatures.ReceiptStatusDuplicate
		}
		if err := tx.Create(receipt).Error; err != nil {
			return err
		}
		slog.InfoContext(ctx, "distinct webhook receipt recorded",
    "provider", receipt.Provider,
    "provider_event_id", receipt.ID,
    "correlation_id", receipt.CorrelationID,
)
		return nil
	})
}

func (r *GORMWebhookRejectionRepository) LogRejection(
	ctx context.Context,
	rejection *signatures.WebhookRejection,
) error {
	return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Create(&rejection)
		if result.Error != nil {
			slog.Warn("unable to insert row into rejection log", "error", result.Error, "rejection_data", rejection)
			return result.Error
		}
		if result.RowsAffected == 1 {
			slog.InfoContext(ctx, "webhook receipt recorded",
    "provider", rejection.Provider,
    "provider_event_id", rejection.ID,
    "correlation_id", rejection.CorrelationID,
)
			return nil
		}
		return errors.New("Unable to Log Rejection. ")
	})
}
