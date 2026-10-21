package signatures

import (
	"accord/services/webhook-gateway/internal/signatures"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm/logger"
)

type RejectionReasonType string


const (
	RejectionReasonTypeMissingSignature RejectionReasonType = "NO_SIGNATURE"

	RejectionReasonTypeInvaildSignature RejectionReasonType = "BAD_SIGNATURE"

	RejectionReasonTypeExpiredSignature RejectionReasonType = "OLD_SIGNATURE"

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
    SanitizedPayload   *string `gorm:"type:text"`

    PayloadSHA256    string
    PayloadSizeBytes int64
    PayloadFormat    string

    SanitizerVersion int

    SourceIP         string  `gorm:"type:inet;not null"`
    RejectionDetails *string `gorm:"type:text"`
}

func (WebhookRejection) TableName() string {
    return "webhook_rejections"

}
func NewWebhookRejection (r http.Request, ctx context.Context, verifier signatures.WebhookVerifier, correlationID uuid.UUID, reasonRejected RejectionReasonType) (WebhookRejection, error) {
	convertedHeader, err := json.Marshal(r.Header)
	if err != nil {
		logger.Error(fmt.Sprintf("unable to parse headers for rejection: %s", err))
		return nil, err
	}
	new_rejection =  WebhookRejection{
		ID: uuid.New(),
		CorrelationID: correlationID,
		ReceivedAt: time.Now().UTC(),
		RejectionReason: reasonRejected,
		RequestPath: r.URL.Path,
		RequestHeaders: convertedHeader,

	}
}
