package fsops

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func code(err error) string {
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return ""
}

func TestMkdirRename(t *testing.T) {
	root := t.TempDir()
	p, err := Mkdir(root, "Software Engineering")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Mkdir(root, "software engineering"); code(err) != "exists" && code(err) != "" {
		t.Fatalf("duplicate: %v", err)
	}
	if _, err := Mkdir(root, ""); code(err) != "name.empty" {
		t.Fatalf("empty: %v", err)
	}
	if _, err := Mkdir(root, "a/b"); code(err) != "name.separator" {
		t.Fatalf("separator: %v", err)
	}
	q, err := Rename(p, "ISW")
	if err != nil || filepath.Base(q) != "ISW" {
		t.Fatalf("rename: %v %s", err, q)
	}
	if _, err := os.Stat(q); err != nil {
		t.Fatal(err)
	}
	if !IsEmptyDir(q) {
		t.Fatal("new folder should be empty")
	}
}

func TestWithin(t *testing.T) {
	root := filepath.Join(t.TempDir(), "Development")
	if Within(root, []string{root}) {
		t.Error("a root is not within itself")
	}
	if !Within(filepath.Join(root, "local", "x"), []string{root}) {
		t.Error("child must be within")
	}
	if Within(filepath.Join(root+"2", "x"), []string{root}) {
		t.Error("sibling with same prefix is not within")
	}
	if Within(filepath.Join(root, "..", "x"), []string{root}) {
		t.Error(".. escapes the root")
	}
}
