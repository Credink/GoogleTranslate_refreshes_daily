package tray

import "testing"

func TestMenuLabelsChinese(t *testing.T) {
	want := map[string]string{
		"show":     "显示窗口",
		"refresh":  "立即刷新",
		"settings": "打开设置",
		"quit":     "退出程序",
	}
	got := map[string]string{
		"show":     MenuLabels.Show,
		"refresh":  MenuLabels.Refresh,
		"settings": MenuLabels.Settings,
		"quit":     MenuLabels.Quit,
	}
	for k, w := range want {
		if got[k] != w {
			t.Fatalf("%s: got %q want %q", k, got[k], w)
		}
	}
}

func TestStatusLine(t *testing.T) {
	if StatusLine("", "") != "尚未检查" {
		t.Fatalf("empty: %q", StatusLine("", ""))
	}
	if StatusLine("ok", "2026-01-01") != "2026-01-01 | ok" {
		t.Fatalf("combo: %q", StatusLine("ok", "2026-01-01"))
	}
}
