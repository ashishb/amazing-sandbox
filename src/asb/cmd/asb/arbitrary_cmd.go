package main

import (
	"github.com/ashishb/asb/src/asb/internal/cmdrunner"
	"github.com/spf13/cobra"
)

func gemExecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gem-exec",
		Short: "Run a gem already installed inside sandbox",
	}
	return createCmd(cmd, cmdrunner.NewRubyGemExecCmdConfig)
}

func cargoExecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cargo-exec",
		Short: "Run a Rust-based binary package already installed inside sandbox",
	}
	return createCmd(cmd, cmdrunner.NewRustCargoExecCmdConfig)
}
