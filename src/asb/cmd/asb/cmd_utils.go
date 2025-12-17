package main

import (
	"os"
	"slices"
	"strings"

	"github.com/ashishb/asb/src/asb/internal/cmdrunner"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
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

func getCmdConfig(cmd *cobra.Command, cwd string) []cmdrunner.Option {
	return []cmdrunner.Option{
		cmdrunner.SetWorkingDir(cwd),
		cmdrunner.SetArgs(getCmdArgs(cmd)),
		cmdrunner.SetMountWorkingDirReadWrite(true),
		cmdrunner.SetMountReferencedDirReadWrite(true),
		cmdrunner.SetRunAsNonRoot(true),
		cmdrunner.SetNetworkType(cmdrunner.NetworkHost),
	}
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
