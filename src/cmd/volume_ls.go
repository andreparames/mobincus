package cmd

import (
	"fmt"

	"mobincus/incus"

	"github.com/spf13/cobra"
)

func extractName(url string) string {
	for i := len(url) - 1; i >= 0; i-- {
		if url[i] == '/' {
			return url[i+1:]
		}
	}
	return url
}

var volumeLsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List volumes",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := incus.NewClient()
		urls, err := client.ListVolumes()
		if err != nil {
			return err
		}

		fmt.Println("DRIVER    VOLUME NAME")
		for _, u := range urls {
			name := extractName(u)
			fmt.Printf("local     %s\n", name)
		}
		return nil
	},
}
