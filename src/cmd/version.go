package cmd

import (
	"fmt"

	"mobincus/docker"
	"mobincus/incus"

	"github.com/spf13/cobra"
)

type clientVersionData struct {
	Version    string
	APIVersion string
}

type serverVersionData struct {
	Version    string
	APIVersion string
	OS         string
	Kernel     string
	Driver     string
}

type versionData struct {
	Client clientVersionData
	Server serverVersionData
}

var versionFormat string

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := incus.NewClient()
		info, err := client.GetServerInfo()
		if err != nil {
			return err
		}

		data := versionData{
			Client: clientVersionData{
				Version:    incus.Version,
				APIVersion: incus.APIVersion,
			},
			Server: serverVersionData{
				Version:    info.Environment.ServerVersion,
				APIVersion: info.APIVersion,
				OS:         info.Environment.OSName + " " + info.Environment.OSVersion,
				Kernel:     info.Environment.Kernel + " " + info.Environment.KernelVersion,
				Driver:     info.Environment.Driver,
			},
		}

		if versionFormat != "" {
			out, err := docker.TemplateOutput(versionFormat, data)
			if err != nil {
				return err
			}
			fmt.Println(out)
			return nil
		}

		fmt.Println("Client:")
		fmt.Printf(" Version:\t%s\n", data.Client.Version)
		fmt.Printf(" API version:\t%s\n", data.Client.APIVersion)
		fmt.Println()
		fmt.Println("Server:")
		fmt.Printf(" Server:\t%s\n", info.Environment.Server)
		fmt.Printf(" Version:\t%s\n", data.Server.Version)
		fmt.Printf(" API version:\t%s\n", data.Server.APIVersion)
		fmt.Printf(" OS:\t\t%s\n", data.Server.OS)
		fmt.Printf(" Kernel:\t%s\n", data.Server.Kernel)
		fmt.Printf(" Driver:\t%s\n", data.Server.Driver)
		fmt.Printf(" Driver version:\t%s\n", info.Environment.DriverVersion)

		return nil
	},
}

func init() {
	versionCmd.Flags().StringVarP(&versionFormat, "format", "f", "", "Format output using a custom template")
	rootCmd.AddCommand(versionCmd)
}
