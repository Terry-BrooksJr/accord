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

// ErrMissingStripeSignature is raised when an incoming HTTP Request's headers has a nil value in or  missing the `Stripe-Signature`. TODO: U
// This error is used in downstream conditional logic and modifies the rejection instatiatioin prior to persisting it in Rejection Log Repositiory.
var ErrMissingPlaidSignature = errors.New("missing plaid signature")

// StripeVerifier verifies Stripe webhook signatures.
type PlaidVerifier struct {
	// EndpointSecret is the signing secret for the Stripe webhook endpoint.
	EndpointSecret string
	EventData      stripe.Event
}
