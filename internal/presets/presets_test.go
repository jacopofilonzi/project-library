package presets

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCatalogIDsAreUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, p := range Catalog() {
		if seen[p.ID] || p.ID == "" || len(p.Patterns) == 0 {
			t.Fatalf("bad preset %+v", p)
		}
		seen[p.ID] = true
	}
}

func TestMatches(t *testing.T) {
	dir := t.TempDir()
	android := filepath.Join(dir, "app", "src", "main")
	if err := os.MkdirAll(android, 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(android, "AndroidManifest.xml"), nil, 0o644)
	os.WriteFile(filepath.Join(dir, "build.gradle.kts"), nil, 0o644)
	names := []string{"app", "build.gradle.kts"}

	a, _ := Find("android", nil)
	g, _ := Find("gradle", nil)
	n, _ := Find("node", nil)
	if !Matches(dir, names, a.Patterns) || !Matches(dir, names, g.Patterns) || Matches(dir, names, n.Patterns) {
		t.Fatal("android project must match android and gradle, not node")
	}
	if !Matches(dir, names, []string{"*/src/main/AndroidManifest.xml"}) {
		t.Fatal("glob in path segments")
	}
	if !Matches(dir, names, []string{"BUILD.gradle*"}) {
		t.Fatal("name match is case-insensitive")
	}
}

func TestFindUserPreset(t *testing.T) {
	user := []Preset{{ID: "u1", Name: "Mine", Patterns: []string{"x"}}}
	if p, ok := Find("u1", user); !ok || p.Builtin {
		t.Fatal("user preset")
	}
	if p, ok := Find("go", user); !ok || !p.Builtin {
		t.Fatal("builtin preset")
	}
	if _, ok := Find("nope", user); ok {
		t.Fatal("unknown preset")
	}
}
