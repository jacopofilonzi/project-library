// Package fsops crea, rinomina ed elimina (nel Cestino) le cartelle.
// Gli errori sono *Error con un codice che il frontend traduce.
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

// Error porta un codice stabile ("name.badChars", "exists"…) e il dettaglio tecnico.
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

// CheckName verifica che name sia un nome valido e libero in parent.
func CheckName(parent, name string) error {
	if p := platform.NameProblem(name); p != "" {
		return &Error{Code: "name." + p}
	}
	if exists(parent, name) {
		return &Error{Code: "exists"}
	}
	return nil
}

// exists confronta i nomi senza distinguere maiuscole dove il file system non le distingue.
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

// Mkdir crea parent/name.
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

// Rename rinomina path in newName nella stessa cartella. Consente di cambiare solo le maiuscole.
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

// IsEmptyDir: la cartella non contiene niente (nemmeno file nascosti).
func IsEmptyDir(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	_, err = f.Readdirnames(1)
	return errors.Is(err, io.EOF)
}

// Trash sposta path nel Cestino. Rifiuta le radici e i percorsi fuori dalle radici.
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

// Within dice se path è strettamente dentro una delle radici (non una radice stessa).
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
