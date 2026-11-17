package providers

import (
	"accord/services/webhook-gateway/internal/signatures"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v87"
	"github.com/stripe/stripe-go/v87/webhook"
	"log/slog"
	"net/http"
	"time"
)

// ErrMissingStripeSignature is raised when an incoming HTTP Request's headers has a nil value in or  missing the `Stripe-Signature`.
// This error is used in downstream conditional logic and modifies the rejection instatiatioin prior to persisting it in Rejection Log Repositiory.
var ErrMissingStripeSignature = errors.New("missing stripe signature")

// StripeVerifier verifies Stripe webhook signatures.
type StripeVerifier struct {
	// EndpointSecret is the signing secret for the Stripe webhook endpoint.
	EndpointSecret string
	EventData      stripe.Event
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
) (signatures.VerifiedEvent, error) {
	signature := headers.Get("Stripe-Signature")

	if signature == " " {
		slog.ErrorContext(ctx, "stripe-signature not in headers")
		return signatures.VerifiedEvent{}, ErrMissingStripeSignature
	}
	fmt.Print(signature) //TODO: Remove me LATER for testing ONLY
	eventData, err := webhook.ConstructEventWithOptions(body, signature, sv.EndpointSecret, webhook.ConstructEventOptions{IgnoreAPIVersionMismatch: true})
	if err != nil {
		slog.Error("Unable to verify Stripe webhook", "err", err)
		return signatures.VerifiedEvent{}, fmt.Errorf("unable to construct Stripe event during signature verification: %w", err)
	}
	slog.Info(fmt.Sprintf("incoming http request has been verified as Stripe event %v", eventData.ID))
	event := signatures.VerifiedEvent{
		CorrelationID:     uuid.New(),
		Provider:          signatures.Stripe,
		ProviderEventID:   eventData.ID,
		EventType:         string(eventData.Type),
		ReceivedAt:        time.Now(),
		ProviderCreatedAt: eventData.Created,
		RawPayload:        json.RawMessage(body),
	}
	return event, nil
}
