package cmd

import (
	"fmt"

	"mobincus/incus"

	"github.com/spf13/cobra"
)

var inspectCmd = &cobra.Command{
	Use:   "inspect [OBJECT]",
	Short: "Return low-level information on Incus instances",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := incus.NewClient()

		statusErrors := []string{}
		for _, ref := range args {
			_, err := client.GetInstance(ref)
			if err != nil {
				statusErrors = append(statusErrors, fmt.Sprintf("error: no such object: %s", ref))
			}
		}

		if len(statusErrors) > 0 {
			fmt.Println("[]")
			return &StatusError{
				StatusCode: 1,
				Status:     statusErrors[0],
			}
		}

		return nil
	},
}
