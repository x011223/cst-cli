package cmd

import (
	"github.com/spf13/cobra"

	"github.com/wujunqiang/cst-cli/internal/tui"
)

var uploadConfig string
var uploadEnv string
var uploadPattern string
var uploadDeployConfig string

var uploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Find built jars, upload them over SFTP, then restart matching docker services",
	Long: `Selects an environment from a local servers config, finds the built jars in the
current directory's Maven projects (target/*.jar), and uploads them to that
environment's destination directory over SFTP.

After a successful upload you can restart the matching docker containers on
the same host, one at a time, waiting 5 seconds after each restart completes.
Jar-to-container mapping comes from the deploy YAML (default
~/.config/cst-cli/deploy.yaml).

Only jars matching the pattern (default *-application-*.jar) are listed; select
the ones to upload interactively. The servers config (default
~/.config/cst-cli/servers.yaml) lists environments with host, port, user, password
and destDir.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return tui.RunUpload(uploadConfig, uploadEnv, uploadPattern, uploadDeployConfig)
	},
}

func init() {
	uploadCmd.Flags().StringVarP(&uploadConfig, "config", "c", "", "path to servers YAML config (default ~/.config/cst-cli/servers.yaml)")
	uploadCmd.Flags().StringVarP(&uploadEnv, "env", "e", "", "environment name to upload to (skips the env selection screen)")
	uploadCmd.Flags().StringVarP(&uploadPattern, "pattern", "p", "", "jar name pattern to include, comma-separated (default *-application-*.jar)")
	uploadCmd.Flags().StringVar(&uploadDeployConfig, "deploy-config", "", "path to deploy YAML (jar/container mapping, default ~/.config/cst-cli/deploy.yaml)")
}
