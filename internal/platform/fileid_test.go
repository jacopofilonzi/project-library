package platform

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileIDSurvivesRenameAndMove(t *testing.T) {
	root := t.TempDir()
	a := filepath.Join(root, "a")
	other := filepath.Join(root, "other")
	for _, d := range []string{filepath.Join(a, "src"), other} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	id := FileID(a)
	if id == "" {
		t.Fatal("no id")
	}
	if FileID(other) == id {
		t.Fatal("two folders with the same id")
	}
	moved := filepath.Join(other, "renamed")
	if err := os.Rename(a, moved); err != nil {
		t.Fatal(err)
	}
	if got := FileID(moved); got != id {
		t.Fatalf("id changed after the move: %s → %s", id, got)
	}
	if FileID(a) != "" {
		t.Fatal("id for a path that no longer exists")
	}
}
