package cmd

import (
	"fmt"
	"strings"

	"mobincus/docker"
	"mobincus/incus"

	"github.com/spf13/cobra"
)

var (
	psQuiet   bool
	psAll     bool
	psFilters []string
	psFormat  string
)

type psRow struct {
	ID     string            `json:"ID"`
	Names  string            `json:"Names"`
	Image  string            `json:"Image"`
	State  string            `json:"State"`
	Status string            `json:"Status"`
	Ports  string            `json:"Ports"`
	Labels map[string]string `json:"Labels,omitempty"`
}

var psCmd = &cobra.Command{
	Use:   "ps",
	Short: "List containers",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := incus.NewClient()
		containers, err := client.ListContainers()
		if err != nil {
			return fmt.Errorf("failed to list containers: %w", err)
		}

		filtered := filterContainers(containers)

		if psFormat != "" {
			for _, c := range filtered {
			labels := make(map[string]string)
			for k, v := range c.Config {
				labels[strings.TrimPrefix(k, "user.")] = v
			}
			row := psRow{
				ID:     c.ID,
				Names:  strings.Join(c.Names, ","),
				Image:  "",
				State:  strings.ToLower(c.Status),
				Status: c.Status,
				Ports:  "",
				Labels: labels,
			}
				out, err := docker.TemplateOutput(psFormat, row)
				if err != nil {
					return fmt.Errorf("template error: %w", err)
				}
				fmt.Println(out)
			}
			return nil
		}

		if psQuiet {
			for _, c := range filtered {
				fmt.Println(c.ID)
			}
		} else {
			output := docker.FormatContainerList(filtered)
			fmt.Print(output)
		}
		return nil
	},
}

func filterContainers(containers []incus.DockerContainer) []incus.DockerContainer {
	if len(psFilters) == 0 {
		return containers
	}

	var result []incus.DockerContainer
	for _, c := range containers {
		if matchFilters(c) {
			result = append(result, c)
		}
	}
	return result
}

func matchFilters(c incus.DockerContainer) bool {
	for _, f := range psFilters {
		if strings.HasPrefix(f, "label=") {
			rest := f[6:]
			parts := strings.SplitN(rest, "=", 2)
			labelKey := parts[0]
			configKey := "user." + labelKey

			if len(parts) > 1 {
				if c.Config[configKey] != parts[1] {
					return false
				}
			} else {
				if _, ok := c.Config[configKey]; !ok {
					return false
				}
			}
		}
	}
	return true
}

func init() {
	psCmd.Flags().BoolVarP(&psQuiet, "quiet", "q", false, "Only display container IDs")
	psCmd.Flags().BoolVarP(&psAll, "all", "a", false, "Show all containers (default shows running)")
	psCmd.Flags().StringArrayVarP(&psFilters, "filter", "f", nil, "Filter output based on conditions provided")
	psCmd.Flags().StringVarP(&psFormat, "format", "", "", "Format output using a custom template")
	rootCmd.AddCommand(psCmd)
}
