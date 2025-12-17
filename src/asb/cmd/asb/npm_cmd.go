package main

import (
	"os"
	"slices"
	"strings"

	"github.com/ashishb/asb/src/asb/internal/cmdrunner"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func npmCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "npm",
		Short: "Run an npm command",
	}
	cmd.FParseErrWhitelist.UnknownFlags = true

	directory := cmd.PersistentFlags().StringP("directory", "d", getCwdOrFail(), "Working directory for this command")
	cmd.Run = func(cmd *cobra.Command, args []string) {
		log.Info().
			Ctx(cmd.Context()).
			Str("directory", *directory).
			Strs("args", args).
			Msg("Running npm command")

		config := cmdrunner.NewNpmCmdConfig(
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

func getCmdArgs(cmd *cobra.Command) []string {
	i1 := slices.Index(os.Args, cmd.Use)
	if i1 == -1 {
		log.Fatal().
			Ctx(cmd.Context()).
			Msgf("Could not find command %q in args %q", cmd.Use, strings.Join(os.Args, " "))
	}

	// Skip the first two args (program name, "npm" command)
	cmdArgs := os.Args[i1+1:]
	return cmdArgs
}
