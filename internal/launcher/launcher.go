// Package launcher opens a project with a configured program.
package launcher

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/jacopofilonzi/project-library/internal/config"
	"github.com/jacopofilonzi/project-library/internal/platform"
)

var ErrNotFound = errors.New("launcher executable not found")

// Target is what gets opened, with the placeholder values.
type Target struct {
	Path   string // {path}
	Name   string // {name}
	Source string // {source}: first folder under the root (e.g. "github")
	Group  string // {group}: intermediate folders (e.g. "jacopofilonzi")
}

// NewTarget computes the placeholders of path relative to the root that contains it.
func NewTarget(path string, roots []string) Target {
	t := Target{Path: path, Name: filepath.Base(path)}
	for _, r := range roots {
		rel, err := filepath.Rel(r, path)
		if err != nil || strings.HasPrefix(rel, "..") {
			continue
		}
		segs := strings.Split(filepath.ToSlash(rel), "/")
		if len(segs) > 1 {
			t.Source = segs[0]
			t.Group = strings.Join(segs[1:len(segs)-1], "/")
		}
		break
	}
	return t
}

// SplitArgs splits an argument line: spaces separate, double quotes group.
// Backslashes are literal (Windows paths).
func SplitArgs(s string) []string {
	var out []string
	var cur strings.Builder
	inQuote, has := false, false
	for _, r := range s {
		switch {
		case r == '"':
			inQuote = !inQuote
			has = true
		case (r == ' ' || r == '\t') && !inQuote:
			if has {
				out = append(out, cur.String())
				cur.Reset()
				has = false
			}
		default:
			cur.WriteRune(r)
			has = true
		}
	}
	if has {
		out = append(out, cur.String())
	}
	return out
}

// Expand splits the arguments and replaces the placeholders.
func Expand(args string, t Target) []string {
	r := strings.NewReplacer("{path}", t.Path, "{name}", t.Name, "{source}", t.Source, "{group}", t.Group)
	parts := SplitArgs(args)
	for i, p := range parts {
		parts[i] = r.Replace(p)
	}
	return parts
}

// Command returns the executable of a launcher: the configured one or, for
// built-in launchers without a path, the detected one. Empty if not found.
func Command(l config.Launcher) string {
	if l.Command != "" {
		return platform.Resolve(l.Command)
	}
	if l.Builtin != "" {
		return platform.EditorPath(l.Builtin)
	}
	return ""
}

// Launch opens target with the launcher l.
func Launch(l config.Launcher, t Target) error {
	cmd := Command(l)
	if cmd == "" {
		return ErrNotFound
	}
	return platform.StartDetached(cmd, Expand(l.Args, t), t.Path)
}
