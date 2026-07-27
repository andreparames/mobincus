package cmd

import (
	"fmt"
	"os"

	"mobincus/incus"

	"github.com/spf13/cobra"
)

var volumeRmForce bool

var volumeRmCmd = &cobra.Command{
	Use:   "rm [OPTIONS] VOLUME [VOLUME...]",
	Short: "Remove one or more volumes",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := incus.NewClient()

		for _, name := range args {
			if err := client.DeleteVolume(name); err != nil {
				fmt.Fprintf(os.Stderr, "Error: failed to remove volume %s: %v\n", name, err)
				continue
			}
			fmt.Println(name)
		}
		return nil
	},
}

func init() {
	volumeRmCmd.Flags().BoolVarP(&volumeRmForce, "force", "f", false, "Force the removal of the volume")
}
