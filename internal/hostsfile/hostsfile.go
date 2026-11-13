// Package hostsfile safely updates the system hosts file using a managed marker block.
package hostsfile

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	BeginMarker = "# >>> GoogleTranslateHostsUpdater BEGIN"
	EndMarker   = "# <<< GoogleTranslateHostsUpdater END"
)

// DefaultDomains are the hosts written for the chosen IP (IPFinder save_hosts).
var DefaultDomains = []string{
	"translate.googleapis.com",
	"translate-pa.googleapis.com",
}

// Path returns the OS hosts file path.
func Path() string {
	if runtime.GOOS == "windows" {
		root := os.Getenv("SystemRoot")
		if root == "" {
			root = `C:\Windows`
		}
		return filepath.Join(root, "System32", "drivers", "etc", "hosts")
	}
	return "/etc/hosts"
}

// Mapping is one IP applied to one or more domains.
type Mapping struct {
	IP      string
	Domains []string
}

// CurrentIPFor returns the first non-comment IP mapped to domain in the hosts file, if any.
func CurrentIPFor(content, domain string) string {
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		comment := strings.Index(trimmed, "#")
		effective := trimmed
		if comment >= 0 {
			effective = strings.TrimSpace(trimmed[:comment])
		}
		fields := strings.Fields(effective)
		if len(fields) < 2 {
			continue
		}
		for _, h := range fields[1:] {
			if strings.EqualFold(h, domain) {
				return fields[0]
			}
		}
	}
	return ""
}

// Apply replaces/inserts the managed marker block and comments conflicting unmanaged lines.
func Apply(content string, m Mapping) (string, error) {
	if m.IP == "" {
		return "", fmt.Errorf("empty IP")
	}
	domains := m.Domains
	if len(domains) == 0 {
		domains = DefaultDomains
	}

	lines := splitLines(content)
	lines = stripManagedBlock(lines)
	lines = commentConflicts(lines, domains)

	var block []string
	block = append(block, BeginMarker)
	for _, d := range domains {
		block = append(block, fmt.Sprintf("%s %s", m.IP, d))
	}
	block = append(block, EndMarker)

	// Ensure trailing newline separation
	if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) != "" {
		lines = append(lines, "")
	}
	lines = append(lines, block...)
	return strings.Join(lines, "\n") + "\n", nil
}

// BackupAndWrite copies path to a timestamped backup then writes newContent.
func BackupAndWrite(path, newContent string) (backupPath string, err error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	backupPath = path + ".bak.gthu-" + time.Now().Format("20060102-150405")
	if err := os.WriteFile(backupPath, raw, 0644); err != nil {
		// Fall back to temp dir if beside-hosts fails (permissions).
		dir, _ := os.UserConfigDir()
		if dir == "" {
			dir = os.TempDir()
		}
		altDir := filepath.Join(dir, "GoogleTranslateHostsUpdater", "backups")
		_ = os.MkdirAll(altDir, 0755)
		backupPath = filepath.Join(altDir, "hosts.bak.gthu-"+time.Now().Format("20060102-150405"))
		if err2 := os.WriteFile(backupPath, raw, 0644); err2 != nil {
			return "", fmt.Errorf("backup failed: %v / %v", err, err2)
		}
	}
	if err := os.WriteFile(path, []byte(newContent), 0644); err != nil {
		return backupPath, err
	}
	return backupPath, nil
}

// Read returns hosts file contents.
func Read(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func splitLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	if s == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(s, "\n"), "\n")
}

func stripManagedBlock(lines []string) []string {
	var out []string
	inBlock := false
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == BeginMarker {
			inBlock = true
			continue
		}
		if trim == EndMarker {
			inBlock = false
			continue
		}
		if !inBlock {
			out = append(out, line)
		}
	}
	return out
}

func commentConflicts(lines []string, domains []string) []string {
	want := make(map[string]struct{}, len(domains))
	for _, d := range domains {
		want[strings.ToLower(d)] = struct{}{}
	}
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			out = append(out, line)
			continue
		}
		comment := strings.Index(trimmed, "#")
		effective := trimmed
		if comment >= 0 {
			effective = strings.TrimSpace(trimmed[:comment])
		}
		fields := strings.Fields(effective)
		hit := false
		if len(fields) >= 2 {
			for _, h := range fields[1:] {
				if _, ok := want[strings.ToLower(h)]; ok {
					hit = true
					break
				}
			}
		}
		if hit {
			out = append(out, "# "+line)
		} else {
			out = append(out, line)
		}
	}
	return out
}
