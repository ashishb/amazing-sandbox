package main

import (
	"github.com/ashishb/asb/src/asb/internal/cmdrunner"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func npxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "npx",
		Short: "Run an npx command",
	}

	directory := cmd.PersistentFlags().StringP("directory", "d", getCwdOrFail(), "Working directory for this command")
	cmd.Run = func(cmd *cobra.Command, npxArgs []string) {
		log.Info().
			Ctx(cmd.Context()).
			Str("directory", *directory).
			Strs("args", npxArgs).
			Msg("Running npx command")

		options := getCmdConfig(cmd, *directory)
		config := cmdrunner.NewNpxCmdConfig(options...)
		err := cmdrunner.RunCmd(cmd.Context(), config)
		if err != nil {
			log.Fatal().
				Ctx(cmd.Context()).
				Err(err).
				Msg("Error running command")
		}
	}
	return cmd
}
