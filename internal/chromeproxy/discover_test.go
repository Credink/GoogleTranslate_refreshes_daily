package chromeproxy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

func seedOmegaLevelDB(dir string) error {
	db, err := leveldb.OpenFile(dir, &opt.Options{})
	if err != nil {
		return err
	}
	defer db.Close()
	if err := db.Put([]byte("schemaVersion"), []byte("2"), nil); err != nil {
		return err
	}
	profile := []byte(`{"name":"proxy","profileType":"FixedProfile","bypassList":[],"revision":"1"}`)
	return db.Put([]byte("+proxy"), profile, nil)
}

func TestLooksLikeExtensionID(t *testing.T) {
	if !looksLikeExtensionID("pfnededegaaopdmhkdmcofjmoldfiped") {
		t.Fatal("CWS ZeroOmega id should match")
	}
	if !looksLikeExtensionID("dmaldhchmoafliphkijbfhaomcgglmgd") {
		t.Fatal("Edge ZeroOmega id should match")
	}
	if looksLikeExtensionID("not-an-id") || looksLikeExtensionID("") {
		t.Fatal("invalid ids must fail")
	}
}

func TestCandidateBrowserRootsLinuxIncludeFlatpakSnap(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	roots := candidateBrowserRootsForGOOS("linux", home)
	joined := strings.Join(rootPaths(roots), "\n")
	wantSub := []string{
		filepath.Join(home, ".config", "google-chrome"),
		filepath.Join(home, ".config", "google-chrome-stable"),
		filepath.Join(home, ".config", "chromium"),
		filepath.Join(home, ".config", "BraveSoftware", "Brave-Browser"),
		filepath.Join(home, ".var", "app", "com.google.Chrome", "config", "google-chrome"),
		filepath.Join(home, "snap", "chromium", "common", "chromium"),
	}
	for _, w := range wantSub {
		if !strings.Contains(joined, w) {
			t.Fatalf("missing linux root %q in:\n%s", w, joined)
		}
	}
}

func TestCandidateBrowserRootsDarwinIncludeBraveEdge(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	roots := candidateBrowserRootsForGOOS("darwin", home)
	joined := strings.Join(rootPaths(roots), "\n")
	wantSub := []string{
		filepath.Join(home, "Library", "Application Support", "Google", "Chrome"),
		filepath.Join(home, "Library", "Application Support", "Google", "Chrome Beta"),
		filepath.Join(home, "Library", "Application Support", "Microsoft Edge"),
		filepath.Join(home, "Library", "Application Support", "BraveSoftware", "Brave-Browser"),
		filepath.Join(home, "Library", "Application Support", "Chromium"),
	}
	for _, w := range wantSub {
		if !strings.Contains(joined, w) {
			t.Fatalf("missing darwin root %q in:\n%s", w, joined)
		}
	}
}

func TestDiscoverOmegaExtensionsFindsSideloadID(t *testing.T) {
	home := t.TempDir()
	// Simulate Linux Chrome Default profile with a non-known Omega storage DB
	// that still has schemaVersion + FixedProfile (via Local Extension Settings scan).
	id := "abcdefghijklmnopabcdefghijklmnop" // 32 chars a-p
	profile := filepath.Join(home, ".config", "google-chrome", "Default")
	storage := filepath.Join(profile, "Local Extension Settings", id)
	if err := os.MkdirAll(storage, 0o755); err != nil {
		t.Fatal(err)
	}
	// Minimal LevelDB-ish marker: CURRENT + empty log is enough for isLevelDBDir;
	// storageLooksLikeOmega needs a real LevelDB — create via patch helper open/write.
	if err := seedOmegaLevelDB(storage); err != nil {
		t.Fatal(err)
	}
	extDir := filepath.Join(profile, "Extensions", id, "1.0.0")
	if err := os.MkdirAll(extDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(extDir, "manifest.json"), []byte(`{"name":"Proxy SwitchyOmega 3 (ZeroOmega)","version":"1.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	oldGOOS := forceGOOS
	forceGOOS = "linux"
	oldHome := forceHome
	forceHome = home
	t.Cleanup(func() {
		forceGOOS = oldGOOS
		forceHome = oldHome
	})

	hits := DiscoverOmegaExtensions()
	if len(hits) == 0 {
		t.Fatalf("expected hit; diag=%+v", DiagnoseDiscovery())
	}
	found := false
	for _, h := range hits {
		if h.ExtensionID == id {
			found = true
			if !strings.Contains(strings.ToLower(h.Name), "omega") {
				t.Fatalf("name=%q", h.Name)
			}
		}
	}
	if !found {
		t.Fatalf("sideload id not found: %+v", hits)
	}
}

func TestDiagnoseDiscoveryListsExisting(t *testing.T) {
	home := t.TempDir()
	chrome := filepath.Join(home, ".config", "google-chrome", "Default")
	if err := os.MkdirAll(chrome, 0o755); err != nil {
		t.Fatal(err)
	}
	oldGOOS := forceGOOS
	forceGOOS = "linux"
	oldHome := forceHome
	forceHome = home
	t.Cleanup(func() {
		forceGOOS = oldGOOS
		forceHome = oldHome
	})
	d := DiagnoseDiscovery()
	if len(d.RootsExisting) == 0 {
		t.Fatalf("expected existing root, got %+v", d)
	}
	if len(d.ProfileDirs) == 0 {
		t.Fatalf("expected profile dir, got %+v", d)
	}
}

func rootPaths(roots []userDataRoot) []string {
	out := make([]string, len(roots))
	for i, r := range roots {
		out[i] = r.path
	}
	return out
}
