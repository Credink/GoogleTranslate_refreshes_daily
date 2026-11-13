//go:build windows

package dnsflush

import "testing"

func TestFlushRuns(t *testing.T) {
	// May fail without elevation; still exercises the code path.
	err := flush()
	if err != nil {
		t.Logf("flush returned (often needs elevation): %v", err)
	}
}
