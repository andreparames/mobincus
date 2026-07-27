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

			state, _ := client.GetInstanceState(ref)

			results = append(results, buildInspectResult(client, inst, state))
		}

		out, _ := json.MarshalIndent(results, "", "    ")
		fmt.Println(string(out))
		return nil
	},
}

func buildInspectResult(client *incus.Client, inst *incus.Instance, state *incus.InstanceState) map[string]interface{} {
	created := inst.CreatedAt
	if created != "" {
		created = strings.Replace(created, "T", " ", 1)
		if idx := strings.Index(created, "+"); idx > 0 {
			created = created[:idx]
		}
	}

	status := inst.Status
	running := status == "Running"

	startedAt := ""
	finishedAt := ""
	if state != nil {
		startedAt = state.StartedAt
	}

	labels := make(map[string]string)
	for k, v := range inst.Config {
		if strings.HasPrefix(k, "user.") {
			labels[strings.TrimPrefix(k, "user.")] = v
		}
	}

	envVars := []string{}
	for k, v := range inst.Config {
		if strings.HasPrefix(k, "environment.") {
			envVars = append(envVars, k[12:]+"="+v)
		}
	}

	var mounts []map[string]interface{}
	for name, raw := range inst.ExpandedDevices {
		var dev map[string]interface{}
		if err := json.Unmarshal(raw, &dev); err != nil {
			continue
		}
		if dev["type"] == "disk" && dev["path"] != nil && dev["path"] != "/" {
			src, _ := dev["source"].(string)
			dst, _ := dev["path"].(string)
			mounts = append(mounts, map[string]interface{}{
				"Type":        "volume",
				"Name":        name,
				"Source":      src,
				"Destination": dst,
			})
		}
	}
	if mounts == nil {
		mounts = []map[string]interface{}{}
	}

	networkSettings := map[string]interface{}{
		"Ports": map[string]interface{}{},
	}

	ipAddress := ""
	macAddress := ""
	networks := map[string]interface{}{}
	if state != nil && state.Network != nil {
		for ifName, iface := range state.Network {
			netEntry := map[string]interface{}{
				"IPAddress":           "",
				"IPPrefixLen":         0,
				"GlobalIPv6Address":   "",
				"GlobalIPv6PrefixLen": 0,
				"MacAddress":          iface.Hwaddr,
			}
			for _, addr := range iface.Addresses {
				if addr.Family == "inet" && addr.Scope == "global" {
					netEntry["IPAddress"] = addr.Address
					var preLen int
					fmt.Sscanf(addr.Netmask, "%d", &preLen)
					netEntry["IPPrefixLen"] = preLen
					if ipAddress == "" {
						ipAddress = addr.Address
					}
				}
				if addr.Family == "inet6" && addr.Scope == "global" {
					netEntry["GlobalIPv6Address"] = addr.Address
				}
				if addr.Family == "inet" && addr.Scope == "link" && macAddress == "" {
					macAddress = iface.Hwaddr
				}
			}
			networks[ifName] = netEntry
		}
	}

	return map[string]interface{}{
		"Id":   inst.Name,
		"Name": "/" + inst.Name,
		"State": map[string]interface{}{
			"Status":     status,
			"Running":    running,
			"ExitCode":   0,
			"StartedAt":  startedAt,
			"FinishedAt": finishedAt,
		},
		"Created": created,
		"Config": map[string]interface{}{
			"Image":  "incus:" + inst.Type,
			"User":   "",
			"Env":    envVars,
			"Labels": labels,
		},
		"Mounts": mounts,
		"NetworkSettings": networkSettings,
		"Ports": []interface{}{},
		"Platform":    inst.Architecture,
		"Type":        inst.Type,
		"Description": inst.Description,
		"Profiles":    inst.Profiles,
		"Project":     inst.Project,
	}
}

func init() {
	inspectCmd.Flags().StringVarP(&inspectType, "type", "", "", "Return JSON for specified type")
}
