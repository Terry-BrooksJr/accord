package journal

import (
	"log"
	"gorm.io/gorm"
	"gorm.io/driver/postgres"
)

dsn := "host=127.19.0.2 user=go_agent password=zhb*ycd0rvc5zmz9YUG dbname=accord port=5432 sslmode=disable"
db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
