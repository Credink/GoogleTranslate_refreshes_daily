package chromeproxy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Known Chrome Web Store / common IDs (best-effort; discovery also scans manifests).
var knownOmegaIDs = []string{
	"padekgcemlokbadohgkifijomclgjgif", // Proxy SwitchyOmega
	"pfnededegaaopdmhkdmcofjmoldfiped", // ZeroOmega (CWS)
	"jnkelfanjkeadonecabehalmbgkfdbbo", // SwitchyOmega (alt citations)
	"eaddigfhpgjenoabcobkgabbnppanjbo", // ZeroOmega (sideload / regional builds)
}

// ExtensionHit is one Omega-like extension with a writable chrome.storage LevelDB.
type ExtensionHit struct {
	Browser     string // Chrome / Edge / Chromium / Chrome Beta
	Profile     string // Default / Profile 1 / …
	ExtensionID string
	Name        string
	StorageDir  string
}

// DiscoverOmegaExtensions finds SwitchyOmega / ZeroOmega Local Extension Settings DBs.
func DiscoverOmegaExtensions() []ExtensionHit {
	var hits []ExtensionHit
	seen := map[string]struct{}{}
	for _, root := range browserUserDataRoots() {
		browser := root.label
		entries, err := os.ReadDir(root.path)
		if err != nil {
			continue
		}
		for _, ent := range entries {
			if !ent.IsDir() {
				continue
			}
			name := ent.Name()
			if name != "Default" && !strings.HasPrefix(name, "Profile ") {
				continue
			}
			profileDir := filepath.Join(root.path, name)
			for _, id := range collectOmegaIDs(profileDir) {
				storage := filepath.Join(profileDir, "Local Extension Settings", id)
				if !isLevelDBDir(storage) {
					continue
				}
				key := storage
				if _, ok := seen[key]; ok {
					continue
				}
				seen[key] = struct{}{}
				extName := extensionDisplayName(profileDir, id)
				hits = append(hits, ExtensionHit{
					Browser:     browser,
					Profile:     name,
					ExtensionID: id,
					Name:        extName,
					StorageDir:  storage,
				})
			}
		}
	}
	return hits
}

type userDataRoot struct {
	label string
	path  string
}

func browserUserDataRoots() []userDataRoot {
	local := os.Getenv("LOCALAPPDATA")
	home, _ := os.UserHomeDir()
	var roots []userDataRoot
	if runtime.GOOS == "windows" && local != "" {
		roots = append(roots,
			userDataRoot{"Chrome", filepath.Join(local, "Google", "Chrome", "User Data")},
			userDataRoot{"Chrome Beta", filepath.Join(local, "Google", "Chrome Beta", "User Data")},
			userDataRoot{"Edge", filepath.Join(local, "Microsoft", "Edge", "User Data")},
			userDataRoot{"Chromium", filepath.Join(local, "Chromium", "User Data")},
		)
	}
	if runtime.GOOS == "darwin" && home != "" {
		roots = append(roots,
			userDataRoot{"Chrome", filepath.Join(home, "Library", "Application Support", "Google", "Chrome")},
			userDataRoot{"Edge", filepath.Join(home, "Library", "Application Support", "Microsoft Edge")},
			userDataRoot{"Chromium", filepath.Join(home, "Library", "Application Support", "Chromium")},
		)
	}
	if runtime.GOOS == "linux" && home != "" {
		roots = append(roots,
			userDataRoot{"Chrome", filepath.Join(home, ".config", "google-chrome")},
			userDataRoot{"Chromium", filepath.Join(home, ".config", "chromium")},
			userDataRoot{"Edge", filepath.Join(home, ".config", "microsoft-edge")},
		)
	}
	out := roots[:0]
	for _, r := range roots {
		if st, err := os.Stat(r.path); err == nil && st.IsDir() {
			out = append(out, r)
		}
	}
	return out
}

func collectOmegaIDs(profileDir string) []string {
	seen := map[string]struct{}{}
	var ids []string
	add := func(id string) {
		id = strings.TrimSpace(id)
		if id == "" {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	for _, id := range knownOmegaIDs {
		add(id)
	}
	extRoot := filepath.Join(profileDir, "Extensions")
	entries, err := os.ReadDir(extRoot)
	if err != nil {
		return ids
	}
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		id := ent.Name()
		if looksLikeOmega(extensionDisplayName(profileDir, id)) {
			add(id)
		}
	}
	return ids
}

func extensionDisplayName(profileDir, id string) string {
	extDir := filepath.Join(profileDir, "Extensions", id)
	manifests, _ := filepath.Glob(filepath.Join(extDir, "*", "manifest.json"))
	if len(manifests) == 0 {
		manifests, _ = filepath.Glob(filepath.Join(extDir, "*", "_locales", "en", "messages.json"))
	}
	for _, m := range manifests {
		if strings.HasSuffix(m, "manifest.json") {
			b, err := os.ReadFile(m)
			if err != nil {
				continue
			}
			var meta struct {
				Name string `json:"name"`
			}
			if json.Unmarshal(b, &meta) == nil && meta.Name != "" {
				if strings.HasPrefix(meta.Name, "__MSG_") {
					return resolveOmegaLocaleName(filepath.Dir(m), meta.Name)
				}
				return meta.Name
			}
		}
	}
	return id
}

func resolveOmegaLocaleName(versionDir, msgKey string) string {
	key := strings.TrimSuffix(strings.TrimPrefix(msgKey, "__MSG_"), "__")
	for _, loc := range []string{"zh_CN", "en", "en_US"} {
		p := filepath.Join(versionDir, "_locales", loc, "messages.json")
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var msgs map[string]struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(b, &msgs) != nil {
			continue
		}
		if m, ok := msgs[key]; ok && m.Message != "" {
			return m.Message
		}
	}
	return msgKey
}

func looksLikeOmega(name string) bool {
	n := strings.ToLower(name)
	return strings.Contains(n, "zeroomega") ||
		strings.Contains(n, "switchyomega") ||
		strings.Contains(n, "proxy switchy")
}

func isLevelDBDir(dir string) bool {
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		return false
	}
	// CURRENT is the LevelDB pointer file.
	if _, err := os.Stat(filepath.Join(dir, "CURRENT")); err == nil {
		return true
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		name := e.Name()
		if strings.HasSuffix(name, ".ldb") || strings.HasSuffix(name, ".log") {
			return true
		}
	}
	return false
}
