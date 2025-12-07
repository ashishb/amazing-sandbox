package main

import (
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func npxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "npx",
		Short: "Run an npx command",
	}

	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Error getting current working directory")
	}

	directory := cmd.PersistentFlags().StringP("directory", "d", cwd, "Working directory for this command")
	cmd.Run = func(cmd *cobra.Command, npxArgs []string) {
		log.Info().
			Ctx(cmd.Context()).
			Str("directory", *directory).
			Strs("args", npxArgs).
			Msg("Running npx command")
		// Implement the logic to run npx command here
	}
	return cmd
}
