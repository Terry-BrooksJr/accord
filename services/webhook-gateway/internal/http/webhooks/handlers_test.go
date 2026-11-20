package webhooks_test

import (
	"accord/services/webhook-gateway/internal/signatures"
	stripeprovider "accord/services/webhook-gateway/internal/signatures/providers/stripe"
	helpers "accord/services/webhook-gateway/tests"
	"bytes"
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCatchWebhookRejectsGET(t *testing.T) {
	verifier := helpers.NewTestVerifier(t)
	fixture := helpers.NewHandlerFixture(t, verifier, 8)

	request := httptest.NewRequest(http.MethodGet, helpers.TestStripeWebhookPath, nil)
	response := httptest.NewRecorder()

	fixture.Handler.CatchWebhook(response, request)

	assert.Equal(t, http.StatusMethodNotAllowed, response.Code)
	assert.Empty(t, verifier.CapturedReasons)
	verifier.AssertNumberOfCalls(t, "Verify", 0)
	fixture.Receipts.AssertNumberOfCalls(t, "RecordEvent", 0)
	fixture.Rejections.AssertNumberOfCalls(t, "LogRejection", 0)
}

func TestCatchWebhookRejectsOversizedBody(t *testing.T) {
	verifier := helpers.NewTestVerifier(t)
	fixture := helpers.NewHandlerFixture(t, verifier, 8)

	verifier.On(
		"CaptureRejection",
		[]byte("12345678"),
		mock.Anything, // request
		mock.Anything, // receivedAt
		mock.Anything, // context
		mock.Anything, // correlation ID
		signatures.RejectionReasonTypePayloadTooLarge,
		mock.Anything, // details
	).Return(
		signatures.WebhookRejection{
			RejectionReason: signatures.RejectionReasonTypePayloadTooLarge,
		},
		nil,
	).Once()

	fixture.Rejections.On(
		"LogRejection",
		mock.Anything,
		mock.MatchedBy(func(rejection *signatures.WebhookRejection) bool {
			return rejection != nil &&
				rejection.RejectionReason ==
					signatures.RejectionReasonTypePayloadTooLarge
		}),
	).Return(nil).Once()

	request := helpers.NewWebhookRequest([]byte("123456789"))
	response := httptest.NewRecorder()

	fixture.Handler.CatchWebhook(response, request)

	assert.Equal(t, http.StatusRequestEntityTooLarge, response.Code)
	assert.Equal(t, []signatures.RejectionReasonType{
		signatures.RejectionReasonTypePayloadTooLarge,
	}, verifier.CapturedReasons)

	verifier.AssertNumberOfCalls(t, "Verify", 0)
	fixture.Receipts.AssertNumberOfCalls(t, "RecordEvent", 0)
}

func TestCatchWebhookAllowsBodyAtLimit(t *testing.T) {
	body := helpers.StripeEventFixture(t)
	verifier := &stripeprovider.StripeVerifier{
		EndpointSecret: helpers.TestSigningSecret,
	}
	fixture := helpers.NewHandlerFixture(t, verifier, int64(len(body)))

	fixture.Receipts.On(
		"RecordEvent",
		mock.Anything,
		mock.MatchedBy(func(receipt *signatures.WebhookReceipt) bool {
			return receipt != nil &&
				receipt.Status == signatures.ReceiptStatusPendingNew &&
				receipt.ProviderEventID != nil &&
				*receipt.ProviderEventID == helpers.TestEventID &&
				receipt.EventType == helpers.TestEventType &&
				bytes.Equal(receipt.Payload, body)
		}),
	).Return(nil).Once()

	request := helpers.NewSignedWebhookRequest(body)
	response := httptest.NewRecorder()

	fixture.Handler.CatchWebhook(response, request)

	assert.Equal(
		t,
		http.StatusAccepted,
		response.Code,
		response.Body.String(),
	)
	fixture.Rejections.AssertNumberOfCalls(t, "LogRejection", 0)
}

func TestCatchWebhookRejectsMissingSignature(t *testing.T) {
	body := helpers.StripeEventFixture(t)
	verifier := helpers.NewTestVerifier(t)
	fixture := helpers.NewHandlerFixture(t, verifier, int64(len(body)))

	verifier.On(
		"Verify",
		mock.Anything,
		mock.Anything,
		body,
	).Return(
		signatures.VerifiedEvent{},
		stripeprovider.ErrMissingStripeSignature,
	).Once()

	verifier.On(
		"CaptureRejection",
		body,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		signatures.RejectionReasonTypeMissingSignature,
		mock.Anything,
	).Return(
		signatures.WebhookRejection{
			RejectionReason: signatures.RejectionReasonTypeMissingSignature,
		},
		nil,
	).Once()

	fixture.Rejections.On(
		"LogRejection",
		mock.Anything,
		mock.MatchedBy(func(rejection *signatures.WebhookRejection) bool {
			return rejection != nil &&
				rejection.RejectionReason ==
					signatures.RejectionReasonTypeMissingSignature
		}),
	).Return(nil).Once()

	request := helpers.NewWebhookRequest(body)
	response := httptest.NewRecorder()

	fixture.Handler.CatchWebhook(response, request)

	assert.Equal(t, http.StatusUnauthorized, response.Code)
	assert.Equal(t, []signatures.RejectionReasonType{
		signatures.RejectionReasonTypeMissingSignature,
	}, verifier.CapturedReasons)
	fixture.Receipts.AssertNumberOfCalls(t, "RecordEvent", 0)
}

func TestCatchWebhookReturns500WhenReceiptSaveFails(t *testing.T) {
	body := helpers.StripeEventFixture(t)
	verifier := &stripeprovider.StripeVerifier{
		EndpointSecret: helpers.TestSigningSecret,
	}

	fixture := helpers.NewHandlerFixture(t, verifier, int64(len(body)))
	fixture.Receipts.On(
		"RecordEvent",
		mock.Anything,
		mock.MatchedBy(func(receipt *signatures.WebhookReceipt) bool {
			return receipt != nil &&
				receipt.Status == signatures.ReceiptStatusPendingNew
		}),
	).Return(errors.New("database unavailable")).Once()

	request := helpers.NewSignedWebhookRequest(body)
	response := httptest.NewRecorder()

	fixture.Handler.CatchWebhook(response, request)

	fixture.Rejections.AssertNumberOfCalls(t, "LogRejection", 0)
	assert.Equal(t, http.StatusInternalServerError, response.Code)

}

type FailingReader struct{}

func (f FailingReader) Read(input []byte) (int, error) {
	return 0, errors.New("Intentional Read Failure")
}

func TestCatchWebhookReturns400WhenBodyReadFails(t *testing.T) {
	verifier := helpers.NewTestVerifier(t)
	fixture := helpers.NewHandlerFixture(t, verifier, 8)

	request := httptest.NewRequest(
		http.MethodPost,
		helpers.TestStripeWebhookPath,
		io.NopCloser(FailingReader{}),
	)
	response := httptest.NewRecorder()

	fixture.Handler.CatchWebhook(response, request)

	assert.Equal(t, http.StatusBadRequest, response.Code)
	verifier.AssertNumberOfCalls(t, "Verify", 0)
	verifier.AssertNumberOfCalls(t, "CaptureRejection", 0)
	fixture.Receipts.AssertNumberOfCalls(t, "RecordEvent", 0)
	fixture.Rejections.AssertNumberOfCalls(t, "LogRejection", 0)
}

func TestCatchWebhookHandlesRejectionCaptureFailure(t *testing.T) {
	body := helpers.StripeEventFixture(t)
	verifier := helpers.NewTestVerifier(t)
	fixture := helpers.NewHandlerFixture(t, verifier, int64(len(body)))

	verifier.On(
		"Verify",
		mock.Anything,
		mock.Anything,
		body,
	).Return(
		signatures.VerifiedEvent{},
		stripeprovider.ErrMissingStripeSignature,
	).Once()
	verifier.On(
		"CaptureRejection",
		body,
		mock.Anything, // request
		mock.Anything, // receivedAt
		mock.Anything, // context
		mock.Anything, // correlation ID
		signatures.RejectionReasonTypeMissingSignature,
		mock.Anything, // details
	).Return(
		signatures.WebhookRejection{},
		errors.New("intentional capture failure"),
	).Once()

	request := helpers.NewWebhookRequest(body)
	response := httptest.NewRecorder()

	fixture.Handler.CatchWebhook(response, request)

	assert.Equal(t, http.StatusUnauthorized, response.Code)
	fixture.Receipts.AssertNumberOfCalls(t, "RecordEvent", 0)
	fixture.Rejections.AssertNumberOfCalls(t, "LogRejection", 0)
}
