package main

import (
	"github.com/ashishb/asb/src/asb/internal/cmdrunner"
	"github.com/spf13/cobra"
)

func npmCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "npm",
		Short: "Run an npm command",
	}
	return createCmd(cmd, cmdrunner.NewNpmCmdConfig)
}
