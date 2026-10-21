package main

import (
	"accord/services/webhook-gateway/internal/http/webhooks"
	"accord/services/webhook-gateway/internal/journal"
	"accord/services/webhook-gateway/internal/signatures"
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"log/slog"
	"net/http"
	"os"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil)) //

	// Allow production to supply environment variables without a .env file.
	if err := godotenv.Load("/workspaces/accord/.env"); err != nil {
		// This will print the exact reason (e.g., file not found or permission denied)
		logger.Error(fmt.Sprintf("load .env: %v", err))
		os.Exit(1)
	}
	_, StripeEndpointSecretExists := os.LookupEnv("STRIPE_ENDPOINT_SECRET")
	if !StripeEndpointSecretExists {
		logger.Error("REQUIRED ENV VAR `STRIPE_ENDPOINT_SECRET` NOT SET...Exiting... ")
		os.Exit(1)

	}

	_, DatabaseDSNExists := os.LookupEnv("DATABASE_DSN")
	if !DatabaseDSNExists {
		logger.Error("REQUIRED ENV VAR `DATABASE_DSN` NOT SET...Exiting... ")
		os.Exit(1)
	}
	GatewayDBConnection, err := journal.NewRepository(os.Getenv("DATABASE_DSN"))
	if err != nil {
		logger.Error(fmt.Sprintf("unable to establish database connection: %v", err))
		os.Exit(1)
	}
	GatewayMux := http.NewServeMux()
	stripeHandler, err := webhooks.NewStripeHandler(
		GatewayDBConnection,
		&signatures.StripeVerifier{EndpointSecret: os.Getenv("STRIPE_ENDPOINT_SECRET")},
		2<<20,
		logger,
	)
	if err != nil {
		logger.Error(fmt.Sprintf("unable to create stripe handler: %v", err))
		os.Exit(1)
	}
	GatewayMux.HandleFunc("POST /v1/stripe", stripeHandler.CatchWebhook)

	log.Println("Server starting on :8080...")
	if err := http.ListenAndServe(":8080", GatewayMux); err != nil {
		logger.Error(fmt.Sprintf("Server failed: %v", err))
		os.Exit(1)
	}

}
