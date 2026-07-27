package cmd

import (
	"fmt"
	"os"
	"time"

	"mobincus/incus"

	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start [OPTIONS] CONTAINER [CONTAINER...]",
	Short: "Start one or more stopped containers",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := incus.NewClient()

		for _, name := range args {
			err := client.StartInstance(name)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: failed to start container %s: %v\n", name, err)
				continue
			}

			waitRunning(client, name, 30*time.Second)

			cmdStr, err := client.GetConfig(name, "user.docker_cmd")
			if err == nil && cmdStr != "" {
				command := splitCommand(cmdStr)
				if len(command) > 0 {
					exitCode, execErr := client.ExecAndStream(name, incus.ExecPost{
						Command:     command,
						WaitForWS:   true,
					}, nil, nil, nil)
					if execErr != nil {
						client.SetConfig(name, "user.docker_exit_code", fmt.Sprintf("%d", -1))
					} else {
						client.SetConfig(name, "user.docker_exit_code", fmt.Sprintf("%d", exitCode))
					}
					client.StopInstance(name, true)
				}
			}

			fmt.Println(name)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
