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

func TestMove(t *testing.T) {
	root := t.TempDir()
	for _, d := range []string{"a/proj/src", "b", "c/proj"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	roots := []string{root}
	src := filepath.Join(root, "a", "proj")

	// errors, nothing moves
	cases := map[string][2]string{
		"sameFolder":   {src, filepath.Join(root, "a")},
		"moveInside":   {src, filepath.Join(src, "src")},
		"exists":       {src, filepath.Join(root, "c")},
		"outsideRoots": {src, t.TempDir()},
	}
	for want, c := range cases {
		if _, err := Move(c[0], c[1], roots); code(err) != want {
			t.Errorf("Move(%s → %s) = %v, want %s", c[0], c[1], err, want)
		}
	}
	if _, err := Move(root, filepath.Join(root, "b"), roots); code(err) != "outsideRoots" {
		t.Errorf("moving a root: %v", err)
	}

	// into a folder and back into the root itself
	got, err := Move(src, filepath.Join(root, "b"), roots)
	if err != nil || got != filepath.Join(root, "b", "proj") {
		t.Fatalf("Move = %s, %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(got, "src")); err != nil {
		t.Fatal("content not moved")
	}
	if got, err = Move(got, root, roots); err != nil || got != filepath.Join(root, "proj") {
		t.Fatalf("Move to root = %s, %v", got, err)
	}
}
