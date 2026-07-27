package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
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
