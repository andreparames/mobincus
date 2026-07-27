package cmd

import (
	"fmt"
	"os"
	"time"

	"mobincus/incus"

	"github.com/spf13/cobra"
)

var waitCmd = &cobra.Command{
	Use:   "wait CONTAINER [CONTAINER...]",
	Short: "Block until one or more containers stop, then print their exit codes",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := incus.NewClient()

		for _, name := range args {
			codeStr, err := client.WaitInstanceStopped(name, 5*time.Minute)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error waiting for %s: %v\n", name, err)
				continue
			}

			code := 0
			if codeStr != "" {
				fmt.Sscanf(codeStr, "%d", &code)
			}
			fmt.Println(code)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(waitCmd)
}
