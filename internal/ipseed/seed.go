// Package ipseed embeds a bundled IPv4 candidate snapshot (from gtdb public lists).
package ipseed

import (
	_ "embed"
	"strings"
)

//go:embed ips.txt
var raw string

// IPs returns the embedded candidate list (whitespace-separated).
func IPs() []string {
	fields := strings.Fields(raw)
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if f != "" {
			out = append(out, f)
		}
	}
	return out
}
