package main

import (
	"fmt"
	"os"

	"GoogleTranslate_refreshes_daily/internal/hostsfile"
)

// runApplyHosts is the elevated helper entry: write patch file contents to system hosts.
func runApplyHosts(patchPath string) int {
	raw, err := os.ReadFile(patchPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read patch: %v\n", err)
		return 1
	}
	path := hostsfile.Path()
	if _, err := hostsfile.BackupAndWrite(path, string(raw)); err != nil {
		if err2 := os.WriteFile(path, raw, 0644); err2 != nil {
			fmt.Fprintf(os.Stderr, "write hosts: %v / %v\n", err, err2)
			return 2
		}
	}
	_ = os.WriteFile(patchPath+".done", []byte("ok\n"), 0644)
	return 0
}
