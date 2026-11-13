package elevate

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

const appDirName = "GoogleTranslateHostsUpdater"

func handoffDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, appDirName, "elevate")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

func handoffDonePath(nonce string) (string, error) {
	dir, err := handoffDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "relaunch-"+nonce+".done"), nil
}

// NewRelaunchNonce returns a unique token for elevated relaunch handoff.
func NewRelaunchNonce() string {
	return strconv.FormatInt(time.Now().UnixNano(), 10)
}

// MarkRelaunchDone is called by the new elevated process so the old one can quit.
func MarkRelaunchDone(nonce string) {
	if nonce == "" {
		return
	}
	path, err := handoffDonePath(nonce)
	if err != nil {
		return
	}
	_ = os.WriteFile(path, []byte("ok"), 0644)
}

func waitRelaunchDone(nonce string, timeout time.Duration) error {
	path, err := handoffDonePath(nonce)
	if err != nil {
		return err
	}
	defer os.Remove(path)

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("等待提权重启超时或用户取消了授权提示")
}
