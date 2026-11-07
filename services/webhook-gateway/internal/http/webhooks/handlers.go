package webhooks

import (
	"accord/services/webhook-gateway/internal/signatures"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"io"
	"log/slog"
	"net/http"
	"time"
)

func NewStripeHandler(
	repo signatures.WebhookReceiptRepository,
	rejectionLog signatures.GORMWebhookRejectionRepository,
	verifier signatures.WebhookVerifier,
	maxBodyBytes int64,
	logger *slog.Logger,
) (*StripeHandler, error) {
	if repo == nil {
		return nil, errors.New("repo cannot be of nil value")
	}
	if rejectionLog.DB == nil {
		return nil, errors.New("rejection log cannot be of nil value")
	}
	if verifier == nil {
		return nil, errors.New("verifier cannot be of nil value")
	}
	if maxBodyBytes <= 0 {
		maxBodyBytes = 1 << 20
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &StripeHandler{
		Repo:         repo,
		RejectionLog:  rejectionLog,
		Verifier:     verifier,
		MaxBodyBytes: maxBodyBytes,
		Logger:       logger,
		now:          time.Now,
	}, nil
}

type StripeHandler struct {
	Repo         signatures.WebhookReceiptRepository
	RejectionLog signatures.GORMWebhookRejectionRepository
	Verifier     signatures.WebhookVerifier
	MaxBodyBytes int64
	Logger       *slog.Logger
	now          func() time.Time
}

// Handler Functions
func (sh *StripeHandler) CatchWebhook(
	w http.ResponseWriter,
	r *http.Request,
) {
	currentTime := sh.now()
	ctx := r.Context()
	incomingRequestCorrelationID := uuid.New()
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, sh.MaxBodyBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			rejection, err := sh.Verifier.CaptureRejection(
				body,
				*r,
				currentTime,
				ctx,
				incomingRequestCorrelationID,
				signatures.RejectionReasonTypePayloadTooLarge,
				"Incoming Request Body Is Over the Set Threshold.",
			)
			if err != nil {
				sh.Logger.ErrorContext(ctx, "unable to capture rejection", "error", err)
			}
			rejectionLoggingIssue := sh.RejectionLog.LogRejection(ctx, &rejection)
			if rejectionLoggingIssue != nil {
				sh.Logger.ErrorContext(ctx, "unable to log rejection", "error", err)

			}
			http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)

			return
		}
		http.Error(w, "Unable to read request's body", http.StatusBadRequest)
		return

	}

	incomingEvent, err := sh.Verifier.Verify(ctx, r.Header, body)

	if err != nil {
		var rejectionReason signatures.RejectionReasonType
		var rejectionDetails string
		if errors.Is(err, signatures.ErrMissingStripeSignature) {
			rejectionReason = signatures.RejectionReasonTypeMissingSignature
			rejectionDetails = "Incoming Request failed to provide Stripe Signature in HTTP Headers"
		}
		sh.Logger.WarnContext(ctx, fmt.Sprintf("warning: unverifiable stripe webhook recieved: %s", err), "correlationID", incomingRequestCorrelationID)
		rejectionReason = signatures.RejectionReasonTypeInvaildSignature
		rejectionDetails = "Incoming Request Cannot Be Verifed and Deemed Invaild"

		rejection, err := sh.Verifier.CaptureRejection(
			body,
			*r,
			currentTime,
			ctx,
			incomingRequestCorrelationID,
			rejectionReason,
			rejectionDetails,
		)
		if err != nil {
			sh.Logger.ErrorContext(ctx, "unable to capture rejection", "error", err)
		}
		rejectionLoggingIssue := sh.RejectionLog.LogRejection(ctx, &rejection)
		if rejectionLoggingIssue != nil {
			sh.Logger.ErrorContext(ctx, "unable to log rejection", "error", err)

		}
		http.Error(w, "unable to verify webhook authenticity", http.StatusUnauthorized)
		return
	}
	receipt := signatures.NewWebhookReceipt(incomingEvent, signatures.ReceiptStatusAccepted, currentTime, incomingRequestCorrelationID)
	err = sh.Repo.RecordEvent(ctx, &receipt)
	if err != nil {
		sh.Logger.ErrorContext(ctx, fmt.Sprintf("db_write_Unable to record an event to databae: %s", err), "correlationID", incomingRequestCorrelationID)
		http.Error(w, "unable catalog event", http.StatusInternalServerError)
		return

	}
	resp := signatures.NewProcessedReciept(receipt)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		sh.Logger.ErrorContext(ctx, fmt.Sprintf("failed to encode Stripe post procerssing responder: %s", err), "correlationID", incomingRequestCorrelationID)
	}
}

// Routes
