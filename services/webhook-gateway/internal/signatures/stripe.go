package signatures

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v87/webhook"
	"log/slog"
	"net/http"
	"time"
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
	fmt.Print(signature)
	eventData, err := webhook.ConstructEvent(body, signature, sv.EndpointSecret)
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
