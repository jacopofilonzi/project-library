// Package presets holds the catalog of project recognition presets
// (Java/Gradle, Node, Android…) and the logic that checks whether a project matches.
//
// A preset is a list of patterns: one match is enough. A pattern without "/"
// is compared with the names in the project's top folder (glob, case-insensitive);
// a pattern with "/" is a path relative to the project (glob in every segment),
// e.g. "app/src/main/AndroidManifest.xml".
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
	// Builtin: preset from the app's catalog (linked: it updates with the app and cannot be edited).
	Builtin bool `json:"builtin"`
}

// Catalog is the built-in catalog. The ids are stable: the rules reference them.
// No preset is active by default: the user links them to an editor in the settings.
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

// Find looks for a preset by id first in the catalog, then among the user's.
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

// Matches tells whether the project in dir (with names in its top folder) matches the preset.
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

// matchPath checks a relative path with globs in its segments (e.g. "*/src/main/AndroidManifest.xml").
func matchPath(dir, pattern string) bool {
	if !strings.ContainsAny(pattern, "*?[") {
		_, err := os.Stat(filepath.Join(dir, filepath.FromSlash(pattern)))
		return err == nil
	}
	matches, err := filepath.Glob(filepath.Join(dir, filepath.FromSlash(pattern)))
	return err == nil && len(matches) > 0
}
