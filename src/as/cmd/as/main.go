package main

import (
	"fmt"
	"os"

	"github.com/ashishb/as/src/as/internal/logger"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
)

func main() {
	logger.ConfigureLogging()
	loadDotEnv()

	log.Trace().
		Msg("This is the 'as' command.")
	if err := getRootCmd().Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func loadDotEnv() {
	// Check if .env file exists
	if _, err := os.Stat(".env"); os.IsNotExist(err) {
		log.Debug().
			Msg("No .env file found, skipping loading environment variables")

		return
	}

	err := godotenv.Load()
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Error loading .env file")
	}

	log.Info().
		Msg("Environment variables loaded from .env file")
}
