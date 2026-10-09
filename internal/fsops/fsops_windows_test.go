package fsops

import (
	"os"
	"path/filepath"
	"testing"
)

// A file held open by another program blocks the move to the Recycle Bin and the rename:
// the error is "inUse", not a system code.
func TestInUse(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "proj")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join(dir, "open.txt"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if err := Trash(dir, []string{root}); code(err) != "inUse" {
		t.Fatalf("Trash: %v, want inUse", err)
	}
	if _, err := Rename(dir, "other"); code(err) != "inUse" {
		t.Fatalf("Rename: %v, want inUse", err)
	}
}
