package main

import (
	"github.com/ashishb/asb/src/asb/internal/cmdrunner"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func gemExecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gem-exec",
		Short: "Run a gem already installed inside sandbox",
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

		options := getCmdConfig(cmd, *directory)
		config := cmdrunner.NewRubyGemExecCmdConfig(options...)
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

func cargoExecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cargo-exec",
		Short: "Run a Rust-based binary package already installed inside sandbox",
	}
	cmd.FParseErrWhitelist.UnknownFlags = true

	directory := cmd.PersistentFlags().StringP("directory", "d", getCwdOrFail(),
		"Working directory for this command")
	cmd.Run = func(cmd *cobra.Command, args []string) {
		log.Info().
			Ctx(cmd.Context()).
			Str("directory", *directory).
			Strs("args", args).
			Msg("Running binary package that is already inside sandbox")

		options := getCmdConfig(cmd, *directory)
		config := cmdrunner.NewRustCargoExecCmdConfig(options...)
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
