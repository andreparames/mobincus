package cmd

import (
	"fmt"

	"mobincus/docker"
	"mobincus/incus"

	"github.com/spf13/cobra"
)

var containerLsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List containers",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := incus.NewClient()
		containers, err := client.ListContainers()
		if err != nil {
			return fmt.Errorf("failed to list containers: %w", err)
		}
		output := docker.FormatContainerList(containers)
		fmt.Print(output)
		return nil
	},
}
