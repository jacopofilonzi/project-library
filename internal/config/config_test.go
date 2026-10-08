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

func TestMigrationV3TurnsOnUpdateCheck(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{"version":3,"setupDone":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	s, _ := Load(path, dir)
	if !s.Get().CheckUpdates {
		t.Fatal("update check must be on after the migration")
	}
	// once migrated, turning it off sticks
	s.Update(func(c *Config) { c.CheckUpdates = false })
	s2, _ := Load(path, dir)
	if s2.Get().CheckUpdates {
		t.Fatal("disabled update check must stay disabled")
	}
}

func TestRelinkFollowsFoldersMovedOutside(t *testing.T) {
	sep := string(filepath.Separator)
	p := func(s string) string { return sep + "dev" + sep + filepath.FromSlash(s) }
	// the disk: path → identity
	disk := map[string]string{p("a/bot"): "1", p("a/ui"): "2", p("a/ui/web"): "3", p("b"): "4"}
	exists := func(s string) bool { _, ok := disk[s]; return ok }
	idOf := func(s string) string { return disk[s] }
	cands := func() []string {
		var out []string
		for k := range disk {
			out = append(out, k)
		}
		return out
	}

	c := Default(sep)
	c.ProjectLaunchers = map[string]string{p("a/bot"): "intellij", p("a/ui/web"): "vscode"}
	c.Overrides = map[string]string{p("a/ui"): OverrideDir}
	c.Recent = []Recent{{Path: p("a/bot"), Launcher: "intellij"}}
	if !c.Relink(exists, idOf, cands()) || len(c.FolderIDs) != 3 {
		t.Fatalf("identities not recorded: %v", c.FolderIDs)
	}

	// outside the app: bot renamed, ui (with web inside) moved under b
	disk = map[string]string{p("a/robot"): "1", p("b"): "4", p("b/ui"): "2", p("b/ui/web"): "3"}
	if !c.Relink(exists, idOf, cands()) {
		t.Fatal("nothing relinked")
	}
	if c.ProjectLaunchers[p("a/robot")] != "intellij" || c.ProjectLaunchers[p("b/ui/web")] != "vscode" || c.Overrides[p("b/ui")] != OverrideDir {
		t.Fatalf("settings not moved: %v %v", c.ProjectLaunchers, c.Overrides)
	}
	if len(c.ProjectLaunchers) != 2 || len(c.Overrides) != 1 || c.Recent[0].Path != p("a/robot") {
		t.Fatalf("old paths left: %v %v %v", c.ProjectLaunchers, c.Overrides, c.Recent)
	}
	if c.FolderIDs[p("a/robot")] != "1" || c.FolderIDs[p("b/ui/web")] != "3" || len(c.FolderIDs) != 3 {
		t.Fatalf("identities: %v", c.FolderIDs)
	}
	// stable: a second pass changes nothing
	if c.Relink(exists, idOf, cands()) {
		t.Fatal("second pass changed the config")
	}

	// a folder that disappeared for good keeps its settings (it may come back) and is not relinked elsewhere
	delete(disk, p("a/robot"))
	c.Relink(exists, idOf, cands())
	if c.ProjectLaunchers[p("a/robot")] != "intellij" {
		t.Fatal("settings of a missing folder dropped")
	}
}

func TestUIScaleNormalized(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	for in, want := range map[string]float64{`{"version":4}`: 1, `{"version":4,"uiScale":3}`: MaxUIScale, `{"version":4,"uiScale":0.1}`: MinUIScale, `{"version":4,"uiScale":1.234}`: 1.23} {
		if err := os.WriteFile(path, []byte(in), 0o644); err != nil {
			t.Fatal(err)
		}
		s, _ := Load(path, dir)
		if got := s.Get().UIScale; got != want {
			t.Errorf("%s → %v, want %v", in, got, want)
		}
	}
}
