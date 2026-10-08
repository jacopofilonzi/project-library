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

// Move moves path (a folder or project) into the folder dest, keeping its name.
// path must be inside the roots; dest can be a root or a folder inside one. Moving between
// disks is refused: it would be a copy and a delete, not a move.
func Move(path, dest string, roots []string) (string, error) {
	path, dest = filepath.Clean(path), filepath.Clean(dest)
	if !Within(path, roots) || !(Within(dest, roots) || isRoot(dest, roots)) {
		return "", &Error{Code: "outsideRoots"}
	}
	if samePath(filepath.Dir(path), dest) {
		return "", &Error{Code: "sameFolder"}
	}
	if samePath(path, dest) || inside(dest, path) {
		return "", &Error{Code: "moveInside"}
	}
	fi, err := os.Stat(dest)
	if err != nil || !fi.IsDir() {
		return "", fail("io", err)
	}
	name := filepath.Base(path)
	if exists(dest, name) {
		return "", &Error{Code: "exists"}
	}
	target := filepath.Join(dest, name)
	if err := os.Rename(path, target); err != nil {
		if platform.CrossDevice(err) {
			return "", &Error{Code: "crossDevice"}
		}
		return "", fail("io", err)
	}
	return target, nil
}

func isRoot(path string, roots []string) bool {
	for _, r := range roots {
		if samePath(path, r) {
			return true
		}
	}
	return false
}

// samePath compares two paths, case-insensitively where the file system is.
func samePath(a, b string) bool {
	a, b = filepath.Clean(a), filepath.Clean(b)
	if platform.CaseInsensitive() {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// inside tells whether p is strictly inside dir.
func inside(p, dir string) bool {
	if platform.CaseInsensitive() {
		p, dir = strings.ToLower(p), strings.ToLower(dir)
	}
	rel, err := filepath.Rel(dir, p)
	return err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}
