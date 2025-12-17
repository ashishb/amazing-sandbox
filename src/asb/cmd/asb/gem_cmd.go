package main

import (
	"github.com/ashishb/asb/src/asb/internal/cmdrunner"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func gemCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gem",
		Short: "Run a Ruby gem-based CLI tool",
	}

	directory := cmd.PersistentFlags().StringP("directory", "d", getCwdOrFail(),
		"Working directory for this command")
	cmd.Run = func(cmd *cobra.Command, args []string) {
		log.Info().
			Ctx(cmd.Context()).
			Str("directory", *directory).
			Strs("args", args).
			Msg("Running Ruby gem-based command")

		options := getCmdConfig(cmd, *directory)
		config := cmdrunner.NewRubyGemCmdConfig(options...)
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
