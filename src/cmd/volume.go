package cmd

import "github.com/spf13/cobra"

var volumeCmd = &cobra.Command{
	Use:   "volume",
	Short: "Manage volumes",
}

func init() {
	volumeCmd.AddCommand(volumeCreateCmd)
	volumeCmd.AddCommand(volumeLsCmd)
	volumeCmd.AddCommand(volumeRmCmd)
	rootCmd.AddCommand(volumeCmd)
}
