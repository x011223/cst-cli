package cmd

import (
	"github.com/spf13/cobra"

	"github.com/wujunqiang/cst-cli/internal/tui"
)

var deployServers string
var deployEnv string
var deployPattern string
var deployMapping string

var deployCmd = &cobra.Command{
	Use:     "deploy",
	Aliases: []string{"deloy", "upload"},
	Short:   "Upload staged jars over SFTP and restart matching docker services",
	Long: `Selects an environment from the servers config, lists jars from the local
staging folder (localJarDir in ~/.config/cst-cli/deploy.yaml, filled by
` + "`cst-cli mvn`" + `), and uploads the selected files over SFTP.

After a successful upload you can restart the matching docker containers on
the same host, one at a time, waiting 5 seconds after each restart completes.
Jar-to-container mapping comes from the same deploy YAML.

The servers config (default ~/.config/cst-cli/servers.yaml) lists environments
with host, port, user, password and destDir. After a successful deploy the
staging folder is cleared.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return tui.RunDeploy(deployServers, deployEnv, deployPattern, deployMapping)
	},
}

func init() {
	deployCmd.Flags().StringVarP(&deployServers, "config", "c", "", "path to servers YAML config (default ~/.config/cst-cli/servers.yaml)")
	deployCmd.Flags().StringVarP(&deployEnv, "env", "e", "", "environment name to deploy to (skips the env selection screen)")
	deployCmd.Flags().StringVarP(&deployPattern, "pattern", "p", "", "jar name pattern to include, comma-separated (default: all jars in localJarDir)")
	deployCmd.Flags().StringVar(&deployMapping, "deploy-config", "", "path to deploy YAML (localJarDir and jar/container mapping, default ~/.config/cst-cli/deploy.yaml)")
}
