package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

type StatusError struct {
	StatusCode int
	Status     string
}

func (e *StatusError) Error() string {
	return e.Status
}

var (
	tlsVerify  bool
	tlsEnabled bool
)

var rootCmd = &cobra.Command{
	Use:           "docker",
	Short:         "mobincus is a Docker CLI-compatible interface for Incus",
	Long:          `mobincus translates Docker CLI commands into Incus API calls, returning output in Docker CLI format.`,
	SilenceErrors: true,
	SilenceUsage:  true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if !cmd.Flags().Changed("tlsverify") {
			return nil
		}

		certPath := os.Getenv("DOCKER_CERT_PATH")
		if certPath == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return err
			}
			certPath = filepath.Join(home, ".docker")
		}

		if tlsVerify {
			caPath := filepath.Join(certPath, "ca.pem")
			if _, err := os.Stat(caPath); os.IsNotExist(err) {
				return &StatusError{
					StatusCode: 1,
					Status:     fmt.Sprintf("open %s: no such file or directory", caPath),
				}
			}
		}

		return &StatusError{
			StatusCode: 1,
			Status:     fmt.Sprintf("unable to resolve docker endpoint: open %s: no such file or directory", filepath.Join(certPath, "ca.pem")),
		}
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		if statusErr, ok := err.(*StatusError); ok {
			if statusErr.Status != "" {
				fmt.Fprintln(os.Stderr, statusErr.Status)
			}
			os.Exit(statusErr.StatusCode)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&tlsVerify, "tlsverify", false, "Use TLS and verify the remote")
	rootCmd.PersistentFlags().BoolVar(&tlsEnabled, "tls", false, "Use TLS; implied by --tlsverify")

	rootCmd.AddCommand(containerCmd)
	rootCmd.AddCommand(inspectCmd)
	rootCmd.AddCommand(psCmd)
}
