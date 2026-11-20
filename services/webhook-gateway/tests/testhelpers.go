package testhelpers

import (
	"accord/services/webhook-gateway/internal/http/webhooks"
	"accord/services/webhook-gateway/internal/signatures"
	"bytes"
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	stripesdk "github.com/stripe/stripe-go/v87"
	"github.com/stripe/stripe-go/v87/webhook"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

type PostgresContainer struct {
	*postgres.PostgresContainer
	ConnectionString string
}

func CreatePostgresContainer(ctx context.Context) (*PostgresContainer, error) {
	pgContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:15.3-alpine"),
		postgres.WithInitScripts(filepath.Join("..", "testdata", "init-db.sql")),
		postgres.WithDatabase("test-db"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(5*time.Second)),
	)
	if err != nil {
		return nil, err
	}
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, err
	}

	return &PostgresContainer{
		PostgresContainer: pgContainer,
		ConnectionString:  connStr,
	}, nil
}

const (
	TestSigningSecret     = "whsec_test_secret"
	TestEventID           = "evt_test_123"
	TestEventType         = "payment_intent.succeeded"
	TestStripeWebhookPath = "/v1/stripe"
)

type HandlerTestVerifier struct {
	mock.Mock
	CapturedReasons []signatures.RejectionReasonType
}

func (m *HandlerTestVerifier) Verify(
	ctx context.Context,
	headers http.Header,
	body []byte,
) (signatures.VerifiedEvent, error) {
	args := m.Called(ctx, headers, body)
	return args.Get(0).(signatures.VerifiedEvent), args.Error(1)
}

func (m *HandlerTestVerifier) CaptureRejection(
	body []byte,
	request http.Request,
	receivedAt time.Time,
	ctx context.Context,
	correlationID uuid.UUID,
	reason signatures.RejectionReasonType,
	details string,
) (signatures.WebhookRejection, error) {
	m.CapturedReasons = append(m.CapturedReasons, reason)
	args := m.Called(body,
		request, receivedAt, ctx,
		correlationID, reason, details)
	return args.Get(0).(signatures.WebhookRejection), args.Error(1)
}

type MockReceiptRepository struct {
	mock.Mock
}

func (m *MockReceiptRepository) RecordEvent(
	ctx context.Context,
	receipt *signatures.WebhookReceipt,
) error {
	args := m.Called(ctx, receipt)
	return args.Error(0)
}

type MockRejectionRepository struct {
	mock.Mock
}

func (m *MockRejectionRepository) LogRejection(
	ctx context.Context,
	rejection *signatures.WebhookRejection,
) error {
	args := m.Called(ctx, rejection)
	return args.Error(0)
}

type HandlerFixture struct {
	Handler    *webhooks.StripeHandler
	Receipts   *MockReceiptRepository
	Rejections *MockRejectionRepository
}

func NewHandlerFixture(
	t *testing.T,
	verifier signatures.WebhookVerifier,
	maxBodyBytes int64,
) *HandlerFixture {
	t.Helper()

	receipts := &MockReceiptRepository{}
	rejections := &MockRejectionRepository{}

	receipts.Test(t)
	rejections.Test(t)

	t.Cleanup(func() {
		receipts.AssertExpectations(t)
		rejections.AssertExpectations(t)
	})

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler, err := webhooks.NewStripeHandler(
		receipts,
		rejections,
		verifier,
		maxBodyBytes,
		logger,
	)
	require.NoError(t, err, "create handler")

	return &HandlerFixture{
		Handler:    handler,
		Receipts:   receipts,
		Rejections: rejections,
	}
}

func NewTestVerifier(t *testing.T) *HandlerTestVerifier {
	t.Helper()

	verifier := &HandlerTestVerifier{}
	verifier.Test(t)
	t.Cleanup(func() {
		verifier.AssertExpectations(t)
	})
	return verifier
}

func StripeEventFixture(t *testing.T) []byte {
	t.Helper()

	body, err := json.Marshal(map[string]any{
		"id":          TestEventID,
		"object":      "event",
		"api_version": stripesdk.APIVersion,
		"type":        TestEventType,
		"created":     int64(1700000000),
		"data": map[string]any{
			"object": map[string]any{
				"id":     "pi_test_123",
				"object": "payment_intent",
			},
		},
	})
	require.NoError(t, err, "create Stripe fixture")
	return body
}

func NewWebhookRequest(body []byte) *http.Request {
	request := httptest.NewRequest(
		http.MethodPost,
		TestStripeWebhookPath,
		bytes.NewReader(body),
	)
	request.Header.Set("Content-Type", "application/json")
	return request
}

func NewSignedWebhookRequest(body []byte) *http.Request {
	signed := webhook.GenerateTestSignedPayload(&webhook.UnsignedPayload{
		Payload:   body,
		Secret:    TestSigningSecret,
		Timestamp: time.Now(),
	})

	request := NewWebhookRequest(body)
	request.Header.Set("Stripe-Signature", signed.Header)
	return request
}
