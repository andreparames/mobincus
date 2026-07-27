package cmd

import (
	"fmt"

	"mobincus/incus"

	"github.com/spf13/cobra"
)

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

		_, err = client.CreateInstance(incus.InstanceCreateRequest{
			Name:       name,
			Source:     source,
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
	createCmd.Flags().SetInterspersed(false)
	rootCmd.AddCommand(createCmd)
}
