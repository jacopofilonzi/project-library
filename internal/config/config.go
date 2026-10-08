// Package config manages config.json: defaults, loading and saving.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/jacopofilonzi/project-library/internal/presets"
)

// Launcher is a program a project can be opened with (editor, terminal…).
// Args is a string with placeholders ({path}, {name}, {source}, {group});
// double quotes group an argument that contains spaces.
type Launcher struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Command string `json:"command"`
	Args    string `json:"args"`
	Enabled bool   `json:"enabled"`
	// Builtin is the id of a known editor ("vscode", "intellij", "androidstudio"…), empty for custom launchers.
	Builtin string `json:"builtin,omitempty"`
}

// Rule links a preset (from the catalog or the user's) to a launcher.
// Rules are ordered: the first one that matches and whose launcher is enabled wins.
type Rule struct {
	Preset   string `json:"preset"`
	Launcher string `json:"launcher"`
}

// LauncherRule is the rule format up to config v2 (plain patterns): read only by the migration.
type LauncherRule struct {
	Patterns []string `json:"patterns"`
	Launcher string   `json:"launcher"`
}

// Recent is an "Open with" event.
type Recent struct {
	Path     string `json:"path"`
	Launcher string `json:"launcher"`
	At       int64  `json:"at"` // unix ms
}

// Override forces how a folder is classified.
const (
	OverrideProject = "project"
	OverrideDir     = "dir"
)

type Config struct {
	Version  int    `json:"version"`
	Language string `json:"language"` // "en" | "it"
	Theme    string `json:"theme"`    // "system" | "light" | "dark"
	// UIScale is the size of the main window's interface (1 = 100%), between MinUIScale and MaxUIScale.
	// The frontend applies it as CSS zoom.
	UIScale float64 `json:"uiScale"`

	Roots           []string   `json:"roots"`
	Launchers       []Launcher `json:"launchers"`
	DefaultLauncher string     `json:"defaultLauncher"`
	// Presets: presets created by the user (the built-in ones are in the catalog, see internal/presets).
	Presets []presets.Preset `json:"presets"`
	// Rules: preset → launcher links, in priority order.
	Rules []Rule `json:"rules"`
	// LauncherRules: v2 format, converted to Presets + Rules by the migration.
	LauncherRules []LauncherRule `json:"launcherRules,omitempty"`
	// ProjectLaunchers: launcher chosen by hand for a project (path → id). Takes precedence over the rules.
	ProjectLaunchers map[string]string `json:"projectLaunchers"`
	// FolderIDs: file system identity (see platform.FileID) of the folders that have per-path settings
	// (ProjectLaunchers, Overrides), so they can be found again after a rename or move outside the app.
	FolderIDs map[string]string `json:"folderIds"`

	Markers        []string          `json:"markers"`
	Ignore         []string          `json:"ignore"`
	FilesAsProject bool              `json:"filesAsProject"`
	ShowEmpty      bool              `json:"showEmpty"`
	MaxDepth       int               `json:"maxDepth"`
	FollowLinks    bool              `json:"followLinks"`
	Overrides      map[string]string `json:"overrides"`

	GitPath         string `json:"gitPath"` // empty = automatic detection
	GitInfo         bool   `json:"gitInfo"`
	GitFetch        bool   `json:"gitFetch"`
	GitFetchMinutes int    `json:"gitFetchMinutes"`
	GitWarningShown bool   `json:"gitWarningShown"`
	// GhPath and GlabPath: paths of GitHub CLI and GitLab CLI (empty = automatic detection).
	GhPath   string `json:"ghPath"`
	GlabPath string `json:"glabPath"`

	Recent []Recent `json:"recent"`

	// Autostart is the old switch (config v1): read only by the migration to StartMode.
	Autostart bool `json:"autostart,omitempty"`
	// StartMode: start with the system. "off", "window" (opens the window) or "tray" (tray and floating search only).
	StartMode   string `json:"startMode"`
	CloseToTray bool   `json:"closeToTray"`
	// SpotlightHotkey opens the floating search. Empty = disabled.
	// (Configs before 1.6 also had "hotkey", a shortcut for the main window: it is ignored.)
	SpotlightHotkey string `json:"spotlightHotkey"`

	// CheckUpdates: look for a new release at startup. NotifiedVersion: the last version
	// a desktop notification was sent for (one notification per version).
	CheckUpdates    bool   `json:"checkUpdates"`
	NotifiedVersion string `json:"notifiedVersion"`

	// LastPath and LastSelected: where you were when the app closed (current folder and selected project).
	LastPath     string `json:"lastPath"`
	LastSelected string `json:"lastSelected"`

	SetupDone bool `json:"setupDone"`
}

const (
	currentVersion = 4
	MinUIScale     = 0.8
	MaxUIScale     = 1.5
	MaxRecent      = 20
)

// Start-with-the-system modes.
const (
	StartOff    = "off"
	StartWindow = "window"
	StartTray   = "tray"
)

var DefaultMarkers = []string{
	".git", ".hg", ".svn",
	"package.json", "deno.json", "go.mod", "Cargo.toml", "pom.xml", "build.gradle*", "settings.gradle*", "gradlew",
	"pyproject.toml", "requirements.txt", "setup.py", "composer.json", "Gemfile", "*.sln", "*.csproj",
	"CMakeLists.txt", "Makefile", "mix.exs", "pubspec.yaml",
	".idea", ".vscode", "mise.toml", ".tool-versions", ".gitignore", "README*",
}

var DefaultIgnore = []string{"node_modules", "vendor", "target", ".cache", ".venv", "__pycache__", "desktop.ini", "Thumbs.db", ".DS_Store"}

// Default returns the initial configuration. home is the user's home folder.
func Default(home string) Config {
	return Config{
		Version:  currentVersion,
		Language: "en",
		Theme:    "system",
		UIScale:  1,
		Roots:    []string{filepath.Join(home, "Development")},
		Launchers: []Launcher{
			{ID: "vscode", Name: "VS Code", Args: `"{path}"`, Enabled: true, Builtin: "vscode"},
			{ID: "intellij", Name: "IntelliJ IDEA", Args: `"{path}"`, Enabled: true, Builtin: "intellij"},
		},
		DefaultLauncher:  "vscode",
		Presets:          []presets.Preset{},
		Rules:            []Rule{},
		ProjectLaunchers: map[string]string{},
		FolderIDs:        map[string]string{},
		Markers:          append([]string(nil), DefaultMarkers...),
		Ignore:           append([]string(nil), DefaultIgnore...),
		FilesAsProject:   true,
		ShowEmpty:        true,
		MaxDepth:         20,
		Overrides:        map[string]string{},
		GitInfo:          true,
		GitFetchMinutes:  15,
		StartMode:        StartOff,
		CloseToTray:      false,
		SpotlightHotkey:  "Super+Ctrl+K",
		CheckUpdates:     true,
	}
}

// normalize repairs missing or out-of-range values (hand-written or old configs).
func (c *Config) normalize(home string) {
	def := Default(home)
	if c.Language != "en" && c.Language != "it" {
		c.Language = def.Language
	}
	if c.Theme != "light" && c.Theme != "dark" {
		c.Theme = "system"
	}
	if c.UIScale == 0 {
		c.UIScale = 1 // configs from before the setting
	}
	c.UIScale = math.Round(math.Min(math.Max(c.UIScale, MinUIScale), MaxUIScale)*100) / 100
	if c.Markers == nil {
		c.Markers = def.Markers
	}
	if c.Ignore == nil {
		c.Ignore = def.Ignore
	}
	if c.MaxDepth <= 0 || c.MaxDepth > 64 {
		c.MaxDepth = def.MaxDepth
	}
	if c.GitFetchMinutes < 5 {
		c.GitFetchMinutes = def.GitFetchMinutes
	}
	if c.Overrides == nil {
		c.Overrides = map[string]string{}
	}
	if c.Launchers == nil {
		c.Launchers = def.Launchers
	}
	// migration v2 → v3: pattern rules become presets + rules
	if c.Version < 3 {
		c.migrateLauncherRules()
	}
	c.LauncherRules = nil
	if c.Presets == nil {
		c.Presets = []presets.Preset{}
	}
	if c.Rules == nil {
		c.Rules = []Rule{}
	}
	if c.ProjectLaunchers == nil {
		c.ProjectLaunchers = map[string]string{}
	}
	if c.FolderIDs == nil {
		c.FolderIDs = map[string]string{}
	}
	if len(c.Recent) > MaxRecent {
		c.Recent = c.Recent[:MaxRecent]
	}
	// migration v1 → v2: the autostart switch becomes StartMode, the spotlight shortcut arrives
	if c.Version < 2 {
		if c.Autostart {
			c.StartMode = StartWindow
		}
		c.SpotlightHotkey = def.SpotlightHotkey
	}
	c.Autostart = false
	if c.StartMode != StartWindow && c.StartMode != StartTray {
		c.StartMode = StartOff
	}
	// migration v3 → v4: the update check arrives, on by default
	if c.Version < 4 {
		c.CheckUpdates = true
	}
	c.Version = currentVersion
}

// oldDefaultRule is the rule config v2 applied by default (Java → IntelliJ).
var oldDefaultRule = []string{"pom.xml", "build.gradle*", "settings.gradle*", "gradlew", "*.iml"}

// migrateLauncherRules converts the v2 rules. The old default rule becomes the equivalent
// built-in presets; the others become user presets.
func (c *Config) migrateLauncherRules() {
	for i, r := range c.LauncherRules {
		if strings.Join(r.Patterns, "\x00") == strings.Join(oldDefaultRule, "\x00") {
			for _, id := range []string{"gradle", "maven", "intellij"} {
				c.Rules = append(c.Rules, Rule{Preset: id, Launcher: r.Launcher})
			}
			continue
		}
		id := fmt.Sprintf("custom-%d", i+1)
		c.Presets = append(c.Presets, presets.Preset{ID: id, Name: fmt.Sprintf("Custom rule %d", i+1), Patterns: r.Patterns})
		c.Rules = append(c.Rules, Rule{Preset: id, Launcher: r.Launcher})
	}
}

// Store keeps the configuration in memory and saves it to disk on every change.
type Store struct {
	mu   sync.RWMutex
	path string
	home string
	cfg  Config
}

// Dir is the app's data folder (%APPDATA%, ~/Library/Application Support, ~/.config).
func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "project-library"), nil
}

// Load reads config.json, or starts from the defaults if it does not exist.
func Load(path, home string) (*Store, error) {
	s := &Store{path: path, home: home, cfg: Default(home)}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		// Unreadable config: keep it aside and start again from the defaults.
		_ = os.Rename(path, path+".broken")
		return s, nil
	}
	c.normalize(home)
	s.cfg = c
	return s, nil
}

func (s *Store) Path() string { return s.path }

// Get returns a copy of the configuration.
func (s *Store) Get() Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg.clone()
}

// Update applies fn to the configuration and saves it.
func (s *Store) Update(fn func(*Config)) (Config, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.cfg.clone()
	fn(&next)
	next.normalize(s.home)
	if err := s.write(next); err != nil {
		return s.cfg.clone(), err
	}
	s.cfg = next
	return next.clone(), nil
}

func (s *Store) write(c Config) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	// Atomic write: temporary file + rename.
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func (c Config) clone() Config {
	out := c
	out.Roots = append([]string(nil), c.Roots...)
	out.Launchers = append([]Launcher(nil), c.Launchers...)
	out.Markers = append([]string(nil), c.Markers...)
	out.Ignore = append([]string(nil), c.Ignore...)
	out.Recent = append([]Recent(nil), c.Recent...)
	out.Overrides = make(map[string]string, len(c.Overrides))
	for k, v := range c.Overrides {
		out.Overrides[k] = v
	}
	out.ProjectLaunchers = make(map[string]string, len(c.ProjectLaunchers))
	for k, v := range c.ProjectLaunchers {
		out.ProjectLaunchers[k] = v
	}
	out.FolderIDs = make(map[string]string, len(c.FolderIDs))
	for k, v := range c.FolderIDs {
		out.FolderIDs[k] = v
	}
	// empty lists stay non-nil: "[]" in JSON, not "null"
	out.Rules = append([]Rule{}, c.Rules...)
	out.Presets = make([]presets.Preset, len(c.Presets))
	for i, p := range c.Presets {
		p.Patterns = append([]string(nil), p.Patterns...)
		out.Presets[i] = p
	}
	out.LauncherRules = nil
	return out
}

// AddRecent puts path at the top of the history, without duplicates.
func (c *Config) AddRecent(path, launcher string, at int64) {
	out := []Recent{{Path: path, Launcher: launcher, At: at}}
	for _, r := range c.Recent {
		if r.Path != path {
			out = append(out, r)
		}
	}
	if len(out) > MaxRecent {
		out = out[:MaxRecent]
	}
	c.Recent = out
}

// RenamePath updates overrides, per-project launchers and history after oldPath
// (and everything inside it) is renamed, or deleted when newPath is empty.
func (c *Config) RenamePath(oldPath, newPath string) {
	move := func(p string) (string, bool) {
		if p == oldPath {
			return newPath, true
		}
		if rel, ok := under(oldPath, p); ok {
			if newPath == "" {
				return "", true
			}
			return filepath.Join(newPath, rel), true
		}
		return p, false
	}
	moveKeys := func(m map[string]string) map[string]string {
		out := map[string]string{}
		for k, v := range m {
			if np, changed := move(k); changed {
				if np != "" {
					out[np] = v
				}
			} else {
				out[k] = v
			}
		}
		return out
	}
	c.Overrides = moveKeys(c.Overrides)
	c.ProjectLaunchers = moveKeys(c.ProjectLaunchers)
	c.FolderIDs = moveKeys(c.FolderIDs)
	if np, changed := move(c.LastPath); changed {
		c.LastPath = np
	}
	if np, changed := move(c.LastSelected); changed {
		c.LastSelected = np
	}
	var rec []Recent
	for _, r := range c.Recent {
		if np, changed := move(r.Path); changed {
			if np == "" {
				continue
			}
			r.Path = np
		}
		rec = append(rec, r)
	}
	c.Recent = rec
}

// under tells whether p is inside dir and returns the relative path.
func under(dir, p string) (string, bool) {
	rel, err := filepath.Rel(dir, p)
	if err != nil || rel == "." || rel == ".." || len(rel) >= 3 && rel[:3] == ".."+string(filepath.Separator) {
		return "", false
	}
	return rel, true
}

// TrackedPaths returns the paths that have per-path settings (per-project launcher or override).
func (c *Config) TrackedPaths() []string {
	seen := map[string]bool{}
	var out []string
	for _, m := range []map[string]string{c.ProjectLaunchers, c.Overrides} {
		for p := range m {
			if !seen[p] {
				seen[p] = true
				out = append(out, p)
			}
		}
	}
	return out
}

// Relink keeps the per-path settings attached to their folders when these are renamed or moved
// outside the app. It records the identity (idOf) of every tracked folder that exists; for the ones
// that no longer exist it looks among candidates (the folders found by the scan) for the same
// identity and moves the settings there. It returns true if the config changed.
func (c *Config) Relink(exists func(string) bool, idOf func(string) string, candidates []string) bool {
	changed := false
	var orphans []string
	for _, p := range c.TrackedPaths() {
		if exists(p) {
			if id := idOf(p); id != "" && c.FolderIDs[p] != id {
				c.FolderIDs[p] = id
				changed = true
			}
		} else if c.FolderIDs[p] != "" {
			orphans = append(orphans, p)
		}
	}
	if len(orphans) > 0 {
		want := map[string]bool{}
		for _, p := range orphans {
			want[c.FolderIDs[p]] = true
		}
		found := map[string]string{} // id → new path
		for _, p := range candidates {
			if len(found) == len(want) {
				break
			}
			if id := idOf(p); want[id] && found[id] == "" {
				found[id] = p
			}
		}
		// parents first: moving a parent's settings also moves the ones of the folders inside it
		sort.Slice(orphans, func(i, j int) bool { return len(orphans[i]) < len(orphans[j]) })
		for _, old := range orphans {
			id := c.FolderIDs[old]
			np := found[id]
			if id == "" || np == "" || np == old {
				continue // already moved with its parent, or not found
			}
			if _, taken := c.ProjectLaunchers[np]; taken {
				continue
			}
			if _, taken := c.Overrides[np]; taken {
				continue
			}
			c.RenamePath(old, np)
			changed = true
		}
	}
	// identities of paths that no longer have settings are not needed
	tracked := map[string]bool{}
	for _, p := range c.TrackedPaths() {
		tracked[p] = true
	}
	for p := range c.FolderIDs {
		if !tracked[p] {
			delete(c.FolderIDs, p)
			changed = true
		}
	}
	return changed
}
