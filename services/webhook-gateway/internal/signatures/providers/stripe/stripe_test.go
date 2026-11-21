package providers

import (
	"accord/services/webhook-gateway/internal/signatures"
	"bytes"
	"context"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	stripesdk "github.com/stripe/stripe-go/v87"
	"github.com/stripe/stripe-go/v87/webhook"
	"net/http"
	"testing"
	"time"
)

const testSigningSecret = "whsec_test_secret"

func stripeEventFixture(t *testing.T) []byte {
	t.Helper()

	body, err := json.Marshal(map[string]any{
		"id":          "evt_test_123",
		"object":      "event",
		"api_version": stripesdk.APIVersion,
		"type":        "payment_intent.succeeded",
		"data": map[string]any{
			"object": map[string]any{
				"id":     "pi_test_123",
				"object": "payment_intent",
			},
		},
	})
	require.NoError(t, err, "create Stripe event fixture")
	return body
}

func TestStripeVerifierVerify(t *testing.T) {
	body := stripeEventFixture(t)
	signed := webhook.GenerateTestSignedPayload(&webhook.UnsignedPayload{
		Payload:   body,
		Secret:    testSigningSecret,
		Timestamp: time.Now(),
	})

	headers := make(http.Header)
	headers.Set("Stripe-Signature", signed.Header)
	verifier := &StripeVerifier{EndpointSecret: testSigningSecret}

	before := time.Now()
	event, err := verifier.Verify(context.Background(), headers, body)
	after := time.Now()

	require.NoError(t, err)
	assert.Equal(t, "evt_test_123", event.ProviderEventID)
	assert.Equal(t, "payment_intent.succeeded", event.EventType)
	assert.Equal(t, body, []byte(event.RawPayload), "preserve the original webhook body")
	assert.WithinRange(t, event.ReceivedAt, before, after)
}

func TestStripeVerifierVerifyRejectsInvalidWebhooks(t *testing.T) {
	body := stripeEventFixture(t)
	now := time.Now()

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
			signingTime:   now,
		},
		{
			name:             "missing signature",
			signingSecret:    testSigningSecret,
			signingTime:      now,
			missingSignature: true,
		},
		{
			name:          "tampered body",
			signingSecret: testSigningSecret,
			signingTime:   now,
			tamperBody:    true,
		},
		{
			name:          "expired signature",
			signingSecret: testSigningSecret,
			signingTime:   now.Add(-time.Hour),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := make(http.Header)
			if !tt.missingSignature {
				signed := webhook.GenerateTestSignedPayload(&webhook.UnsignedPayload{
					Payload:   body,
					Secret:    tt.signingSecret,
					Timestamp: tt.signingTime,
				})
				headers.Set("Stripe-Signature", signed.Header)
			}

			requestBody := body
			if tt.tamperBody {
				requestBody = bytes.Replace(
					body,
					[]byte("evt_test_123"),
					[]byte("evt_test_456"),
					1,
				)
			}

			verifier := &StripeVerifier{EndpointSecret: testSigningSecret}
			event, err := verifier.Verify(context.Background(), headers, requestBody)

			require.Error(t, err)
			assert.Empty(t, event.ProviderEventID)
			assert.Empty(t, event.RawPayload)
		})
	}
}

func TestVerifiedEvent_HashPayload(t *testing.T) {
	tests := []struct {
		name    string
		payload []byte
		want    string
	}{
		{
			name: "nil payload",
			want: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name:    "empty payload",
			payload: []byte{},
			want:    "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name:    "nonempty payload",
			payload: []byte("abc"),
			want:    "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := signatures.VerifiedEvent{RawPayload: tt.payload}
			assert.Equal(t, tt.want, event.HashPayload())
		})
	}
}
