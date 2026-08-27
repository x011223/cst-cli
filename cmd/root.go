// Package cmd defines the cst-cli subcommands using cobra.
package cmd

import (
	"github.com/spf13/cobra"
)

// Version is the cst-cli version, overridable at build time via -ldflags.
var Version = "0.1.0"

var rootCmd = &cobra.Command{
	Use:   "cst-cli",
	Short: "A collection of developer commands",
	Long:  "cst-cli bundles small developer workflows (Maven builds, and more) behind a single command.",
}

// Execute runs the root command. It is the single entry point called from main.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(mvnCmd)
	rootCmd.AddCommand(gitCmd)
	rootCmd.AddCommand(deployCmd)
	rootCmd.AddCommand(jarsCmd)
	rootCmd.AddCommand(uploadCmd)
	rootCmd.AddCommand(versionCmd)
}
