package assets

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func writeAsset(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestURL_AddsContentVersion(t *testing.T) {
	dir := t.TempDir()
	writeAsset(t, dir, "js/app.min.js", "console.log(1)")

	got := NewVersioner(dir, "/static").URL("js/app.min.js")
	if !regexp.MustCompile(`^/static/js/app\.min\.js\?v=[0-9a-f]{12}$`).MatchString(got) {
		t.Errorf("URL = %q, want /static/js/app.min.js?v=<12 hex chars>", got)
	}
}

func TestURL_VersionChangesWithContent(t *testing.T) {
	dirA, dirB := t.TempDir(), t.TempDir()
	writeAsset(t, dirA, "css/styles.css", "body{color:red}")
	writeAsset(t, dirB, "css/styles.css", "body{color:blue}")

	a := NewVersioner(dirA, "/static").URL("css/styles.css")
	b := NewVersioner(dirB, "/static").URL("css/styles.css")
	if a == b {
		t.Errorf("different content produced the same URL %q", a)
	}
}

func TestURL_CachesVersionUntilRestart(t *testing.T) {
	dir := t.TempDir()
	writeAsset(t, dir, "js/app.min.js", "one")
	v := NewVersioner(dir, "/static")
	first := v.URL("js/app.min.js")

	writeAsset(t, dir, "js/app.min.js", "two")
	if got := v.URL("js/app.min.js"); got != first {
		t.Errorf("URL after rewrite = %q, want cached %q", got, first)
	}
}

func TestURL_MissingFileFallsBackToUnversioned(t *testing.T) {
	v := NewVersioner(t.TempDir(), "/static")
	if got := v.URL("js/missing.js"); got != "/static/js/missing.js" {
		t.Errorf("URL = %q, want /static/js/missing.js", got)
	}
}
