package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"time"

	"mobincus/incus"

	"github.com/spf13/cobra"
)

type runOptions struct {
	attach []string
	rm     bool
	interactive bool
}

var runOpts runOptions

var runCmd = &cobra.Command{
	Use:   "run [OPTIONS] IMAGE [COMMAND] [ARG...]",
	Short: "Run a command in a new container",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		imageName := args[0]
		command := args[1:]
		if len(command) == 0 {
			command = []string{"/bin/sh"}
		}

		attachStdin := false
		attachStdout := false
		attachStderr := false

		if runOpts.interactive {
			attachStdin = true
		}

		if len(runOpts.attach) == 0 {
			attachStdin = true
			attachStdout = true
			attachStderr = true
		} else {
			for _, s := range runOpts.attach {
				switch s {
				case "stdin":
					attachStdin = true
				case "stdout":
					attachStdout = true
				case "stderr":
					attachStderr = true
				}
			}
		}

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

		err = client.StartInstance(name)
		if err != nil {
			return fmt.Errorf("failed to start instance: %w", err)
		}

		err = waitRunning(client, name, 30*time.Second)
		if err != nil {
			return fmt.Errorf("waiting for instance to start: %w", err)
		}

		var stdin io.Reader
		var stdout, stderr io.Writer

		if attachStdin {
			stdin = os.Stdin
		}
		if attachStdout {
			stdout = os.Stdout
		}
		if attachStderr {
			stderr = os.Stderr
		}

		interactive := false
		exitCode, execErr := client.ExecAndStream(name, incus.ExecPost{
			Command:     command,
			Interactive: interactive,
			WaitForWS:   true,
		}, stdin, stdout, stderr)

		if runOpts.rm {
			client.StopInstance(name, true)
			for i := 0; i < 5; i++ {
				time.Sleep(200 * time.Millisecond)
				err := client.DeleteInstance(name)
				if err == nil {
					break
				}
			}
		}

		if execErr != nil {
			return execErr
		}

		if exitCode != 0 {
			os.Exit(exitCode)
		}

		return nil
	},
}

func generateName() string {
	b := make([]byte, 8)
	rand.Read(b)
	return "mobincus-" + hex.EncodeToString(b)
}

func waitStopped(client *incus.Client, name string, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		inst, err := client.GetInstance(name)
		if err != nil || inst.Status == "Stopped" {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func waitRunning(client *incus.Client, name string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		inst, err := client.GetInstance(name)
		if err != nil {
			return err
		}
		if inst.Status == "Running" {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for instance %s to start", name)
}

func init() {
	runCmd.Flags().StringArrayVarP(&runOpts.attach, "attach", "a", nil, "Attach to STDIN, STDOUT or STDERR")
	runCmd.Flags().BoolVarP(&runOpts.rm, "rm", "", false, "Automatically remove the container when it exits")
	runCmd.Flags().BoolVarP(&runOpts.interactive, "interactive", "i", false, "Keep STDIN open even if not attached")
	runCmd.Flags().SetInterspersed(false)
	rootCmd.AddCommand(runCmd)
}
