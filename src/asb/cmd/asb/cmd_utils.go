package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/ashishb/asb/src/asb/internal/cmdrunner"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func createCmd(cmd *cobra.Command, f func(options ...cmdrunner.Option) cmdrunner.Config) *cobra.Command {
	cmd.FParseErrWhitelist.UnknownFlags = true
	cmd.Run = func(cmd *cobra.Command, args []string) {
		directory := getStringFlagOrFail(cmd, "directory")
		enableNetwork := !getBoolFlagOrFail(cmd, "no-network")
		log.Debug().
			Ctx(cmd.Context()).
			Str("name", cmd.Name()).
			Str("directory", directory).
			Strs("args", args).
			Msg("Running command")

		options := getCmdConfig(cmd, directory, enableNetwork)
		config := f(options...)
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

func getCwdOrFail() string {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Error getting current working directory")
	}
	return cwd
}

func getStringFlagOrFail(cmd *cobra.Command, name string) string {
	value, err := cmd.Flags().GetString(name)
	if err != nil {
		log.Fatal().
			Err(err).
			Str("flagName", name).
			Msg("Failed to fetch flag")
	}
	return value
}

func getBoolFlagOrFail(cmd *cobra.Command, name string) bool {
	value, err := cmd.Flags().GetBool(name)
	if err != nil {
		log.Fatal().
			Err(err).
			Str("flagName", name).
			Msg("Failed to fetch flag")
	}
	return value
}

func getCmdConfig(cmd *cobra.Command, cwd string, enableNetwork bool) []cmdrunner.Option {
	envFile := filepath.Join(cwd, ".env")
	envFileExists := false
	if fileInfo, _ := os.Stat(envFile); fileInfo != nil && !fileInfo.IsDir() {
		log.Debug().
			Ctx(cmd.Context()).
			Str("envFile", envFile).
			Msg(".env file found, will be loaded inside the sandbox")
		envFileExists = true
	}

	options := []cmdrunner.Option{
		cmdrunner.SetWorkingDir(cwd),
		cmdrunner.SetArgs(getCmdArgs(cmd)),
		cmdrunner.SetMountWorkingDirReadWrite(true),
		cmdrunner.SetMountReferencedDirReadWrite(true),
		cmdrunner.SetRunAsNonRoot(true),
	}

	if enableNetwork {
		options = append(options, cmdrunner.SetNetworkType(cmdrunner.NetworkHost))
	} else {
		options = append(options, cmdrunner.SetNetworkType(cmdrunner.NetworkNone))
	}
	if envFileExists {
		options = append(options, cmdrunner.SetLoadDotEnv(true))
	}
	return options
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
