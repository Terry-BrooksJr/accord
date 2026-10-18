package signatures

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"net/http"
	"time"
)

// Provider identifies the external service that produced a webhook event.
type Provider int

// ReceiptStatus represents the processing status of a webhook receipt.
type ReceiptStatus string

const (
	// Unknown indicates that the webhook provider is not known.
	Unknown Provider = iota

	// Stripe identifies Stripe as the webhook provider.
	Stripe

	// Shippo identifies Shippo as the webhook provider.
	Shippo

	// Zendesk identifies Zendesk as the webhook provider.
	Zendesk

	// Plaid identifies Plaid as the webhook provider.
	Plaid

	// Other identifies a provider not represented by a named Provider constant.
	Other
)

const (
	// ReceiptStatusAccepted indicates that the webhook was accepted.
	ReceiptStatusAccepted ReceiptStatus = "ACCEPTED"

	// ReceiptStatusDuplicate indicates that the webhook was identified as a duplicate.
	ReceiptStatusDuplicate ReceiptStatus = "DUPLICATE"

	// ReceiptStatusError indicates that an error occurred while processing the webhook.
	ReceiptStatusError ReceiptStatus = "ERROR"
)

type JournalError struct {
	msg string
}

func (je *JournalError) Error() string {
	return fmt.Sprintf("jouraling error: $s", je.msg)
}

// VerifiedEvent contains the normalized data extracted from a verified webhook.
type VerifiedEvent struct {
	CorrelationID   uuid.UUID
	Provider        Provider
	ProviderEventID string
	EventType       string
	ReceivedAt      time.Time
	RawPayload      json.RawMessage
}

// WebhookReceipt represents a persisted record of a received webhook.
type WebhookReceipt struct {
	ID              uuid.UUID       `gorm:"type:uuid;primaryKey"`
	CorrelationID   uuid.UUID       `gorm:"type:uuid;not null"`
	Provider        Provider        `gorm:"not null"`
	ProviderEventID *string         `gorm:"index"`
	EventType       string          `gorm:"not null"`
	Payload         json.RawMessage `gorm:"type:jsonb;not null"`
	PayloadHash     string          `gorm:"not null"`
	ReceivedAt      time.Time       `gorm:"not null"`
	Status          ReceiptStatus   `gorm:"not null"`
	Duplicate       bool            `gorm:"not null"`
}

// WebhookVerifier verifies webhook requests and returns their normalized event data.
type WebhookVerifier interface {
	Verify(
		ctx context.Context,
		headers http.Header,
		body []byte,
	) (VerifiedEvent, error)
}

// WebhookReceiptRepository persists webhook receipt records.
type WebhookReceiptRepository interface {
	RecordEvent(
		ctx context.Context,
		receipt *WebhookReceipt,
	) error
}

// GORMWebhookReceiptRepository persists webhook receipts using GORM.
type GORMWebhookReceiptRepository struct {
	db *gorm.DB
}

// NewWebhookReceipt creates a webhook receipt from a verified event and processing status.
func NewWebhookReceipt(
	event VerifiedEvent,
	status ReceiptStatus,
) WebhookReceipt {
	return WebhookReceipt{
		ID:              uuid.New(),
		CorrelationID:   event.CorrelationID,
		Provider:        event.Provider,
		ProviderEventID: &event.ProviderEventID,
		EventType:       event.EventType,
		Payload:         event.RawPayload,
		PayloadHash:     event.HashPayload(),
		ReceivedAt:      event.ReceivedAt,
		Status:          status,
	}
}

// TableName returns the database table name used for webhook receipts.
func (WebhookReceipt) TableName() string {
	return "webhook_receipts"
}

// Create persists a webhook receipt in the configured database.
func (r *GORMWebhookReceiptRepository) RecordEvent(
	ctx context.Context,
	receipt *WebhookReceipt,
) error {

	var existing WebhookReceipt

	results := r.db.WithContext(ctx).Where(
		"provider = ? AND provider_event_id = ?",
		receipt.Provider,
		receipt.ProviderEventID).
		First(&existing)
	// There is not Duplicate Record Path
	if errors.Is(results.Error, gorm.ErrRecordNotFound) {
		return r.db.WithContext(ctx).Create(receipt).Error
	}

	// There is a Duplicate
	if results.Error != nil {
		return fmt.Errorf(
			"checking existing webhook receipt: %w",
			results.Error,
		)
	}
	return errors.New("error: unable to record event")
}
