package main

import (
	"github.com/ashishb/as/src/as/internal/logger"
	"github.com/rs/zerolog/log"
)

func main() {
	logger.ConfigureLogging()
	log.Trace().
		Msg("This is the 'as' command.")
}
