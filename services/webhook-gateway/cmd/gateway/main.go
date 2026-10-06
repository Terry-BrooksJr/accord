package main

import (
	"errors"
	"log"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	// Allow production to supply environment variables without a .env file.
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Fatalf("load .env: %v", err)
	}

}
