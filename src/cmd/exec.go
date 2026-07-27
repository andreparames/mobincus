package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"mobincus/incus"

	"github.com/spf13/cobra"
)

type execOptions struct {
	interactive bool
	tty         bool
	env         []string
	user        string
	workdir     string
}

var execOpts execOptions

var execCmd = &cobra.Command{
	Use:   "exec [OPTIONS] CONTAINER COMMAND [ARG...]",
	Short: "Execute a command in a running container",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		containerName := args[0]
		command := args[1:]

		client := incus.NewClient()

		inst, err := client.GetInstance(containerName)
		if err != nil {
			return fmt.Errorf("error: no such object: %s", containerName)
		}
		if inst.Status != "Running" {
			return fmt.Errorf("error: container %s is not running", containerName)
		}

		env := make(map[string]string)
		for _, e := range execOpts.env {
			parts := strings.SplitN(e, "=", 2)
			if len(parts) == 2 {
				env[parts[0]] = parts[1]
			}
		}

		user := 0
		group := 0
		if execOpts.user != "" {
			fmt.Sscanf(execOpts.user, "%d:%d", &user, &group)
		}

		var stdin io.Reader
		var stdout, stderr io.Writer

		if execOpts.interactive || execOpts.tty {
			stdin = os.Stdin
		}
		stdout = os.Stdout
		stderr = os.Stderr

		exitCode, execErr := client.ExecAndStream(containerName, incus.ExecPost{
			Command:     command,
			Interactive: execOpts.tty,
			WaitForWS:   true,
			Environment: env,
			User:        user,
			Group:       group,
			Cwd:         execOpts.workdir,
		}, stdin, stdout, stderr)

		if execErr != nil {
			return execErr
		}

		if exitCode != 0 {
			os.Exit(exitCode)
		}

		return nil
	},
}

func init() {
	execCmd.Flags().BoolVarP(&execOpts.interactive, "interactive", "i", false, "Keep STDIN open even if not attached")
	execCmd.Flags().BoolVarP(&execOpts.tty, "tty", "t", false, "Allocate a pseudo-TTY")
	execCmd.Flags().StringArrayVarP(&execOpts.env, "env", "e", nil, "Set environment variables")
	execCmd.Flags().StringVarP(&execOpts.user, "user", "u", "", "Username or UID (format: <name|uid>[:<group|gid>])")
	execCmd.Flags().StringVarP(&execOpts.workdir, "workdir", "w", "", "Working directory inside the container")
	execCmd.Flags().SetInterspersed(false)
	rootCmd.AddCommand(execCmd)
}
