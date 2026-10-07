package signatures

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stripe/stripe-go/v87"
	"github.com/stripe/stripe-go/v87/webhook"
)

// TestStripeVerifierVerify is the Happy Path Test for the StripeVerifer Verify Reciever Function.
func TestStripeVerifierVerify(t *testing.T) {
	// Arrange: prepare the verifier, payload, and signed headers.
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

	verifier := &StripeVerifer{
		EndpointSecret: secret,
	}
	// Act: call Verify.
	before := time.Now()
	event, err := verifier.Verify(
		context.Background(),
		headers,
		body,
	)
	after := time.Now()

	// Assert: check the error and returned event.
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

// TestStripeVeriferVerifyRejectsInvalidWebhooks test the common occurance of a mishaped, invalid or pampered request.
func TestStripeVeriferVerifyRejectsInvalidWebhooks(t *testing.T) {
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

	tests := []struct {
		name             string
		signingSecret    string
		signingTime      time.Time
		missingSignature bool
		tamperBody       bool
	}{
		{
			name:          "wrong secret",
			signingSecret: "whsec_wrong_secret",
			signingTime:   time.Now(),
		},
		{
			name:             "missing signature",
			signingSecret:    secret,
			signingTime:      time.Now(),
			missingSignature: true,
		},
		{
			name:          "tampered body",
			signingSecret: secret,
			signingTime:   time.Now(),
			tamperBody:    true,
		},
		{
			name:          "expired signature",
			signingSecret: secret,
			signingTime:   time.Now().Add(-time.Hour),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signed := webhook.GenerateTestSignedPayload(
				&webhook.UnsignedPayload{
					Payload:   body,
					Secret:    tt.signingSecret,
					Timestamp: tt.signingTime,
				},
			)

			headers := make(http.Header)
			if !tt.missingSignature {
				headers.Set("Stripe-Signature", signed.Header)
			}

			requestBody := body
			if tt.tamperBody {
				// Change a value after signing, keeping valid JSON.
				requestBody = bytes.Replace(
					body,
					[]byte("evt_test_123"),
					[]byte("evt_test_456"),
					1,
				)
			}

			verifier := &StripeVerifer{
				EndpointSecret: secret,
			}

			event, err := verifier.Verify(
				context.Background(),
				headers,
				requestBody,
			)

			if err == nil {
				t.Fatal("expected verification to fail")
			}

			if event.ProviderEventID != "" ||
				len(event.Payload) != 0 {
				t.Error("verification failure returned event data")
			}
		})
	}
}
