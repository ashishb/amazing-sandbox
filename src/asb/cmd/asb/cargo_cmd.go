package main

import (
	"github.com/ashishb/asb/src/asb/internal/cmdrunner"
	"github.com/spf13/cobra"
)

func cargoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cargo",
		Short: "Run a cargo command",
	}
	return createCmd(cmd, cmdrunner.NewCargoCmdConfig)
}
