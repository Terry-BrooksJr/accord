package main

import (
	"accord/services/webhook-gateway/internal/webhooks"
	"errors"
	"github.com/joho/godotenv"
	"log/slog"
	"net/http"
	"os"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil)) //

	// Allow production to supply environment variables without a .env file.
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		slog.Fatalf("load .env: %v", err)
	}

	webhooks.GatewayMux.ListenAndServe(":8080", nil)

}
