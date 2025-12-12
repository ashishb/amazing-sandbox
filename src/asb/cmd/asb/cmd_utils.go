package main

import (
	"os"

	"github.com/rs/zerolog/log"
)

func getCwdOrFail() string {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Error getting current working directory")
	}
	return cwd
}
