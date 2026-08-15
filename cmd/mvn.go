package cmd

import (
	"github.com/spf13/cobra"

	"github.com/wujunqiang/cst-cli/internal/tui"
)

var mvnEnv string

var mvnCmd = &cobra.Command{
	Use:     "mvn",
	Aliases: []string{"maven"},
	Short:   "List Maven projects and build them (clean → compile → package)",
	Long: `Scans the current directory for Maven projects (folders containing a pom.xml),
lets you select one or more, then builds each with:

  mvn -P<env> clean
  mvn -P<env> compile
  mvn -P<env> package

The environment (dev/prod/...) is passed to Maven as the active profile (-P),
which only swaps injected config values (e.g. nacos addresses). Selected projects
build in parallel; the three phases run serially within a project. Failures are
reported with the Maven error output.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return tui.RunMvnBuild(mvnEnv)
	},
}

func init() {
	mvnCmd.Flags().StringVarP(&mvnEnv, "env", "e", "", "build environment / Maven profile (dev, prod); prompts if omitted")
}
