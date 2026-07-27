package cmd

import (
	"fmt"
	"strings"

	"mobincus/incus"

	"github.com/spf13/cobra"
)

var volumeCreateCmd = &cobra.Command{
	Use:   "create [OPTIONS] [VOLUME]",
	Short: "Create a volume",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := ""
		if len(args) > 0 {
			name = args[0]
		} else {
			name = generateName()
		}

		client := incus.NewClient()
		if err := client.CreateVolume(name); err != nil {
			return err
		}

		// Docker prints only the volume name (strip pool prefix if any)
		if strings.Contains(name, "/") {
			parts := strings.Split(name, "/")
			name = parts[len(parts)-1]
		}
		fmt.Println(name)
		return nil
	},
}
