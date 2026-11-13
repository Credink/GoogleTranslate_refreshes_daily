package chromeproxy

import (
	"encoding/json"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

// Known Chrome Web Store / Edge Add-ons / common IDs (best-effort; discovery also
// scans manifests and LevelDB content — Linux sideload / regional IDs often differ).
var knownOmegaIDs = []string{
	"padekgcemlokbadohgkifijomclgjgif", // Proxy SwitchyOmega (legacy CWS)
	"pfnededegaaopdmhkdmcofjmoldfiped", // ZeroOmega / Proxy SwitchyOmega 3 (CWS)
	"jnkelfanjkeadonecabehalmbgkfdbbo", // SwitchyOmega (alt citations)
	"eaddigfhpgjenoabcobkgabbnppanjbo", // ZeroOmega (sideload / regional builds)
	"dmaldhchmoafliphkijbfhaomcgglmgd", // ZeroOmega (Microsoft Edge Add-ons)
}

// ExtensionHit is one Omega-like extension with a writable chrome.storage LevelDB.
type ExtensionHit struct {
	Browser     string // Chrome / Edge / Chromium / Brave / …
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
			if !isChromeProfileDir(name) {
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
				if extName == id || extName == "" {
					if n := omegaNameFromStorage(storage); n != "" {
						extName = n
					}
				}
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

// DiscoveryDiagnostics explains which browser roots were considered (for UI / logs).
type DiscoveryDiagnostics struct {
	HomeDir       string   `json:"home_dir"`
	RootsExisting []string `json:"roots_existing"`
	RootsMissing  []string `json:"roots_missing"`
	ProfileDirs   []string `json:"profile_dirs,omitempty"`
}

// DiagnoseDiscovery returns path roots searched and which exist — used when nothing found.
func DiagnoseDiscovery() DiscoveryDiagnostics {
	home := chromeHomeDir()
	d := DiscoveryDiagnostics{HomeDir: home}
	for _, r := range candidateBrowserRoots(home) {
		line := r.label + ": " + r.path
		if st, err := os.Stat(r.path); err == nil && st.IsDir() {
			d.RootsExisting = append(d.RootsExisting, line)
			entries, _ := os.ReadDir(r.path)
			for _, ent := range entries {
				if ent.IsDir() && isChromeProfileDir(ent.Name()) {
					d.ProfileDirs = append(d.ProfileDirs, filepath.Join(r.path, ent.Name()))
				}
			}
		} else {
			d.RootsMissing = append(d.RootsMissing, line)
		}
	}
	return d
}

type userDataRoot struct {
	label string
	path  string
}

// Test hooks (empty in production).
var (
	forceGOOS string
	forceHome string
)

func currentGOOS() string {
	if forceGOOS != "" {
		return forceGOOS
	}
	return runtime.GOOS
}

func browserUserDataRoots() []userDataRoot {
	home := chromeHomeDir()
	candidates := candidateBrowserRoots(home)
	out := candidates[:0]
	for _, r := range candidates {
		if st, err := os.Stat(r.path); err == nil && st.IsDir() {
			out = append(out, r)
		}
	}
	return out
}

// chromeHomeDir prefers the real desktop user's home when the process was started
// via sudo/pkexec (HOME=/root would miss ~/.config Chrome profiles).
func chromeHomeDir() string {
	if forceHome != "" {
		return forceHome
	}
	goos := currentGOOS()
	if goos != "windows" {
		if u := strings.TrimSpace(os.Getenv("SUDO_USER")); u != "" && os.Geteuid() == 0 {
			if home := homeForUsername(u); home != "" {
				return home
			}
		}
		if uidStr := strings.TrimSpace(os.Getenv("PKEXEC_UID")); uidStr != "" && os.Geteuid() == 0 {
			if uid, err := strconv.Atoi(uidStr); err == nil {
				if home := homeForUID(uid); home != "" {
					return home
				}
			}
		}
	}
	home, _ := os.UserHomeDir()
	return home
}

func homeForUsername(name string) string {
	u, err := user.Lookup(name)
	if err != nil || u.HomeDir == "" {
		return ""
	}
	return u.HomeDir
}

func homeForUID(uid int) string {
	u, err := user.LookupId(strconv.Itoa(uid))
	if err != nil || u.HomeDir == "" {
		return ""
	}
	return u.HomeDir
}

func candidateBrowserRoots(home string) []userDataRoot {
	return candidateBrowserRootsForGOOS(currentGOOS(), home)
}

func candidateBrowserRootsForGOOS(goos, home string) []userDataRoot {
	local := os.Getenv("LOCALAPPDATA")
	var roots []userDataRoot
	add := func(label, path string) {
		if path == "" {
			return
		}
		roots = append(roots, userDataRoot{label: label, path: path})
	}

	switch goos {
	case "windows":
		if local != "" {
			add("Chrome", filepath.Join(local, "Google", "Chrome", "User Data"))
			add("Chrome Beta", filepath.Join(local, "Google", "Chrome Beta", "User Data"))
			add("Edge", filepath.Join(local, "Microsoft", "Edge", "User Data"))
			add("Chromium", filepath.Join(local, "Chromium", "User Data"))
			add("Brave", filepath.Join(local, "BraveSoftware", "Brave-Browser", "User Data"))
		}
	case "darwin":
		if home != "" {
			appSupport := filepath.Join(home, "Library", "Application Support")
			add("Chrome", filepath.Join(appSupport, "Google", "Chrome"))
			add("Chrome Beta", filepath.Join(appSupport, "Google", "Chrome Beta"))
			add("Chrome Canary", filepath.Join(appSupport, "Google", "Chrome Canary"))
			add("Chromium", filepath.Join(appSupport, "Chromium"))
			add("Edge", filepath.Join(appSupport, "Microsoft Edge"))
			add("Brave", filepath.Join(appSupport, "BraveSoftware", "Brave-Browser"))
			add("Brave Beta", filepath.Join(appSupport, "BraveSoftware", "Brave-Browser-Beta"))
		}
	case "linux":
		if home != "" {
			cfg := filepath.Join(home, ".config")
			// Distro / apt / .deb layouts
			add("Chrome", filepath.Join(cfg, "google-chrome"))
			add("Chrome Stable", filepath.Join(cfg, "google-chrome-stable"))
			add("Chrome Beta", filepath.Join(cfg, "google-chrome-beta"))
			add("Chrome Unstable", filepath.Join(cfg, "google-chrome-unstable"))
			add("Chromium", filepath.Join(cfg, "chromium"))
			add("Chromium (chromium-browser)", filepath.Join(cfg, "chromium-browser"))
			add("Edge", filepath.Join(cfg, "microsoft-edge"))
			add("Edge Beta", filepath.Join(cfg, "microsoft-edge-beta"))
			add("Edge Dev", filepath.Join(cfg, "microsoft-edge-dev"))
			add("Brave", filepath.Join(cfg, "BraveSoftware", "Brave-Browser"))
			add("Brave Beta", filepath.Join(cfg, "BraveSoftware", "Brave-Browser-Beta"))
			add("Brave (legacy)", filepath.Join(cfg, "brave"))

			// Flatpak: ~/.var/app/<id>/config/…
			flat := filepath.Join(home, ".var", "app")
			add("Chrome (Flatpak)", filepath.Join(flat, "com.google.Chrome", "config", "google-chrome"))
			add("Chromium (Flatpak)", filepath.Join(flat, "org.chromium.Chromium", "config", "chromium"))
			add("Brave (Flatpak)", filepath.Join(flat, "com.brave.Browser", "config", "BraveSoftware", "Brave-Browser"))
			add("Edge (Flatpak)", filepath.Join(flat, "com.microsoft.Edge", "config", "microsoft-edge"))

			// Snap common data (Chromium / Brave / Edge)
			snap := filepath.Join(home, "snap")
			add("Chromium (Snap)", filepath.Join(snap, "chromium", "common", "chromium"))
			add("Brave (Snap)", filepath.Join(snap, "brave", "common", "brave"))
			add("Edge (Snap)", filepath.Join(snap, "microsoft-edge", "common", "microsoft-edge"))
		}
	}
	return roots
}

func isChromeProfileDir(name string) bool {
	return name == "Default" || strings.HasPrefix(name, "Profile ")
}

func collectOmegaIDs(profileDir string) []string {
	seen := map[string]struct{}{}
	var ids []string
	add := func(id string) {
		id = strings.TrimSpace(id)
		if id == "" || !looksLikeExtensionID(id) {
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
	if entries, err := os.ReadDir(extRoot); err == nil {
		for _, ent := range entries {
			if !ent.IsDir() {
				continue
			}
			id := ent.Name()
			if looksLikeOmega(extensionDisplayName(profileDir, id)) {
				add(id)
			}
		}
	}
	// Also scan Local Extension Settings: Linux unpacked / CRX installs may use IDs
	// not in knownOmegaIDs and without a matching Extensions/*/manifest name yet.
	storageRoot := filepath.Join(profileDir, "Local Extension Settings")
	if entries, err := os.ReadDir(storageRoot); err == nil {
		for _, ent := range entries {
			if !ent.IsDir() {
				continue
			}
			id := ent.Name()
			if !looksLikeExtensionID(id) {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			dir := filepath.Join(storageRoot, id)
			if !isLevelDBDir(dir) {
				continue
			}
			name := extensionDisplayName(profileDir, id)
			if looksLikeOmega(name) || storageLooksLikeOmega(dir) {
				add(id)
			}
		}
	}
	return ids
}

func looksLikeExtensionID(id string) bool {
	if len(id) != 32 {
		return false
	}
	for _, c := range id {
		if c < 'a' || c > 'p' {
			return false
		}
	}
	return true
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
		strings.Contains(n, "proxy switchy") ||
		(strings.Contains(n, "switchy") && strings.Contains(n, "omega"))
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

// storageLooksLikeOmega peeks LevelDB for Omega chrome.storage markers without writing.
func storageLooksLikeOmega(dir string) bool {
	db, err := leveldb.OpenFile(dir, &opt.Options{ReadOnly: true})
	if err != nil {
		return false
	}
	defer db.Close()

	if raw, err := db.Get([]byte("schemaVersion"), nil); err == nil {
		sv := strings.TrimSpace(string(raw))
		if sv == "1" || sv == "2" {
			// Confirm at least one profile-like value.
			iter := db.NewIterator(nil, nil)
			defer iter.Release()
			for iter.Next() {
				key := string(iter.Key())
				if !strings.HasPrefix(key, "+") {
					continue
				}
				val := string(iter.Value())
				if strings.Contains(val, "profileType") &&
					(strings.Contains(val, "FixedProfile") ||
						strings.Contains(val, "SwitchProfile") ||
						strings.Contains(val, "RuleListProfile")) {
					return true
				}
			}
		}
	}
	return false
}

func omegaNameFromStorage(dir string) string {
	if storageLooksLikeOmega(dir) {
		return "ZeroOmega/SwitchyOmega"
	}
	return ""
}
