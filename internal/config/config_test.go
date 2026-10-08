package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRenamePathMovesProjectLaunchers(t *testing.T) {
	root := filepath.Join("C:", "dev")
	old := filepath.Join(root, "local", "a")
	c := Default(root)
	c.ProjectLaunchers[old] = "intellij"
	c.ProjectLaunchers[filepath.Join(old, "sub")] = "vscode"
	c.Overrides[old] = OverrideProject
	c.AddRecent(old, "intellij", 1)

	nw := filepath.Join(root, "local", "b")
	c.RenamePath(old, nw)
	if c.ProjectLaunchers[nw] != "intellij" || c.ProjectLaunchers[filepath.Join(nw, "sub")] != "vscode" || c.Overrides[nw] != OverrideProject || c.Recent[0].Path != nw {
		t.Fatalf("rename not applied: %+v %+v %+v", c.ProjectLaunchers, c.Overrides, c.Recent)
	}
	c.RenamePath(nw, "")
	if len(c.ProjectLaunchers) != 0 || len(c.Overrides) != 0 || len(c.Recent) != 0 {
		t.Fatalf("delete not applied: %+v %+v %+v", c.ProjectLaunchers, c.Overrides, c.Recent)
	}
}

func TestMigrationV1(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{"version":1,"autostart":true,"hotkey":""}`), 0o644); err != nil {
		t.Fatal(err)
	}
	s, _ := Load(path, dir)
	c := s.Get()
	if c.StartMode != StartWindow || c.Autostart || c.SpotlightHotkey == "" || c.Hotkey != "" || c.Version != currentVersion {
		t.Fatalf("migration: %+v", c)
	}
	// after the migration an empty (disabled) spotlight shortcut stays empty
	s.Update(func(c *Config) { c.SpotlightHotkey = "" })
	s2, _ := Load(path, dir)
	if s2.Get().SpotlightHotkey != "" {
		t.Fatal("disabled spotlight hotkey must stay disabled")
	}
}

func TestMigrationV2Rules(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	v2 := `{"version":2,"launcherRules":[
		{"patterns":["pom.xml","build.gradle*","settings.gradle*","gradlew","*.iml"],"launcher":"intellij"},
		{"patterns":["*.ino"],"launcher":"arduino"}]}`
	if err := os.WriteFile(path, []byte(v2), 0o644); err != nil {
		t.Fatal(err)
	}
	s, _ := Load(path, dir)
	c := s.Get()
	// the old default rule becomes the equivalent built-in presets, the others user presets
	want := []Rule{{"gradle", "intellij"}, {"maven", "intellij"}, {"intellij", "intellij"}, {"custom-2", "arduino"}}
	if len(c.Rules) != len(want) {
		t.Fatalf("rules: %+v", c.Rules)
	}
	for i := range want {
		if c.Rules[i] != want[i] {
			t.Fatalf("rule %d: %+v want %+v", i, c.Rules[i], want[i])
		}
	}
	if len(c.Presets) != 1 || c.Presets[0].ID != "custom-2" || c.Presets[0].Patterns[0] != "*.ino" || c.LauncherRules != nil {
		t.Fatalf("presets: %+v", c.Presets)
	}
}

func TestNewConfigHasNoRules(t *testing.T) {
	c := Default(t.TempDir())
	if len(c.Rules) != 0 || len(c.Presets) != 0 {
		t.Fatal("presets must not be pre-applied")
	}
}
