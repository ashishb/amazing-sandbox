package main

import (
	"github.com/ashishb/asb/src/asb/internal/cmdrunner"
	"github.com/spf13/cobra"
)

func gemCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gem",
		Short: "Run a Ruby gem-based CLI tool",
	}
	return createCmd(cmd, cmdrunner.NewRubyGemCmdConfig)
}
