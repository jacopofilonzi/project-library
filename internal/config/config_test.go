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

func TestLoadAddsDefaultRulesToOldConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	// config scritta prima delle regole: il campo manca
	if err := os.WriteFile(path, []byte(`{"version":1,"language":"it","roots":["x"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := Load(path, dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := s.Get().LauncherRules; len(got) != 1 || got[0].Launcher != "intellij" {
		t.Fatalf("default rules not added: %+v", got)
	}
	// una lista vuota salvata dall'utente resta vuota
	if _, err := s.Update(func(c *Config) { c.LauncherRules = []LauncherRule{} }); err != nil {
		t.Fatal(err)
	}
	s2, _ := Load(path, dir)
	if got := s2.Get().LauncherRules; got == nil || len(got) != 0 {
		t.Fatalf("empty rules must stay empty: %+v", got)
	}
}
