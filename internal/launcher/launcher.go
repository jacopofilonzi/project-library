// Package launcher apre un progetto con un programma configurato.
package launcher

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/jacopofilonzi/project-library/internal/config"
	"github.com/jacopofilonzi/project-library/internal/platform"
)

var ErrNotFound = errors.New("launcher executable not found")

// Target è ciò che viene aperto, con i valori dei segnaposti.
type Target struct {
	Path   string // {path}
	Name   string // {name}
	Source string // {source}: prima cartella sotto la radice (es. "github")
	Group  string // {group}: cartelle intermedie (es. "jacopofilonzi")
}

// NewTarget calcola i segnaposti di path rispetto alla radice che lo contiene.
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

// SplitArgs divide una riga di argomenti: gli spazi separano, le virgolette doppie raggruppano.
// Il backslash è letterale (percorsi Windows).
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

// Expand divide gli argomenti e sostituisce i segnaposti.
func Expand(args string, t Target) []string {
	r := strings.NewReplacer("{path}", t.Path, "{name}", t.Name, "{source}", t.Source, "{group}", t.Group)
	parts := SplitArgs(args)
	for i, p := range parts {
		parts[i] = r.Replace(p)
	}
	return parts
}

// Command restituisce l'eseguibile di un launcher: quello configurato o, per
// i launcher predefiniti senza percorso, quello rilevato. Vuoto se non trovato.
func Command(l config.Launcher) string {
	if l.Command != "" {
		return platform.Resolve(l.Command)
	}
	if l.Builtin != "" {
		return platform.EditorPath(l.Builtin)
	}
	return ""
}

// Launch apre target con il launcher l.
func Launch(l config.Launcher, t Target) error {
	cmd := Command(l)
	if cmd == "" {
		return ErrNotFound
	}
	return platform.StartDetached(cmd, Expand(l.Args, t), t.Path)
}
