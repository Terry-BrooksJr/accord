package signatures

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"time"
	"uuid"

	"github.com/stripe/stripe-go/v87"
)

// Top-Level Type  and Variable Definitions
type Provider int
type ReceiptStatus string

const (
	Unknown Provider = iota
	Stripe
	Shippo
	Zendesk
	Plaid
	Other
)

const (
	ReceiptStatusAccepted  ReceiptStatus = "ACCEPTED"
	ReceiptStatusDuplicate ReceiptStatus = "DUPLICATE"
	ReceiptStatusError     ReceiptStatus = "ERROR"
)

type VerifiedEvent struct {
	CorrelationID   uuid.UUID
	Provider        Provider
	ProviderEventID string
	EventType       string
	ReceivedAt      time.Time
	RawPayload      json.RawMessage
}

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

// Provider Clients:
var sc *stripe.Client = stripe.NewClient(os.Getenv("STRIPE_API_KEY"))

func (WebhookReceipt) TableName() string {
	return "webhook_receipts"
}

type Verifer interface {
	Verify(
		ctx context.Context,
		headers http.Header,
		body []byte,
	) (VerifiedEvent, error)
}

type HashPayload interface {
}
