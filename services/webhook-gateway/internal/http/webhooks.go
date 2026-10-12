package webhooks

import (
	"context"
	"io"
	"net/http"

	"accord/services/webhook-gateway/internal/signatures"
	"github.com/google/uuid"
)



func NewWebhookHandler(
	verifier signatures.WebhookVerifier,
	receipts signatures.WebhookReceiptRepository,
) *WebhookHandler {
	return &WebhookHandler{
		verifier: verifier,
		receipts: receipts,
	}
}
type EventJournal interface {
	Accept(
		ctx context.Context,
		event signatures.VerifiedEvent,
	) (Acceptance, error)
}

type Acceptance struct {
	EventID   uuid.UUID // Accord-owned, persisted event ID.
	Duplicate bool
}

GatewayMux := http.NewServeMux()


// Handler Functions
func (h *Handler) CatchStripeWebhook(
	w http.ResponseWriter
	r *http.Request
){
	body, err := io.Readall(r.body)
	if err != nil {
		http.Error(w, "unable to read body", http.StatusBadRequest)
		return
	}
	event, err := h.ve

}



// Routes
mux.HandlerFunc(
	"POST /gateway/stripe",


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


