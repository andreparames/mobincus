package cmd

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"mobincus/incus"
)

func generateName() string {
	b := make([]byte, 8)
	rand.Read(b)
	return "mobincus-" + hex.EncodeToString(b)
}

func joinCommand(args []string) string {
	return strings.Join(args, "\x00")
}

func splitCommand(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, "\x00")
}

func containerID(name string) string {
	h := sha256.Sum256([]byte(name))
	return hex.EncodeToString(h[:])
}

func resolveName(client *incus.Client, ref string) string {
	inst := resolveRef(client, ref)
	if inst == nil {
		return ref
	}
	return inst.Name
}

func resolveRef(client *incus.Client, ref string) *incus.Instance {
	inst, err := client.GetInstance(ref)
	if err == nil {
		return inst
	}
	containers, err := client.ListContainers()
	if err != nil {
		return nil
	}
	for _, c := range containers {
		if containerID(c.ID) == ref || containerID(c.ID)[:12] == ref {
			inst, err := client.GetInstance(c.ID)
			if err == nil {
				return inst
			}
		}
	}
	return nil
}
