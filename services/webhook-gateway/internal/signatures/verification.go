package signatures

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// Top-Level Type  and Variable Definitions
type Provider int

const (
	Stripe Provider = iota
	Shippo
	Zendesk
	Plaid
	Other
)

type VerifiedEvent struct {
	Provider        Provider
	ProviderEventID string // May be absent for some providers.
	EventType       string
	ReceivedAt      time.Time
	Payload         json.RawMessage
}

type Verifer interface {
	Verify(
		ctx context.Context,
		headers http.Header,
		body []byte,
	) (VerifiedEvent, error)
}
