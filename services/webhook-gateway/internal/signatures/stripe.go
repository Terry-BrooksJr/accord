package signatures

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
	"uuid"
)

// StripeVerifier verifies Stripe webhook signatures.
type StripeVerifier struct {
	// EndpointSecret is the signing secret for the Stripe webhook endpoint.
	EndpointSecret string
}

// HashPayload returns the SHA-256 hash of the verified event's raw payload
// encoded as a hexadecimal string.
func (ve *VerifiedEvent) HashPayload() string {
	sum := sha256.Sum256(ve.RawPayload)
	return hex.EncodeToString(sum[:])
}

// Verify validates the Stripe webhook signature using the Stripe-Signature
// header, the raw request body, and the configured endpoint secret.
// It returns a VerifiedEvent containing the event ID, type, original payload,
// and local receipt time. It returns an error if event construction or
// signature verification fails.
func (sv *StripeVerifier) Verify(
	ctx context.Context,
	headers http.Header,
	body []byte,
) (VerifiedEvent, error) {
	signature := headers.Get("Stripe-Signature")
	eventData, err := sc.ConstructEvent(body, signature, sv.EndpointSecret)
	if err != nil {
		slog.Error("Unable to verify Stripe webhook", "err", err)
		return VerifiedEvent{}, fmt.Errorf("unable to construct Stripe event during signature verification: %w", err)
	}
	event := VerifiedEvent{
		CorrelationID:   uuid.New(),
		Provider:        1,
		ProviderEventID: eventData.ID,
		EventType:       string(eventData.Type),
		ReceivedAt:      time.Now(),
		RawPayload:      json.RawMessage(body),
	}
	return event, nil
}

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
