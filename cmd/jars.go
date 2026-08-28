package cmd

import (
	"github.com/spf13/cobra"

	"github.com/wujunqiang/cst-cli/internal/tui"
)

var jarsDest string
var jarsPattern string

var jarsCmd = &cobra.Command{
	Use:   "jars",
	Short: "Find built jars (target/*.jar) and copy them to a directory",
	Long: `Scans Maven projects in the current directory, finds the built artifacts under
each project's target/ directory (skipping repackaged originals and attached
sources/tests jars), and copies them to the destination (default ~/Documents/Jars/).
Only jars matching the pattern (default *-application*.jar) are listed; select
the ones to copy interactively.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return tui.RunJars(jarsDest, jarsPattern)
	},
}

func init() {
	jarsCmd.Flags().StringVarP(&jarsDest, "dest", "d", "~/Documents/Jars/", "destination directory for collected jars")
	jarsCmd.Flags().StringVarP(&jarsPattern, "pattern", "p", "", "jar name pattern to include, comma-separated (default *-application*.jar)")
}
