package signatures

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"net"
	"net/http"
	"net/netip"
	"strings"
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

	// ReceiptStatusRejected indicates that the webhook was rejected.
	ReceiptStatusRejected ReceiptStatus = "REJECTED"

	// ReceiptStatusRejected indicates that the webhook is pending - New (DEFAULT STATUS ON CREATION).
	ReceiptStatusPendingNew ReceiptStatus = "PENDING - NEW"
)

// VerifiedEvent contains the normalized data extracted from a verified webhook.
type VerifiedEvent struct {
	CorrelationID     uuid.UUID
	Provider          Provider
	ProviderEventID   string `json:"id"`
	EventType         string `json:"type"`
	ReceivedAt        time.Time
	ProviderCreatedAt int64 `json:"created"`
	RawPayload        json.RawMessage
}

// WebhookReceipt represents a persisted record of a received webhook.
type WebhookReceipt struct {
	ID                uuid.UUID       `gorm:"type:uuid;primaryKey"`
	CorrelationID     uuid.UUID       `gorm:"type:uuid;not null"`
	Provider          Provider        `gorm:"not null"`
	ProviderEventID   *string         `gorm:"index"`
	EventType         string          `gorm:"not null"`
	Payload           json.RawMessage `gorm:"type:jsonb;not null"`
	PayloadHash       string          `gorm:"not null"`
	ReceivedAt        time.Time       `gorm:"not null"`
	ProviderCreatedAt time.Time       `gorm:"not null"`
	Status            ReceiptStatus   `gorm:"not null"`
	IsDuplicate       bool            `gorm:"not null"`
}

// WebhookVerifier verifies webhook requests and returns their normalized event data.
type WebhookVerifier interface {
	Verify(
		ctx context.Context,
		headers http.Header,
		body []byte,
	) (VerifiedEvent, error)
	CaptureRejection(
		body []byte,
		r http.Request,
		timeReceived time.Time,
		ctx context.Context,
		correlationID uuid.UUID,
		reasonRejected RejectionReasonType,
		details string,
	) (WebhookRejection, error)
}

// WebhookReceiptRepository persists webhook receipt records.
type WebhookReceiptRepository interface {
	RecordEvent(
		ctx context.Context,
		receipt *WebhookReceipt,
	) error
}

type WebhookRejectionRepository interface {
	LogRejection(
		ctx context.Context,
		receipt *WebhookReceipt,
	) error
}

// GORMWebhookReceiptRepository persists webhook receipts using GORM.
type GORMWebhookReceiptRepository struct {
	DB *gorm.DB
}

type GORMWebhookRejectionRepository struct {
	DB *gorm.DB
}

// HashPayload returns the SHA-256 hash of the verified event's raw payload
// encoded as a hexadecimal string.
func (ve *VerifiedEvent) HashPayload() string {
	sum := sha256.Sum256(ve.RawPayload)
	return hex.EncodeToString(sum[:])
}

// ProcessedReceipt contains the identifiers and processing metadata for a webhook receipt.
type ProcessedReceipt struct {
	AccordID      uuid.UUID
	CorrelationID uuid.UUID
	RecordedAt    time.Time
	RecievedAt    time.Time
	IsDuplicate   bool
}

// NewProcessedReciept creates a processed receipt from a webhook receipt.
func NewProcessedReciept(reciept WebhookReceipt) ProcessedReceipt {
	return ProcessedReceipt{
		AccordID:      reciept.ID,
		CorrelationID: reciept.CorrelationID,
		RecordedAt:    time.Now(),
		RecievedAt:    reciept.ReceivedAt,
		IsDuplicate:   reciept.IsDuplicate,
	}
}

// NewWebhookReceipt creates a webhook receipt from a verified event, processing status, and receipt time.
func NewWebhookReceipt(
	event VerifiedEvent,
	status ReceiptStatus,
	recievedAt time.Time,
	correlationID uuid.UUID,
) WebhookReceipt {
	epochConvertedTime := time.Unix(event.ProviderCreatedAt, 0)
	return WebhookReceipt{
		ID:                uuid.New(),
		CorrelationID:     correlationID,
		Provider:          event.Provider,
		ProviderEventID:   &event.ProviderEventID,
		EventType:         event.EventType,
		Payload:           event.RawPayload,
		PayloadHash:       event.HashPayload(),
		ReceivedAt:        recievedAt,
		ProviderCreatedAt: epochConvertedTime.UTC(),
		Status:            ReceiptStatusPendingNew,
	}
}

// TableName returns the database table name used for webhook receipts.
func (WebhookReceipt) TableName() string {
	return "webhook_receipts"
}

// RecordEvent persists a webhook receipt in the configured database.
func (r *GORMWebhookReceiptRepository) RecordEvent(
	ctx context.Context,
	receipt *WebhookReceipt,
) error {
	return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing WebhookReceipt
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
			receipt.Status = ReceiptStatusDuplicate
		}
		if err := tx.Create(receipt).Error; err != nil {
			return err
		}
		return nil
	})
}

func GetClientIP(r *http.Request, trustedProxy bool) (string, error) {
	peerIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "", fmt.Errorf("parse remote address: %w", err)
	}

	ip := peerIP

	if trustedProxy {
		forwarded := r.Header.Get("X-Forwarded-For")
		if forwarded != "" {
			// Only safe if Traefik has validated the forwarded chain.
			ip = strings.TrimSpace(strings.Split(forwarded, ",")[0])
		}
	}

	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return "", fmt.Errorf("parse client IP: %w", err)
	}

	return addr.Unmap().String(), nil
}
