package cmd

import (
	"github.com/spf13/cobra"

	"github.com/wujunqiang/cst-cli/internal/tui"
)

var mvnCmd = &cobra.Command{
	Use:     "mvn",
	Aliases: []string{"maven"},
	Short:   "List Maven projects and build them (clean → compile → package)",
	Long: `Scans the current directory for Maven projects (folders containing a pom.xml),
lets you select one or more, then builds each with:

  mvn clean
  mvn compile
  mvn package

Selected projects build in parallel; the three phases run serially within a project.
Failures are reported with the Maven error output.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return tui.RunMvnBuild()
	},
}
