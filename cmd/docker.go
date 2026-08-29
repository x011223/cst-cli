package cmd

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/wujunqiang/cst-cli/internal/docker"
	"github.com/wujunqiang/cst-cli/internal/tui"
)

var dockerConfig string
var dockerEnv string
var dockerTimeout time.Duration

var dockerCmd = &cobra.Command{
	Use:     "docker",
	Aliases: []string{"ps", "containers"},
	Short:   "List remote docker containers and restart them in groups",
	Long: `Connects to a servers.yaml environment over SSH and lists docker containers
(lazydocker-style). Multi-select a group, then press enter to restart them
all at once.

Live docker logs are shown for every container in the group. Each container
stops following logs when a line contains 启动成功, or when it fails/times out.
After the whole group finishes (success or failure), the log pane closes and
you can select the next group. Containers that have already been restarted
are marked in the list.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if dockerTimeout <= 0 {
			dockerTimeout = docker.DefaultLogTimeout
		}
		return tui.RunDocker(dockerConfig, dockerEnv, dockerTimeout)
	},
}

func init() {
	dockerCmd.Flags().StringVarP(&dockerConfig, "config", "c", "", "path to servers YAML (default ~/.config/cst-cli/servers.yaml)")
	dockerCmd.Flags().StringVarP(&dockerEnv, "env", "e", "", "environment name (skips the env selection screen)")
	dockerCmd.Flags().DurationVar(&dockerTimeout, "timeout", docker.DefaultLogTimeout, "how long to wait for 启动成功 after each restart")
}
