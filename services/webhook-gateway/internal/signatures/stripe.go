package signatures

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stripe/stripe-go/v87"
	"github.com/stripe/stripe-go/v87/webhook"
)

var sc *stripe.Client = stripe.NewClient(os.Getenv("STRIPE_API_KEY"))

// StripeVerifier verifies Stripe webhook signatures.
type StripeVerifier struct {
	// EndpointSecret is the signing secret for the Stripe webhook endpoint.
	EndpointSecret string
}

func (sv *StripeVerifier) HashPayload() string {
	sum :=  sha256.Sum256(sv.)
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
		slog.Error("ERROR: Unable to Verify Stripe Webhood", "err", err)
		return VerifiedEvent{}, fmt.Errorf("Unable to Construct Event During Strip Signature Verififcation:%w", err)
	}
	event := VerifiedEvent{
		Provider:        1,
		ProviderEventID: eventData.ID,
		EventType:       string(eventData.Type),
		ReceivedAt:      time.Now(),
		RawPayload:      json.RawMessage(body),
	}
	return event, nil
}

func TestStripeVeriferVerifyValidWebhook(t *testing.T) {
	// Arrange: create a fixture compatible with this SDK version.
	const secret = "whsec_test_secret"

	body, err := json.Marshal(map[string]any{
		"id":          "evt_test_123",
		"object":      "event",
		"api_version": stripe.APIVersion,
		"type":        "payment_intent.succeeded",
		"data": map[string]any{
			"object": map[string]any{
				"id":     "pi_test_123",
				"object": "payment_intent",
			},
		},
	})
	if err != nil {
		t.Fatalf("create fixture: %v", err)
	}

	signed := webhook.GenerateTestSignedPayload(
		&webhook.UnsignedPayload{
			Payload:   body,
			Secret:    secret,
			Timestamp: time.Now(),
		},
	)

	headers := make(http.Header)
	headers.Set("Stripe-Signature", signed.Header)

	verifier := &StripeVerifier{
		EndpointSecret: secret,
	}

	// Act.
	before := time.Now()
	event, err := verifier.Verify(
		context.Background(),
		headers,
		body,
	)
	after := time.Now()

	// Assert.
	if err != nil {
		t.Fatalf("expected successful verification, got: %v", err)
	}

	if event.ProviderEventID != "evt_test_123" {
		t.Errorf("unexpected event ID: %q", event.ProviderEventID)
	}

	if event.EventType != "payment_intent.succeeded" {
		t.Errorf("unexpected event type: %q", event.EventType)
	}

	if !bytes.Equal(event.Payload, body) {
		t.Error("payload does not preserve the original webhook body")
	}

	if event.ReceivedAt.Before(before) ||
		event.ReceivedAt.After(after) {
		t.Error("received timestamp is outside the verification interval")
	}
}
