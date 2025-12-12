package main

import (
	"os"

	"github.com/ashishb/asb/src/asb/internal/cmdrunner"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func gemExecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gem-exec",
		Short: "Run a gem already inside sandbox",
	}
	cmd.FParseErrWhitelist.UnknownFlags = true

	directory := cmd.PersistentFlags().StringP("directory", "d", getCwdOrFail(),
		"Working directory for this command")
	cmd.Run = func(cmd *cobra.Command, args []string) {
		log.Info().
			Ctx(cmd.Context()).
			Str("directory", *directory).
			Strs("args", args).
			Msg("Running gem that is already inside sandbox")

		// Skip the first two args (program name, "gem-exec" command_
		cmdArgs := os.Args[2:]
		config := cmdrunner.NewRubyGemExecCmdConfig(
			cmdrunner.SetWorkingDir(*directory),
			cmdrunner.SetArgs(cmdArgs),
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
