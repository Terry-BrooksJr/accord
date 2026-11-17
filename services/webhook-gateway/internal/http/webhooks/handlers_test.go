package webhooks_test

import (
	"accord/services/webhook-gateway/internal/signatures"
	stripeprovider "accord/services/webhook-gateway/internal/signatures/providers/stripe"
	helpers "accord/services/webhook-gateway/tests"
	"bytes"
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

func TestCatchWebhookReturns401WhenRejectionSaveFails(t *testing.T) {
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

	fixture.Rejections.On(
		"LogRejection",
		mock.Anything,
		mock.MatchedBy(func(rejection *signatures.WebhookRejection) bool {
			return rejection != nil &&
				rejection.RejectionReason ==
					signatures.RejectionReasonTypeMissingSignature
		}),
	).Return(errors.New("Non=specific Databaser Error")).Once()

	request := helpers.NewSignedWebhookRequest(body)
	response := httptest.NewRecorder()

	fixture.Handler.CatchWebhook(response, request)

	assert.Equal(t, http.StatusUnauthorized, response.Code)

}
