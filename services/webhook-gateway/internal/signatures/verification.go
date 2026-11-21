package signatures

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"net/http"
	"time"
	"log/slog"
	"strings"
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
//go:generate stringer -type=Provider -trimprefix=Provider
func (p Provider) LogValue() slog.Value {
    return slog.StringValue(strings.ToLower(p.String()))
}
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
		rejection *WebhookRejection,
	) error
}
