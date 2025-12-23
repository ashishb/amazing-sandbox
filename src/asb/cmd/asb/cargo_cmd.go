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

func cargoExecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cargo-exec",
		Short: "Run a Rust-based binary package already installed inside sandbox",
	}
	return createCmd(cmd, cmdrunner.NewRustCargoExecCmdConfig)
}

func pipCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pip",
		Short: "Install Python packages using pip",
	}
	return createCmd(cmd, cmdrunner.NewPipCmdConfig)
}

func pipExecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pip-exec",
		Short: "Run a Python-based package already installed inside sandbox",
	}
	return createCmd(cmd, cmdrunner.NewPipExecCmdConfig)
}

func uvxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uvx",
		Short: "Run a Python-based package already installed inside sandbox using uvx",
	}
	return createCmd(cmd, cmdrunner.NewUvxCmdConfig)
}

func poetryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "poetry",
		Short: "Run a poetry command",
	}
	return createCmd(cmd, cmdrunner.NewPoetryCmdConfig)
}
