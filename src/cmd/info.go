package cmd

import (
	"fmt"

	"mobincus/docker"
	"mobincus/incus"

	"github.com/spf13/cobra"
)

type infoData struct {
	OSType        string
	Containers    int
	ServerVersion string
	Driver        string
	KernelVersion string
	OSName        string
	Architecture  string
}

var infoFormat string

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Display system-wide information",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := incus.NewClient()
		info, err := client.GetServerInfo()
		if err != nil {
			return fmt.Errorf("failed to get server info: %w", err)
		}

		containers, err := client.ListContainers()
		if err != nil {
			return err
		}

		data := infoData{
			OSType:        "linux",
			Containers:    len(containers),
			ServerVersion: info.Environment.ServerVersion,
			Driver:        info.Environment.Driver,
			KernelVersion: info.Environment.KernelVersion,
			OSName:        info.Environment.OSName + " " + info.Environment.OSVersion,
		}

		if infoFormat != "" {
			out, err := docker.TemplateOutput(infoFormat, data)
			if err != nil {
				return err
			}
			fmt.Println(out)
			return nil
		}

		fmt.Printf("OSType: %s\n", data.OSType)
		fmt.Printf("Containers: %d\n", data.Containers)
		fmt.Printf("Server Version: %s\n", data.ServerVersion)
		fmt.Printf("Driver: %s\n", data.Driver)
		fmt.Printf("Kernel Version: %s\n", data.KernelVersion)
		fmt.Printf("OS: %s\n", data.OSName)
		return nil
	},
}

func init() {
	infoCmd.Flags().StringVarP(&infoFormat, "format", "f", "", "Format output using a custom template")
	rootCmd.AddCommand(infoCmd)
}
