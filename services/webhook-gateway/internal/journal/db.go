package journal

import (
	"accord/services/webhook-gateway/internal/signatures"
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func NewRepository(dsn string) (*signatures.GORMWebhookReceiptRepository, error) {
	db, err := gorm.Open(
		postgres.Open(dsn),
		&gorm.Config{},
	)
	if err != nil {
		return nil, fmt.Errorf("unable to establish connection to DB: %w", err)
	}
	return &signatures.GORMWebhookReceiptRepository{DB: db}, nil
}
