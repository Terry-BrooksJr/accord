package webhooks

import (
	"fmt"
)




type VerifiedEvent struct {
	Provider        Provider
	ProviderEventID string // May be absent for some providers.
	EventType       string
	ReceivedAt      time.Time
	Payload         json.RawMessage
}

type Verifier interface {
	Verify(
		ctx context.Context,
		headers http.Header,
		body []byte,
	) (VerifiedEvent, error)
}

type EventJournal interface {
	Accept(
		ctx context.Context,
		event VerifiedEvent,
	) (Acceptance, error)
}

type Acceptance struct {
	EventID   uuid.UUID // Accord-owned, persisted event ID.
	Duplicate bool
}

mux := http.NewServeMux()


// Handler Functions 
func (h *Handler) Handle(
	w http.ResponseWriter
	r *http.Request
){}



// Routes
mux.HandlerFunc(
	"POST /gateway/{provider}",

)

mux.HandlerFunc(
	"GET /gateway/health",
	
)

mux.HandlerFunc(
	"GET /gateway/readyt",
	
)


mux.HandlerFunc(
	"GET /gateway/metrics",
	
)


