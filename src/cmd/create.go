package cmd

import (
	"fmt"
	"strings"

	"mobincus/incus"

	"github.com/spf13/cobra"
)

var createLabels []string

var createCmd = &cobra.Command{
	Use:   "create [OPTIONS] IMAGE [COMMAND] [ARG...]",
	Short: "Create a new container",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		imageName := args[0]
		command := args[1:]

		client := incus.NewClient()

		source, err := client.FindImage(imageName)
		if err != nil {
			return fmt.Errorf("failed to find image %s: %w", imageName, err)
		}

		name := generateName()

		labels := make(map[string]string)
		for _, l := range createLabels {
			parts := strings.SplitN(l, "=", 2)
			key := "user." + parts[0]
			if len(parts) > 1 {
				labels[key] = parts[1]
			} else {
				labels[key] = ""
			}
		}

		_, err = client.CreateInstance(incus.InstanceCreateRequest{
			Name:       name,
			Source:     source,
			Config:     labels,
			Ephemeral:  false,
			Profiles:   []string{"default"},
		})
		if err != nil {
			return fmt.Errorf("failed to create instance: %w", err)
		}

		if len(command) > 0 {
			client.SetConfig(name, "user.docker_cmd", joinCommand(command))
		}

		fmt.Println(name)
		return nil
	},
}

func init() {
	createCmd.Flags().StringArrayVarP(&createLabels, "label", "l", nil, "Set metadata on container")
	createCmd.Flags().SetInterspersed(false)
	rootCmd.AddCommand(createCmd)
}
