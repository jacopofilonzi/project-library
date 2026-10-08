// Package presets contiene il catalogo dei preset di riconoscimento dei progetti
// (Java/Gradle, Node, Android…) e la logica per verificare se un progetto corrisponde.
//
// Un preset è un elenco di pattern: basta che uno corrisponda. Un pattern senza "/"
// si confronta con i nomi nella cartella principale del progetto (glob, senza maiuscole);
// un pattern con "/" è un percorso relativo al progetto (glob per ogni segmento),
// es. "app/src/main/AndroidManifest.xml".
package presets

import (
	"os"
	"path/filepath"
	"strings"
)

type Preset struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Patterns []string `json:"patterns"`
	// Builtin: preset del catalogo dell'app (collegato: si aggiorna con l'app, non si modifica).
	Builtin bool `json:"builtin"`
}

// Catalog è il catalogo integrato. Gli id sono stabili: le regole li referenziano.
// Nessun preset è attivo di default: l'utente li associa a un editor dalle impostazioni.
func Catalog() []Preset {
	return []Preset{
		{ID: "android", Name: "Android", Patterns: []string{"app/src/main/AndroidManifest.xml", "src/main/AndroidManifest.xml", "AndroidManifest.xml"}},
		{ID: "gradle", Name: "Java / Kotlin (Gradle)", Patterns: []string{"build.gradle", "build.gradle.kts", "settings.gradle", "settings.gradle.kts", "gradlew"}},
		{ID: "maven", Name: "Java (Maven)", Patterns: []string{"pom.xml", "mvnw"}},
		{ID: "intellij", Name: "IntelliJ project", Patterns: []string{".idea", "*.iml"}},
		{ID: "node", Name: "Node.js", Patterns: []string{"package.json"}},
		{ID: "deno", Name: "Deno", Patterns: []string{"deno.json", "deno.jsonc"}},
		{ID: "bun", Name: "Bun", Patterns: []string{"bun.lockb", "bun.lock", "bunfig.toml"}},
		{ID: "go", Name: "Go", Patterns: []string{"go.mod", "go.work"}},
		{ID: "rust", Name: "Rust", Patterns: []string{"Cargo.toml"}},
		{ID: "python", Name: "Python", Patterns: []string{"pyproject.toml", "requirements.txt", "setup.py", "setup.cfg", "Pipfile"}},
		{ID: "php", Name: "PHP", Patterns: []string{"composer.json"}},
		{ID: "ruby", Name: "Ruby", Patterns: []string{"Gemfile", "*.gemspec"}},
		{ID: "dotnet", Name: ".NET", Patterns: []string{"*.sln", "*.csproj", "*.fsproj", "*.vbproj"}},
		{ID: "cpp", Name: "C / C++ (CMake, Meson)", Patterns: []string{"CMakeLists.txt", "meson.build"}},
		{ID: "flutter", Name: "Flutter / Dart", Patterns: []string{"pubspec.yaml"}},
		{ID: "swift", Name: "Swift / Xcode", Patterns: []string{"Package.swift", "*.xcodeproj", "*.xcworkspace"}},
		{ID: "unity", Name: "Unity", Patterns: []string{"ProjectSettings/ProjectVersion.txt"}},
		{ID: "godot", Name: "Godot", Patterns: []string{"project.godot"}},
		{ID: "elixir", Name: "Elixir", Patterns: []string{"mix.exs"}},
		{ID: "vscode", Name: "VS Code workspace", Patterns: []string{"*.code-workspace", ".vscode"}},
	}
}

// Find cerca un preset per id prima nel catalogo, poi tra quelli dell'utente.
func Find(id string, user []Preset) (Preset, bool) {
	for _, p := range Catalog() {
		if p.ID == id {
			p.Builtin = true
			return p, true
		}
	}
	for _, p := range user {
		if p.ID == id {
			return p, true
		}
	}
	return Preset{}, false
}

// Matches dice se il progetto in dir (con i nomi names nella cartella principale) corrisponde al preset.
func Matches(dir string, names []string, patterns []string) bool {
	for _, pat := range patterns {
		pat = strings.TrimSpace(pat)
		if pat == "" {
			continue
		}
		if strings.Contains(pat, "/") {
			if matchPath(dir, pat) {
				return true
			}
			continue
		}
		for _, n := range names {
			if glob(pat, n) {
				return true
			}
		}
	}
	return false
}

func glob(pattern, name string) bool {
	ok, err := filepath.Match(strings.ToLower(pattern), strings.ToLower(name))
	return err == nil && ok
}

// matchPath controlla un percorso relativo con glob nei segmenti (es. "*/src/main/AndroidManifest.xml").
func matchPath(dir, pattern string) bool {
	if !strings.ContainsAny(pattern, "*?[") {
		_, err := os.Stat(filepath.Join(dir, filepath.FromSlash(pattern)))
		return err == nil
	}
	matches, err := filepath.Glob(filepath.Join(dir, filepath.FromSlash(pattern)))
	return err == nil && len(matches) > 0
}
