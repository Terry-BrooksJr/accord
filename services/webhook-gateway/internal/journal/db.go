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
)

func NewRepository(dsn string) (*GORMWebhookReceiptRepository, error) {
	db, err := gorm.Open(
		postgres.Open(dsn),
		&gorm.Config{},
	)
	if err != nil {
		return nil, fmt.Errorf("unable to establish connection to DB: %w", err)
	}
	return &GORMWebhookReceiptRepository{DB: db}, nil
}

func NewRejectionLog(dsn string) (*GORMWebhookRejectionRepository, error) {
	db, err := gorm.Open(
		postgres.Open(dsn),
		&gorm.Config{},
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
			receipt.IsDuplicate = true
			receipt.Status = signatures.ReceiptStatusDuplicate
		}
		if err := tx.Create(receipt).Error; err != nil {
			return err
		}
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
			slog.Info(fmt.Sprintf("rejection %v succcessfully logged ", rejection.ID))
			return nil
		}
		return errors.New("Unable to Log Rejection. ")
	})
}
