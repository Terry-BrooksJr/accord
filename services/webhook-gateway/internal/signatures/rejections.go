package signatures

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"
)

var MaxStoredPayloadBytes int

func init() {
	var err error
	MaxStoredPayloadBytes, err = strconv.Atoi(os.Getenv("REJECTED_BYTES_STORAGE_LIMIT"))
	if err != nil {
		slog.Error("unable to convert REJECTED_BYTES_STORAGE_LIMIT env var to int. Setting default to 16KiB")
		MaxStoredPayloadBytes = 16384
	}
	if MaxStoredPayloadBytes <= 0 {
		MaxStoredPayloadBytes = 16384

	}
}

func HashRequestBody(body []byte) string {
	hash := sha256.Sum256(body)
	return hex.EncodeToString(hash[:])
}

func extract_metadata(r http.Request) (ip string, extractedHeaders map[string][]string) {
	// Get source IP by calling the GetClientIP function defined in the signatureds package
	isBehindProxy, err := strconv.ParseBool(os.Getenv("USING_PROXY"))
	extractedHeaders = make(map[string][]string)

	if err != nil {
		slog.Error("unable to parse env for BEHIND_PROXY env var. Setting to false")
		isBehindProxy = false
	}
	ip, err = GetClientIP(&r, isBehindProxy)
	if err != nil {
		slog.Error("unable to get source ip", "error", err)
		ip = "0.0.0.0"
	}

	// Iterate Over All the Headers
	for name, values := range r.Header {
		for _, value := range values {
			extractedHeaders[name] = append(extractedHeaders[name], value)
		}
	}
	return ip, extractedHeaders
}

type RejectionReasonType string

const (
	RejectionReasonTypeMissingSignature RejectionReasonType = "NO_SIGNATURE"

	RejectionReasonTypeInvaildSignature RejectionReasonType = "BAD_SIGNATURE"

	RejectionReasonTypeExpiredSignature RejectionReasonType = "OLD_SIGNATURE"

	RejectionReasonTypeMishapedSignature RejectionReasonType = "MISHAPEN_SIGNATURE"

	RejectionReasonTypeInvalidPayload RejectionReasonType = "BAD_PAYLOAD"

	RejectionReasonTypePayloadTooLarge RejectionReasonType = "OVERSIZED_PAYLOAD"

	RejectionReasonTypeUnsupportedContentType RejectionReasonType = "BAD_CONTENT_TYPE"

	RejectionReasonTypeBodyReadFailure RejectionReasonType = "UNABLE_TO_PARSE"
)

type WebhookRejection struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey"`
	CorrelationID uuid.UUID `gorm:"type:uuid;not null;index"`

	Provider        Provider            `gorm:"not null;index"`
	ReceivedAt      time.Time           `gorm:"column:received_at;not null;index"`
	RejectionReason RejectionReasonType `gorm:"not null"`

	RequestPath    string
	RequestHeaders json.RawMessage `gorm:"type:jsonb;not null"`

	IsPayloadTruncated bool
	CapturedPayload    []byte `gorm:"type:bytea"`
	PayloadSHA256      *string

	SourceIP         string  `gorm:"type:inet;not null"`
	RejectionDetails *string `gorm:"type:text"`
}

func (WebhookRejection) TableName() string {
	return "webhook_rejections"

}

func NewWebhookRejection(r http.Request,
	body []byte,
	timeReceived time.Time,
	ctx context.Context,
	correlationID uuid.UUID,
	reasonRejected RejectionReasonType,
	truncated bool,
	payload []byte,
	details *string,
) WebhookRejection {
	ip, extractedHeaders := extract_metadata(r)
	encodedHash := HashRequestBody(body)
	jsonHeaders, err := json.Marshal(extractedHeaders)
	if err != nil {
		slog.Warn("unable to marshall headers", "error", err)
	}
	new_rejection := WebhookRejection{
		ID:                 uuid.New(),
		Provider:           Stripe,
		CorrelationID:      correlationID,
		ReceivedAt:         timeReceived,
		RejectionReason:    reasonRejected,
		RequestPath:        r.URL.Path,
		RequestHeaders:     jsonHeaders,
		IsPayloadTruncated: truncated,
		CapturedPayload:    payload,
		PayloadSHA256:      &encodedHash,
		SourceIP:           ip,
		RejectionDetails:   details,
	}
	return new_rejection

}
func (r *GORMWebhookRejectionRepository) LogRejection(
	ctx context.Context,
	rejection *WebhookRejection,
) error {
	return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Create(&rejection)
		if result.Error != nil {
			slog.Warn("unable to insert row into rejection loa", "error", result.Error, "rejection_data", rejection)
			return result.Error
		}
		if result.RowsAffected == 1 {
			slog.Info("New Rejection Logged Succcessfully", "rejection", rejection.ID)
			return nil
		}
		return errors.New("Unable to Log Rejection. ")
	})
}
