// Package fsops creates, renames and deletes (to the trash) folders.
// Errors are *Error with a code the frontend translates.
package fsops

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/jacopofilonzi/project-library/internal/platform"
)

// Error carries a stable code ("name.badChars", "exists"…) and the technical detail.
type Error struct {
	Code   string
	Detail string
}

func (e *Error) Error() string {
	if e.Detail == "" {
		return e.Code
	}
	return e.Code + ": " + e.Detail
}

func fail(code string, err error) error {
	if err == nil {
		return &Error{Code: code}
	}
	return &Error{Code: code, Detail: err.Error()}
}

// CheckName checks that name is a valid and free name in parent.
func CheckName(parent, name string) error {
	if p := platform.NameProblem(name); p != "" {
		return &Error{Code: "name." + p}
	}
	if exists(parent, name) {
		return &Error{Code: "exists"}
	}
	return nil
}

// exists compares names case-insensitively where the file system is case-insensitive.
func exists(parent, name string) bool {
	if _, err := os.Lstat(filepath.Join(parent, name)); err == nil {
		return true
	}
	if !platform.CaseInsensitive() {
		return false
	}
	entries, _ := os.ReadDir(parent)
	for _, e := range entries {
		if strings.EqualFold(e.Name(), name) {
			return true
		}
	}
	return false
}

// Mkdir creates parent/name.
func Mkdir(parent, name string) (string, error) {
	name = strings.TrimSpace(name)
	if err := CheckName(parent, name); err != nil {
		return "", err
	}
	path := filepath.Join(parent, name)
	if err := os.Mkdir(path, 0o755); err != nil {
		return "", fail("io", err)
	}
	return path, nil
}

// Rename renames path to newName in the same folder. Changing only the case is allowed.
func Rename(path, newName string) (string, error) {
	newName = strings.TrimSpace(newName)
	parent := filepath.Dir(path)
	if newName == filepath.Base(path) {
		return path, nil
	}
	if p := platform.NameProblem(newName); p != "" {
		return "", &Error{Code: "name." + p}
	}
	if !strings.EqualFold(newName, filepath.Base(path)) && exists(parent, newName) {
		return "", &Error{Code: "exists"}
	}
	dest := filepath.Join(parent, newName)
	if err := os.Rename(path, dest); err != nil {
		return "", fail("io", err)
	}
	return dest, nil
}

// IsEmptyDir: the folder contains nothing (not even hidden files).
func IsEmptyDir(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	_, err = f.Readdirnames(1)
	return errors.Is(err, io.EOF)
}

// Trash moves path to the trash. It refuses the roots and paths outside the roots.
func Trash(path string, roots []string) error {
	if !Within(path, roots) {
		return &Error{Code: "outsideRoots"}
	}
	if err := platform.MoveToTrash(path); err != nil {
		return fail("trash", err)
	}
	if _, err := os.Lstat(path); err == nil {
		return fail("trash", fmt.Errorf("%s still exists", path))
	}
	return nil
}

// Within tells whether path is strictly inside one of the roots (not a root itself).
func Within(path string, roots []string) bool {
	path = filepath.Clean(path)
	for _, r := range roots {
		rel, err := filepath.Rel(filepath.Clean(r), path)
		if err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel) {
			return true
		}
	}
	return false
}
