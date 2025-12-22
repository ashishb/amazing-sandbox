package main

import (
	_ "embed"
	"fmt"
	"github.com/spf13/cobra"
)

//go:embed version.txt
var version string

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Display asb version",
		Long:  _description,
		Run: func(c *cobra.Command, _ []string) {
			fmt.Printf("Amazing Sandbox (asb)\nversion: %s\n%s\n", version, _description)
		},
	}
}
