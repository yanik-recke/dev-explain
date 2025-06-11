package main

import (
	"context"
	"log"
	"os"

	"github.com/joho/godotenv"
	application "github.com/yanik-recke/devexplain/internal/app"
)

func main() {
	err := godotenv.Load()

	if err != nil {
		log.Fatalf("error loading .env file: %v", err)
	}

	log.Printf("Starting server")

	app := application.New(
		os.Getenv("OLLAMA_BASE"),
		os.Getenv("CHAT_ENDPOINT"),
		os.Getenv("EMBED_MODEL"),
		os.Getenv("CHAT_MODEL"),
		os.Getenv("GITHUB_TOKEN"),
		os.Getenv("INTENT_URL"),
		os.Getenv("HEALTH_URL"),
	)

	err = app.Start(context.TODO())

	if err != nil {
		log.Fatalf("Error during server start up, aborting...")
	}
}
