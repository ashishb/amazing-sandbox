package main

import (
	"github.com/ashishb/asb/src/asb/internal/cmdrunner"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func yarnCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "yarn",
		Short: "Run a yarn command",
	}
	cmd.FParseErrWhitelist.UnknownFlags = true

	directory := cmd.PersistentFlags().StringP("directory", "d", getCwdOrFail(), "Working directory for this command")
	cmd.Run = func(cmd *cobra.Command, args []string) {
		log.Info().
			Ctx(cmd.Context()).
			Str("directory", *directory).
			Strs("args", args).
			Msg("Running command")

		config := cmdrunner.NewYarnCmdConfig(
			cmdrunner.SetWorkingDir(*directory),
			cmdrunner.SetArgs(getCmdArgs(cmd)),
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
