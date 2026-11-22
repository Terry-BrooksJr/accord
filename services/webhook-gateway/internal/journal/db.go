package journal

import (
	"accord/services/webhook-gateway/internal/signatures"
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"io"
	"log"
	"log/slog"
	"net"
	"os"
	"strings"
	"time"
)

var ErrPostGresUnavailable = errors.New("postgres servert is unavailable")

func isDatabaseUnavailable(err error) bool {
	if err == nil ||
		errors.Is(err, context.Canceled) ||
		errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// Match PostgreSQL errors without depending on a specific pgx version.
	var pgErr interface {
		SQLState() string
	}
	if errors.As(err, &pgErr) {
		code := pgErr.SQLState()

		switch code {
		case "53300": // Too many connections.
			return true
		case "57P01", "57P02", "57P03": // Shutdown/crash/not accepting connections.
			return true
		}

		return strings.HasPrefix(code, "08") // Connection exceptions.
	}

	var networkErr net.Error
	return errors.As(err, &networkErr) ||
		errors.Is(err, driver.ErrBadConn) ||
		errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrUnexpectedEOF)
}

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
		return nil, fmt.Errorf("%w: %w", ErrPostGresUnavailable, err)
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
		return nil, fmt.Errorf("%w: %w", ErrPostGresUnavailable, err)
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
	err := r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing signatures.WebhookReceipt

		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where(
				"provider = ? AND provider_event_id = ?",
				receipt.Provider,
				receipt.ProviderEventID,
			).
			First(&existing).Error

		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			// First receipt for this provider event.
		case err != nil:
			return err
		default:
			receipt.IsDuplicate = true
			receipt.Status = signatures.ReceiptStatusDuplicate
		}

		return tx.Create(receipt).Error
	})

	if err != nil {
		if isDatabaseUnavailable(err) {
			return fmt.Errorf("%w: %w", ErrPostGresUnavailable, err)
		}

		return fmt.Errorf("record webhook receipt: %w", err)
	}

	slog.InfoContext(ctx, "webhook receipt recorded",
		"provider", receipt.Provider,
		"provider_event_id", receipt.ProviderEventID,
		"correlation_id", receipt.CorrelationID,
		"is_duplicate", receipt.IsDuplicate,
	)

	return nil
}

func (r *GORMWebhookRejectionRepository) LogRejection(
	ctx context.Context,
	rejection *signatures.WebhookRejection,
) error {
	err := r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Create(rejection)

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected != 1 {
			return errors.New("expected one rejection row to be inserted")
		}

		return nil
	})

	if err != nil {
		if isDatabaseUnavailable(err) {
			return fmt.Errorf("%w: %w", ErrPostGresUnavailable, err)
		}

		return fmt.Errorf("log webhook rejection: %w", err)
	}

	slog.InfoContext(ctx, "logged rejected http request",
		"provider", rejection.Provider,
		"correlation_id", rejection.CorrelationID,
	)

	return nil
}
