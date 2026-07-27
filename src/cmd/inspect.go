package cmd

import (
	"crypto/sha256"
	"encoding/hex"
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
				inst = resolveByHash(client, ref)
			}
			if inst == nil {
				fmt.Println("[]")
				return &StatusError{
					StatusCode: 1,
					Status:     fmt.Sprintf("error: no such object: %s", ref),
				}
			}

			state, _ := client.GetInstanceState(ref)

			results = append(results, buildInspectResult(inst, state))
		}

		out, _ := json.MarshalIndent(results, "", "    ")
		fmt.Println(string(out))
		return nil
	},
}

func containerID(name string) string {
	h := sha256.Sum256([]byte(name))
	return hex.EncodeToString(h[:])
}

func dockerTime(t string) string {
	if t == "" {
		return "0001-01-01T00:00:00Z"
	}
	t = strings.Replace(t, " ", "T", 1)
	if idx := strings.Index(t, "+"); idx > 0 {
		t = t[:idx] + "Z"
	}
	return t
}

func buildInspectResult(inst *incus.Instance, state *incus.InstanceState) map[string]interface{} {
	id := containerID(inst.Name)
	status := strings.ToLower(inst.Status)
	running := inst.Status == "Running"

	startedAt := ""
	finishedAt := "0001-01-01T00:00:00Z"
	pid := 0
	if state != nil {
		startedAt = state.StartedAt
		pid = state.PID
	}
	if !running {
		finishedAt = dockerTime(inst.CreatedAt)
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
	if len(envVars) == 0 {
		envVars = []string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"}
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
				"Driver":      "incus",
				"Mode":        "",
				"RW":          true,
				"Propagation": "",
			})
		}
	}
	if mounts == nil {
		mounts = []map[string]interface{}{}
	}

	networks := map[string]interface{}{}
	ipAddress := ""
	ipPrefixLen := 0
	gateway := ""
	macAddress := ""
	globalIPv6 := ""
	globalIPv6PrefixLen := 0
	ipV6Gateway := ""
	if state != nil && state.Network != nil {
		for ifName, iface := range state.Network {
			netEntry := map[string]interface{}{
				"IPAddress":           "",
				"IPPrefixLen":         0,
				"GlobalIPv6Address":   "",
				"GlobalIPv6PrefixLen": 0,
				"MacAddress":          iface.Hwaddr,
				"Gateway":             "",
				"IPv6Gateway":         "",
			}
			for _, addr := range iface.Addresses {
				if addr.Family == "inet" && addr.Scope == "global" {
					netEntry["IPAddress"] = addr.Address
					var pl int
					fmt.Sscanf(addr.Netmask, "%d", &pl)
					netEntry["IPPrefixLen"] = pl
					if ipAddress == "" {
						ipAddress = addr.Address
						ipPrefixLen = pl
					}
				}
				if addr.Family == "inet6" && addr.Scope == "global" {
					netEntry["GlobalIPv6Address"] = addr.Address
					if globalIPv6 == "" {
						globalIPv6 = addr.Address
					}
				}
				if addr.Family == "inet" && macAddress == "" {
					macAddress = iface.Hwaddr
				}
			}
			if ifName == "eth0" {
				networks["bridge"] = netEntry
			} else {
				networks[ifName] = netEntry
			}
		}
	}

	networkSettings := map[string]interface{}{
		"Bridge":                 "",
		"SandboxID":              "",
		"HairpinMode":            false,
		"LinkLocalIPv6Address":   "",
		"LinkLocalIPv6PrefixLen": 0,
		"Ports":                  map[string]interface{}{},
		"SandboxKey":             "",
		"SecondaryIPAddresses":   nil,
		"SecondaryIPv6Addresses": nil,
		"EndpointID":             "",
		"Gateway":                gateway,
		"GlobalIPv6Address":      globalIPv6,
		"GlobalIPv6PrefixLen":    globalIPv6PrefixLen,
		"IPAddress":              ipAddress,
		"IPPrefixLen":            ipPrefixLen,
		"IPv6Gateway":            ipV6Gateway,
		"MacAddress":             macAddress,
		"Networks":               networks,
	}

	created := dockerTime(inst.CreatedAt)

	return map[string]interface{}{
		"Id":     id,
		"Name":   "/" + inst.Name,
		"Created": created,
		"Path":   "",
		"Args":   []string{},
		"State": map[string]interface{}{
			"Status":     status,
			"Running":    running,
			"Paused":     false,
			"Restarting": false,
			"OOMKilled":  false,
			"Dead":       false,
			"Pid":        pid,
			"ExitCode":   0,
			"Error":      "",
			"StartedAt":  dockerTime(startedAt),
			"FinishedAt": finishedAt,
		},
		"Image": "sha256:" + id,
		"ResolvConfPath": "",
		"HostnamePath":   "",
		"HostsPath":      "",
		"LogPath":        "",
		"RestartCount":    0,
		"Driver":          "incus",
		"Platform":        "linux",
		"MountLabel":      "",
		"ProcessLabel":    "",
		"AppArmorProfile": "",
		"ExecIDs":         []string{},
		"HostConfig": map[string]interface{}{
			"Binds":           nil,
			"ContainerIDFile": "",
			"LogConfig": map[string]interface{}{
				"Type":   "json-file",
				"Config": map[string]interface{}{},
			},
			"NetworkMode":      "bridge",
			"PortBindings":     map[string]interface{}{},
			"RestartPolicy": map[string]interface{}{
				"Name":              "no",
				"MaximumRetryCount": 0,
			},
			"AutoRemove":       false,
			"VolumeDriver":     "",
			"VolumesFrom":      nil,
			"CapAdd":           nil,
			"CapDrop":          nil,
			"Dns":              []string{},
			"DnsOptions":       []string{},
			"DnsSearch":        []string{},
			"ExtraHosts":       nil,
			"GroupAdd":         nil,
			"IpcMode":          "private",
			"Cgroup":           "",
			"Links":            nil,
			"OomScoreAdj":      0,
			"PidMode":          "",
			"Privileged":       false,
			"PublishAllPorts":  false,
			"ReadonlyRootfs":   false,
			"SecurityOpt":      nil,
			"UTSMode":          "",
			"UsernsMode":       "",
			"ShmSize":          0,
			"Runtime":          "runc",
			"ConsoleSize":      []int{0, 0},
			"Isolation":        "",
			"CpuShares":        0,
			"Memory":           0,
			"NanoCpus":         0,
			"CgroupParent":     "",
			"BlkioWeight":      0,
			"BlkioWeightDevice": nil,
			"BlkioDeviceReadBps":   nil,
			"BlkioDeviceWriteBps":  nil,
			"BlkioDeviceReadIOps":  nil,
			"BlkioDeviceWriteIOps": nil,
			"CpuPeriod":             0,
			"CpuQuota":              0,
			"CpuRealtimePeriod":     0,
			"CpuRealtimeRuntime":    0,
			"CpusetCpus":            "",
			"CpusetMems":            "",
			"Devices":               []interface{}{},
			"DeviceCgroupRules":     nil,
			"DeviceRequests":        nil,
			"KernelMemory":          0,
			"KernelMemoryTCP":       0,
			"MemoryReservation":     0,
			"MemorySwap":            0,
			"MemorySwappiness":      0,
			"OomKillDisable":        false,
			"PidsLimit":             0,
			"Ulimits":               nil,
			"CpuCount":              0,
			"CpuPercent":            0,
			"IOMaximumIOps":         0,
			"IOMaximumBandwidth":    0,
			"Tmpfs":                 map[string]interface{}{},
		},
		"GraphDriver": map[string]interface{}{
			"Data": map[string]interface{}{
				"LowerDir": "",
				"MergedDir": "",
				"UpperDir": "",
				"WorkDir":  "",
			},
			"Name": "incus",
		},
		"SizeRw":     nil,
		"SizeRootFs": nil,
		"Config": map[string]interface{}{
			"Hostname":       id[:12],
			"Domainname":     "",
			"User":           "",
			"AttachStdin":    false,
			"AttachStdout":   false,
			"AttachStderr":   false,
			"ExposedPorts":   nil,
			"Tty":            false,
			"OpenStdin":      false,
			"StdinOnce":      false,
			"Env":            envVars,
			"Cmd":            []string{},
			"Healthcheck":    nil,
			"ArgsEscaped":    false,
			"Image":          "incus:" + inst.Type,
			"Volumes":        nil,
			"VolumeDriver":   "",
			"WorkingDir":     "",
			"Entrypoint":     nil,
			"NetworkDisabled": false,
			"MacAddress":     "",
			"OnBuild":        nil,
			"Labels":         labels,
			"StopSignal":     "SIGTERM",
			"StopTimeout":    0,
			"Shell":          nil,
		},
		"NetworkSettings": networkSettings,
		"Mounts":          mounts,
	}
}

func resolveByHash(client *incus.Client, hash string) *incus.Instance {
	containers, err := client.ListContainers()
	if err != nil {
		return nil
	}
	for _, c := range containers {
		if containerID(c.ID) == hash {
			inst, err := client.GetInstance(c.ID)
			if err == nil {
				return inst
			}
		}
	}
	return nil
}

func init() {
	inspectCmd.Flags().StringVarP(&inspectType, "type", "", "", "Return JSON for specified type")
}
