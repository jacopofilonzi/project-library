// Package languages calcola la composizione dei linguaggi di un progetto (come la barra di GitHub):
// byte per linguaggio, contando i file sorgente per estensione e saltando dipendenze e build.
package languages

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Stat struct {
	Name    string  `json:"name"`
	Color   string  `json:"color"`
	Bytes   int64   `json:"bytes"`
	Percent float64 `json:"percent"`
}

type lang struct{ name, color string }

// colori presi da GitHub Linguist
var (
	golang     = lang{"Go", "#00ADD8"}
	typescript = lang{"TypeScript", "#3178c6"}
	javascript = lang{"JavaScript", "#f1e05a"}
	java       = lang{"Java", "#b07219"}
	kotlin     = lang{"Kotlin", "#A97BFF"}
	cpp        = lang{"C++", "#f34b7d"}
)

var byExt = map[string]lang{
	".go": golang,
	".ts": typescript, ".tsx": typescript, ".mts": typescript, ".cts": typescript,
	".js": javascript, ".jsx": javascript, ".mjs": javascript, ".cjs": javascript,
	".svelte": {"Svelte", "#ff3e00"}, ".vue": {"Vue", "#41b883"}, ".astro": {"Astro", "#ff5a03"},
	".html": {"HTML", "#e34c26"}, ".htm": {"HTML", "#e34c26"},
	".css": {"CSS", "#663399"}, ".scss": {"SCSS", "#c6538c"}, ".sass": {"Sass", "#a53b70"}, ".less": {"Less", "#1d365d"},
	".java": java, ".kt": kotlin, ".kts": kotlin, ".groovy": {"Groovy", "#4298b8"}, ".gradle": {"Groovy", "#4298b8"},
	".scala": {"Scala", "#c22d40"},
	".rs":    {"Rust", "#dea584"},
	".py":    {"Python", "#3572A5"}, ".pyi": {"Python", "#3572A5"},
	".c": {"C", "#555555"}, ".h": {"C", "#555555"},
	".cpp": cpp, ".cc": cpp, ".cxx": cpp, ".hpp": cpp, ".hh": cpp, ".hxx": cpp,
	".cs": {"C#", "#178600"}, ".fs": {"F#", "#b845fc"}, ".vb": {"Visual Basic", "#945db7"},
	".php": {"PHP", "#4F5D95"}, ".rb": {"Ruby", "#701516"},
	".swift": {"Swift", "#F05138"}, ".m": {"Objective-C", "#438eff"}, ".mm": {"Objective-C++", "#6866fb"},
	".dart": {"Dart", "#00B4AB"},
	".ex":   {"Elixir", "#6e4a7e"}, ".exs": {"Elixir", "#6e4a7e"}, ".erl": {"Erlang", "#B83998"},
	".hs": {"Haskell", "#5e5086"}, ".lua": {"Lua", "#000080"}, ".r": {"R", "#198CE7"}, ".pl": {"Perl", "#0298c3"},
	".zig": {"Zig", "#ec915c"}, ".nim": {"Nim", "#ffc200"}, ".jl": {"Julia", "#a270ba"},
	".sh": {"Shell", "#89e051"}, ".bash": {"Shell", "#89e051"}, ".zsh": {"Shell", "#89e051"},
	".ps1": {"PowerShell", "#012456"}, ".bat": {"Batchfile", "#C1F12E"}, ".cmd": {"Batchfile", "#C1F12E"},
	".sql": {"SQL", "#e38c00"}, ".gd": {"GDScript", "#355570"}, ".shader": {"ShaderLab", "#222c37"},
	".tf": {"HCL", "#844FBA"}, ".nix": {"Nix", "#7e7eff"},
}

var byName = map[string]lang{
	"dockerfile": {"Dockerfile", "#384d54"},
	"makefile":   {"Makefile", "#427819"},
}

// cartelle di dipendenze, build e strumenti: non sono codice del progetto
var skipDirs = map[string]bool{
	".git": true, ".hg": true, ".svn": true, "node_modules": true, "vendor": true, "dist": true, "build": true,
	"out": true, "target": true, "bin": true, "obj": true, ".idea": true, ".vscode": true, ".gradle": true,
	".venv": true, "venv": true, "__pycache__": true, ".next": true, ".nuxt": true, ".svelte-kit": true,
	"coverage": true, "Pods": true, "DerivedData": true, ".dart_tool": true, ".cache": true, ".task": true,
	"bindings": true, "Library": true, "Temp": true,
}

const (
	maxFiles = 30000
	budget   = 3 * time.Second
	// sotto questa quota i linguaggi finiscono in "Other"
	minPercent = 1.0
)

// Analyze restituisce i linguaggi di dir, dal più presente; partial è true se la scansione si è fermata ai limiti.
func Analyze(dir string) (stats []Stat, partial bool) {
	totals := map[lang]int64{}
	files := 0
	deadline := time.Now().Add(budget)
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path != dir && (skipDirs[d.Name()] || strings.HasPrefix(d.Name(), ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		files++
		if files > maxFiles || time.Now().After(deadline) {
			partial = true
			return filepath.SkipAll
		}
		l, ok := classify(d.Name())
		if !ok {
			return nil
		}
		if info, err := d.Info(); err == nil {
			totals[l] += info.Size()
		}
		return nil
	})
	return summarize(totals), partial
}

func classify(name string) (lang, bool) {
	lower := strings.ToLower(name)
	if l, ok := byName[lower]; ok {
		return l, true
	}
	if strings.HasSuffix(lower, ".min.js") || strings.HasSuffix(lower, ".min.css") || strings.HasSuffix(lower, ".d.ts") {
		return lang{}, false // generati o dichiarazioni, non codice scritto
	}
	l, ok := byExt[filepath.Ext(lower)]
	return l, ok
}

func summarize(totals map[lang]int64) []Stat {
	var sum int64
	for _, b := range totals {
		sum += b
	}
	if sum == 0 {
		return nil
	}
	var out []Stat
	var other int64
	for l, b := range totals {
		p := float64(b) * 100 / float64(sum)
		if p < minPercent {
			other += b
			continue
		}
		out = append(out, Stat{Name: l.name, Color: l.color, Bytes: b, Percent: p})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Bytes > out[j].Bytes })
	if other > 0 {
		out = append(out, Stat{Name: "Other", Color: "#9a9aa0", Bytes: other, Percent: float64(other) * 100 / float64(sum)})
	}
	return out
}
