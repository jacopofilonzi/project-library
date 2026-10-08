package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jacopofilonzi/project-library/internal/config"
)

// A project renamed and then moved outside the app keeps the editor chosen for it.
func TestRelinkAfterOutsideRenameAndMove(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(home, "Development")
	bot := filepath.Join(root, "local", "bot")
	for _, d := range []string{bot, filepath.Join(root, "github")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(bot, "go.mod"), []byte("module bot\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	store, err := config.Load(filepath.Join(home, "config.json"), home)
	if err != nil {
		t.Fatal(err)
	}
	store.Update(func(c *config.Config) {
		c.Roots = []string{root}
		c.ProjectLaunchers[bot] = "intellij"
	})
	l := New(store)
	l.rescan(false) // records the folder identity

	// outside the app: rename, then move into another folder
	renamed := filepath.Join(root, "local", "robot")
	moved := filepath.Join(root, "github", "robot")
	if err := os.Rename(bot, renamed); err != nil {
		t.Fatal(err)
	}
	l.rescan(false)
	if got := store.Get().ProjectLaunchers[renamed]; got != "intellij" {
		t.Fatalf("after rename: %v", store.Get().ProjectLaunchers)
	}
	if err := os.Rename(renamed, moved); err != nil {
		t.Fatal(err)
	}
	l.rescan(false)
	cfg := store.Get()
	if cfg.ProjectLaunchers[moved] != "intellij" || len(cfg.ProjectLaunchers) != 1 {
		t.Fatalf("after move: %v", cfg.ProjectLaunchers)
	}
}
