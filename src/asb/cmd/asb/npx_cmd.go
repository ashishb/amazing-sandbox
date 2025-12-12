package main

import (
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/ashishb/asb/src/asb/internal/cmdrunner"
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

		config := cmdrunner.NewNpxCmdConfig(
			cmdrunner.SetWorkingDir(*directory),
			cmdrunner.SetArgs(npxArgs),
			cmdrunner.SetMountWorkingDirReadWrite(true),
			cmdrunner.SetRunAsNonRoot(true),
			cmdrunner.SetNetworkType(cmdrunner.NetworkHost),
		)

		err := cmdrunner.RunNpxCmd(cmd.Context(), config)
		if err != nil {
			log.Fatal().
				Ctx(cmd.Context()).
				Err(err).
				Msg("Error running npx command")
		}
	}
	return cmd
}
