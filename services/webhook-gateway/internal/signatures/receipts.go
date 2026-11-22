package signatures

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/google/uuid"
	"time"
)

// WebhookReceipt represents a persisted record of a received webhook.
type WebhookReceipt struct {
	ID                uuid.UUID       `gorm:"type:uuid;primaryKey"`
	CorrelationID     uuid.UUID       `gorm:"type:uuid;not null"`
	Provider          Provider        `gorm:"not null"`
	ProviderEventID   *string         `gorm:"index"`
	EventType         string          `gorm:"not null"`
	Payload           json.RawMessage `gorm:"type:jsonb;not null"`
	PayloadHash       string
	ReceivedAt        time.Time     `gorm:"not null"`
	ProviderCreatedAt time.Time     `gorm:"not null"`
	Status            ReceiptStatus `gorm:"not null"`
	IsDuplicate       bool          `gorm:"not null"`
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
