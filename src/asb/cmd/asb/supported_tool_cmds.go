package main

import (
	"github.com/spf13/cobra"

	"github.com/ashishb/asb/src/asb/internal/cmdrunner"
)

func cargoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cargo",
		Short: "Run a cargo command",
	}
	return createCmd(cmd, cmdrunner.CmdTypeRustCargo)
}

func cargoExecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cargo-exec",
		Short: "Run a Rust-based binary package already installed inside sandbox",
	}
	return createCmd(cmd, cmdrunner.CmdTypeRustCargoExec)
}

func pipCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pip",
		Short: "Install Python packages using pip",
	}
	return createCmd(cmd, cmdrunner.CmdTypePythonPip)
}

func pipExecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pip-exec",
		Short: "Run a Python-based package already installed inside sandbox",
	}
	return createCmd(cmd, cmdrunner.CmdTypePythonPipExec)
}

func uvCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uv",
		Short: "Run a uv command",
	}
	return createCmd(cmd, cmdrunner.CmdTypePythonUv)
}

func uvxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uvx",
		Short: "Run a Python-based package already installed inside sandbox using uvx",
	}
	return createCmd(cmd, cmdrunner.CmdTypePythonUvx)
}

func poetryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "poetry",
		Short: "Run a poetry command",
	}
	return createCmd(cmd, cmdrunner.CmdTypePythonPoetry)
}

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

func bunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bun",
		Short: "Run a bun command",
	}
	return createCmd(cmd, cmdrunner.CmdTypeBun)
}

func npmCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "npm",
		Short: "Run an npm command",
	}
	return createCmd(cmd, cmdrunner.CmdTypeNpm)
}

func npxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "npx",
		Short: "Run an npx command",
	}
	return createCmd(cmd, cmdrunner.CmdTypeNpx)
}

func yarnCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "yarn",
		Short: "Run a yarn command",
	}
	return createCmd(cmd, cmdrunner.CmdTypeYarn)
}
