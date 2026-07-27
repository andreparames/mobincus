package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"mobincus/incus"

	"github.com/spf13/cobra"
)

var inspectType string

var inspectCmd = &cobra.Command{
	Use:   "inspect [OBJECT]",
	Short: "Return low-level information on Incus instances",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := incus.NewClient()

		var results []interface{}

		for _, ref := range args {
			inst, err := client.GetInstance(ref)
			if err != nil {
				fmt.Println("[]")
				return &StatusError{
					StatusCode: 1,
					Status:     fmt.Sprintf("error: no such object: %s", ref),
				}
			}

			results = append(results, toDockerInspect(inst))
		}

		out, _ := json.MarshalIndent(results, "", "    ")
		fmt.Println(string(out))
		return nil
	},
}

func toDockerInspect(inst *incus.Instance) map[string]interface{} {
	created := inst.CreatedAt
	if created != "" {
		created = strings.Replace(created, "T", " ", 1)
		if idx := strings.Index(created, "+"); idx > 0 {
			created = created[:idx]
		}
	}

	status := inst.Status
	running := status == "Running"

	config := map[string]interface{}{
		"Image": "incus:" + inst.Type,
		"Env":   []string{},
	}

	return map[string]interface{}{
		"Id":     inst.Name,
		"Name":   "/" + inst.Name,
		"Status": status,
		"State": map[string]interface{}{
			"Status":  status,
			"Running": running,
			"ExitCode": 0,
		},
		"Created":     created,
		"Config":      config,
		"Platform":    inst.Architecture,
		"Type":        inst.Type,
		"Description": inst.Description,
		"Profiles":    inst.Profiles,
		"Project":     inst.Project,
	}
}

type inspectOptions struct {
	objectType string
}

var inspectOpts inspectOptions

func init() {
	inspectCmd.Flags().StringVarP(&inspectType, "type", "", "", "Return JSON for specified type")
}
