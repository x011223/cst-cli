package cmd

import (
	"github.com/spf13/cobra"

	"github.com/wujunqiang/cst-cli/internal/tui"
)

var deployConfig string

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy jars on the server: backup old jar, move new jar, restart container",
	Long: `Runs on the deployment server (not via SSH). For each selected service it:
  1. checks the freshly uploaded jar exists in tmpDir (you upload it separately)
  2. renames the existing jar in jarDir to <jar>.bak
  3. moves the new jar from tmpDir into jarDir
  4. runs 'docker restart <container>'

Services and their jar/container mapping are read from a YAML config
(default ~/.config/cst-cli/deploy.yaml).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return tui.RunDeploy(deployConfig)
	},
}

func init() {
	deployCmd.Flags().StringVarP(&deployConfig, "config", "c", "", "path to deploy YAML config (default ~/.config/cst-cli/deploy.yaml)")
}
