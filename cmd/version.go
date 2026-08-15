package cmd

import (
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:     "version",
	Aliases: []string{"v"},
	Short:   "Show the cst-cli version",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Printf("cst-cli %s\n", Version)
	},
}
