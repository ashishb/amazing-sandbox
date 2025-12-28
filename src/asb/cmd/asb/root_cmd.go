package main

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

const _description = "asb is CLI tool for running tools inside Sandbox\n" +
	"See https://ashishb.net/programming/run-tools-inside-docker/ for reasoning behind this tool"

func getRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "asb",
		Short: "asb is CLI tool for running tools inside Sandbox",
		Long:  _description,
		Run: func(c *cobra.Command, _ []string) {
			err := c.Help()
			if err != nil {
				log.Fatal().
					Err(err).
					Msg("Error displaying help")
			}
		},
	}

	_ = rootCmd.PersistentFlags().StringP("directory", "d", getCwdOrFail(), "Working directory for this command")
	_ = rootCmd.PersistentFlags().BoolP("no-network", "n", false, "Disable network access inside the sandbox")
	_ = rootCmd.PersistentFlags().BoolP("read-only", "r", false, "Mount working directory and referenced directories as read-only")

	rootCmd.AddCommand(versionCmd())

	// Python related
	if false { // Disabled for now
		rootCmd.AddCommand(pipCmd())
		rootCmd.AddCommand(pipExecCmd())
	}
	rootCmd.AddCommand(uvxCmd())
	rootCmd.AddCommand(poetryCmd())

	// Rust related
	rootCmd.AddCommand(cargoCmd())
	rootCmd.AddCommand(cargoExecCmd())

	// Ruby related
	rootCmd.AddCommand(gemCmd())
	rootCmd.AddCommand(gemExecCmd())

	// Javascript related
	rootCmd.AddCommand(npmCmd())
	rootCmd.AddCommand(npxCmd())
	rootCmd.AddCommand(yarnCmd())

	return rootCmd
}
