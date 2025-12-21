package main

import (
	"github.com/ashishb/asb/src/asb/internal/cmdrunner"
	"github.com/spf13/cobra"
)

func npxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "npx",
		Short: "Run an npx command",
	}
	return createCmd(cmd, cmdrunner.NewNpxCmdConfig)
}
