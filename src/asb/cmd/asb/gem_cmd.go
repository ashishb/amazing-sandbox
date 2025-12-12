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
	cmd.Run = func(cmd *cobra.Command, npxArgs []string) {
		log.Info().
			Ctx(cmd.Context()).
			Str("directory", *directory).
			Strs("args", npxArgs).
			Msg("Running Ruby gem-based command")

		config := cmdrunner.NewRubyGemCmdConfig(
			cmdrunner.SetWorkingDir(*directory),
			cmdrunner.SetArgs(npxArgs),
			cmdrunner.SetMountWorkingDirReadWrite(true),
			cmdrunner.SetRunAsNonRoot(true),
			cmdrunner.SetNetworkType(cmdrunner.NetworkHost),
		)

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
