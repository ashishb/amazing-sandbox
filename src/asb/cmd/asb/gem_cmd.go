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
	return createCmd(cmd, cmdrunner.CmdTypeRubyGem)
}

func gemExecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gem-exec",
		Short: "Run a gem already installed inside sandbox",
	}
	return createCmd(cmd, cmdrunner.CmdTypeRubyGemExec)
}
