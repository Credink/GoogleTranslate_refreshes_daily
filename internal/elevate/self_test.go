package elevate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLooksLikeAppImageMount(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"/tmp/.mount_GoogleXYZ/usr/bin/App", true},
		{"/home/u/.mount_foo/usr/bin/App", true},
		{"/home/u/Downloads/App.AppImage", false},
		{"/usr/bin/GoogleTranslateHostsUpdater", false},
	}
	for _, c := range cases {
		if got := looksLikeAppImageMount(c.in); got != c.want {
			t.Fatalf("looksLikeAppImageMount(%q)=%v want %v", c.in, got, c.want)
		}
	}
}

func TestSelfExecutablePrefersAPPIMAGE(t *testing.T) {
	dir := t.TempDir()
	ai := filepath.Join(dir, "App.AppImage")
	if err := os.WriteFile(ai, []byte("x"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("APPIMAGE", ai)
	got, err := SelfExecutable()
	if err != nil {
		t.Fatal(err)
	}
	want, _ := filepath.Abs(ai)
	if got != want {
		t.Fatalf("SelfExecutable()=%q want %q", got, want)
	}
}

func TestSelfExecutableIgnoresPoisonedAPPIMAGE(t *testing.T) {
	t.Setenv("APPIMAGE", filepath.Join(t.TempDir(), "evil-binary"))
	got, err := SelfExecutable()
	if err != nil {
		t.Fatal(err)
	}
	if looksLikeAppImageFile(got) && filepath.Base(got) == "evil-binary" {
		t.Fatalf("poisoned APPIMAGE used: %q", got)
	}
	if filepath.Base(got) == "evil-binary" {
		t.Fatalf("poisoned APPIMAGE used: %q", got)
	}
}

func TestBuildPkexecArgv(t *testing.T) {
	t.Setenv("DISPLAY", ":0")
	t.Setenv("XAUTHORITY", "/home/u/.Xauthority")
	t.Setenv("APPIMAGE", "/home/u/App.AppImage")
	t.Setenv("APPDIR", "/tmp/.mount_x")
	t.Setenv("OWD", "/home/u")
	argv := buildPkexecArgv("/usr/bin/env", "/home/u/App.AppImage", "--apply-hosts", "/tmp/p.patch")
	if argv[0] != "/usr/bin/env" {
		t.Fatalf("argv[0]=%q", argv[0])
	}
	n := len(argv)
	if n < 4 {
		t.Fatalf("argv too short: %#v", argv)
	}
	if argv[n-3] != "/home/u/App.AppImage" || argv[n-2] != "--apply-hosts" || argv[n-1] != "/tmp/p.patch" {
		t.Fatalf("trailing args = %#v", argv[n-3:])
	}
	need := []string{
		"DISPLAY=:0",
		"XAUTHORITY=/home/u/.Xauthority",
		"APPIMAGE=/home/u/App.AppImage",
		"APPDIR=/tmp/.mount_x",
		"OWD=/home/u",
		"APPIMAGE_EXTRACT_AND_RUN=1",
	}
	for _, s := range need {
		if !containsString(argv, s) {
			t.Fatalf("missing %q in %#v", s, argv)
		}
	}
}

func TestSupportsProcessElevation(t *testing.T) {
	_ = SupportsProcessElevation()
	if PlatformNotes() == "" {
		t.Fatal("PlatformNotes empty")
	}
	if RootGUIWarning == "" {
		t.Fatal("RootGUIWarning empty")
	}
}

func containsString(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}
