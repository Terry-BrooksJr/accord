package signatures

import (
	"accord/services/webhook-gateway/internal/signatures"
	"context"
	"fmt"
	"github.com/google/uuid"
	"log/slog"
	"net/http"
	"time"
)

func (sh *StripeVerifier) CaptureRejection(
	body []byte,
	r http.Request,
	timeReceived time.Time,
	ctx context.Context,
	correlationID uuid.UUID,
	reasonRejected signatures.RejectionReasonType,
	details string,
) (signatures.WebhookRejection, error) {

	if len(body) > signatures.MaxStoredPayloadBytes {
		slog.Error("rejected http request body over rejection limits. discarding the body.")
		details += fmt.Sprintf("Payload was over the rejection payload retention amount of %d bytes", signatures.MaxStoredPayloadBytes)

		return signatures.NewWebhookRejection(
			r,
			body,
			timeReceived,
			ctx,
			correlationID,
			reasonRejected,
			false,
			nil,
			&details,
		), nil
	}
	return signatures.NewWebhookRejection(
		r,
		body,
		timeReceived,
		ctx,
		correlationID,
		reasonRejected,
		false,
		body,
		&details,
	), nil

}
