package cmd

import (
	"github.com/spf13/cobra"

	"github.com/wujunqiang/cst-cli/internal/tui"
)

var gitCmd = &cobra.Command{
	Use:     "gst",
	Aliases: []string{"git", "status"},
	Short:   "Show git changes across repositories in the current directory",
	Long: `Scans the current directory for git repositories (immediate sub-directories
with a .git entry, plus the directory itself) and lists those with uncommitted
changes. For each changed repository it shows the branch and a tree of modified,
added, deleted and untracked files.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return tui.RunGitStatus()
	},
}
