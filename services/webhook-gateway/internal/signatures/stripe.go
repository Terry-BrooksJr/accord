package signatures

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/stripe/stripe-go/v87"
)

var sc *stripe.Client = stripe.NewClient(os.Getenv("STRIPE_API_KEY"))

type StripeVerifer struct {
	EndpointSecret string
}

func (sv *StripeVerifer) Verify(
	ctx context.Context,
	headers http.Header,
	body []byte,
) (VerifiedEvent, error) {
	signature := headers.Get("Stripe-Signature")
	eventData, err := sc.ConstructEvent(body, signature, sv.EndpointSecret)
	if err != nil {
		slog.Error("ERROR: Unable to Verify Stripe Webhood", "err", err)
		return VerifiedEvent{}, errors.New("Unable to Construct Event During Strip Signature Verififcation. ")
	}
	event := VerifiedEvent{
		Provider:        0,
		ProviderEventID: eventData.ID,
		EventType:       string(eventData.Type),
		ReceivedAt:      time.Now(),
		Payload:         json.RawMessage(eventData.Object),
	}
	return event, nil
}
