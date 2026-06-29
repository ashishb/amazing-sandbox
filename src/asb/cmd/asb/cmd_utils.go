package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"syscall"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/ashishb/amazing-sandbox/src/asb/internal/cmdrunner"
)

func createCmd(cmd *cobra.Command, cmdType cmdrunner.CmdType) *cobra.Command {
	cmd.FParseErrWhitelist.UnknownFlags = true

	// This convoluted setup passes help properly to sub-command, "cobra CLI framework"
	// has no good support to handle this
	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		// E.g asb uvx yamllint --help
		log.Debug().
			Ctx(cmd.Context()).
			Str("name", cmd.Name()).
			Strs("args", args).
			Msg("Deprecated: use `gem exec` instead")
		cmd.Run(cmd, args)
	})
	cmd.Run = func(cmd *cobra.Command, args []string) {
		options := getCmdConfig(cmd, args)
		cfg := cmdrunner.NewConfig(cmdType, options...)
		shellResult, err := cmdrunner.RunCmd(cmd.Context(), cfg)
		if err == nil {
			return
		}

		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode := exitErr.ExitCode()
			// SIGINT = Graceful termination of "Interactive foreground tasks"
			// SIGTERM = Graceful termination of "Background tasks"
			// SIGKILL = Forceful termination of "unresponsive tasks"
			// Note that, shell's exitCode = 128 + signal number
			if slices.Contains([]syscall.Signal{syscall.SIGKILL, syscall.SIGTERM, syscall.SIGINT},
				syscall.Signal(exitCode-128)) {
				os.Exit(exitCode - 128)
				return
			}

			// If the command has already printed something to stdout or stderr,
			//  then we don't want to print verbose error message
			if shellResult.NumBytesStdOut() > 0 || shellResult.NumBytesStdErr() > 0 {
				os.Exit(exitCode)
				return
			}

			log.Error().
				Ctx(cmd.Context()).
				Str("name", cmd.Name()).
				Int("exitCode", exitCode).
				Err(err).
				Msg("Error running command")
			os.Exit(exitCode)
		}

		log.Fatal().
			Ctx(cmd.Context()).
			Err(err).
			Msg("Error running command")
	}
	return cmd
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

func getStringArrayFlagOrFail(cmd *cobra.Command, name string) []string {
	value, err := cmd.Flags().GetStringArray(name)
	if err != nil {
		log.Fatal().
			Err(err).
			Str("flagName", name).
			Msg("Failed to fetch flag")
	}
	return value
}

func getCmdConfig(cmd *cobra.Command, args []string) []cmdrunner.Option {
	directory := getStringFlagOrFail(cmd, "directory")
	enableNetwork := !getBoolFlagOrFail(cmd, "no-network")
	readWrite := getBoolFlagOrFail(cmd, "read-write")
	readOnly := getBoolFlagOrFail(cmd, "read-only")
	noDiskAccess := getBoolFlagOrFail(cmd, "no-disk-access")
	loadEnv := getBoolFlagOrFail(cmd, "load-env")
	customDockerImage := getStringFlagOrFail(cmd, "custom-docker-image") // Optional
	mountRODirs := getStringArrayFlagOrFail(cmd, "mount-ro")             // Optional
	mode := strings.ToLower(getStringFlagOrFail(cmd, "mode"))

	// Note that, readWrite is true by default
	if noDiskAccess || readOnly {
		readWrite = false
	}

	if readOnly && noDiskAccess {
		log.Fatal().
			Ctx(cmd.Context()).
			Msg("Both read-only and no-disk-access flags cannot be enabled together")
	}

	log.Debug().
		Ctx(cmd.Context()).
		Str("name", cmd.Name()).
		Str("directory", directory).
		Strs("args", args).
		Msg("Running command")

	options := []cmdrunner.Option{
		cmdrunner.SetWorkingDir(directory),
		cmdrunner.SetArgs(getCmdArgs(cmd)),
		cmdrunner.SetRunAsNonRoot(true),
	}

	if readWrite {
		options = append(options, cmdrunner.SetMountWorkingDirReadWrite(true))
	} else if readOnly {
		options = append(options, cmdrunner.SetMountWorkingDirReadOnly(true))
	} else if noDiskAccess {
		options = append(options,
			cmdrunner.SetMountWorkingDirReadOnly(false),
			cmdrunner.SetMountWorkingDirReadWrite(false),
		)
	}

	networkType := cmdrunner.NetworkNone
	if enableNetwork {
		networkType = cmdrunner.NetworkHost
	}
	options = append(options, cmdrunner.SetNetworkType(networkType))

	if loadEnv {
		envFile := filepath.Join(directory, ".env")
		if fileInfo, _ := os.Stat(envFile); fileInfo != nil && !fileInfo.IsDir() {
			log.Debug().
				Ctx(cmd.Context()).
				Str("envFile", envFile).
				Msg(".env file found, will be loaded inside the sandbox")
			options = append(options, cmdrunner.SetLoadDotEnv(true))
		}
	}

	if customDockerImage != "" {
		log.Debug().
			Ctx(cmd.Context()).
			Str("customDockerImage", customDockerImage).
			Msg("Using custom Docker image for the sandbox")
		options = append(options, cmdrunner.SetCustomDockerImage(customDockerImage))
	}

	if len(mountRODirs) > 0 {
		log.Debug().
			Ctx(cmd.Context()).
			Strs("mountRODirs", mountRODirs).
			Msg("Mounting additional directories as read-only inside the sandbox")
		options = append(options, cmdrunner.SetExtraMountRODirs(mountRODirs))
	}

	options = append(options, cmdrunner.SetExecMode(cmdrunner.ExecMode(mode)))
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
