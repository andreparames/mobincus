package cmd

import (
	"fmt"
	"os"
	"time"

	"mobincus/incus"

	"github.com/spf13/cobra"
)

var rmForce bool

var rmCmd = &cobra.Command{
	Use:   "rm [OPTIONS] CONTAINER [CONTAINER...]",
	Short: "Remove one or more containers",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := incus.NewClient()

		for _, name := range args {
			if rmForce {
				client.StopInstance(name, true)
				time.Sleep(200 * time.Millisecond)
			}

			if err := client.DeleteInstance(name); err != nil {
				fmt.Fprintf(os.Stderr, "error: failed to remove container %s: %v\n", name, err)
				continue
			}
			fmt.Println(name)
		}

		return nil
	},
}

func init() {
	rmCmd.Flags().BoolVarP(&rmForce, "force", "f", false, "Force the removal of a running container")
	rootCmd.AddCommand(rmCmd)
}
