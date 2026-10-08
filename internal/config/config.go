// Package config gestisce config.json: valori di default, caricamento e salvataggio.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/jacopofilonzi/project-library/internal/presets"
)

// Launcher è un programma con cui aprire un progetto (editor, terminale…).
// Args è una stringa con segnaposti ({path}, {name}, {source}, {group});
// le virgolette raggruppano un argomento con spazi.
type Launcher struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Command string `json:"command"`
	Args    string `json:"args"`
	Enabled bool   `json:"enabled"`
	// Builtin è l'id di un editor noto ("vscode", "intellij", "androidstudio"…), vuoto per i launcher custom.
	Builtin string `json:"builtin,omitempty"`
}

// Rule associa un preset (del catalogo o dell'utente) a un launcher.
// Le regole sono ordinate: vince la prima che corrisponde e il cui launcher è abilitato.
type Rule struct {
	Preset   string `json:"preset"`
	Launcher string `json:"launcher"`
}

// LauncherRule è il formato delle regole fino alla config v2 (pattern diretti): letto solo per la migrazione.
type LauncherRule struct {
	Patterns []string `json:"patterns"`
	Launcher string   `json:"launcher"`
}

// Recent è un'apertura con "Apri con".
type Recent struct {
	Path     string `json:"path"`
	Launcher string `json:"launcher"`
	At       int64  `json:"at"` // unix ms
}

// Override forza la classificazione di una cartella.
const (
	OverrideProject = "project"
	OverrideDir     = "dir"
)

type Config struct {
	Version  int    `json:"version"`
	Language string `json:"language"` // "en" | "it"
	Theme    string `json:"theme"`    // "system" | "light" | "dark"

	Roots           []string   `json:"roots"`
	Launchers       []Launcher `json:"launchers"`
	DefaultLauncher string     `json:"defaultLauncher"`
	// Presets: preset creati dall'utente (quelli integrati sono nel catalogo, vedi internal/presets).
	Presets []presets.Preset `json:"presets"`
	// Rules: associazioni preset → launcher, in ordine di priorità.
	Rules []Rule `json:"rules"`
	// LauncherRules: formato v2, convertito in Presets + Rules dalla migrazione.
	LauncherRules []LauncherRule `json:"launcherRules,omitempty"`
	// ProjectLaunchers: launcher scelto a mano per un progetto (percorso → id). Ha la precedenza sulle regole.
	ProjectLaunchers map[string]string `json:"projectLaunchers"`

	Markers        []string          `json:"markers"`
	Ignore         []string          `json:"ignore"`
	FilesAsProject bool              `json:"filesAsProject"`
	ShowEmpty      bool              `json:"showEmpty"`
	MaxDepth       int               `json:"maxDepth"`
	FollowLinks    bool              `json:"followLinks"`
	Overrides      map[string]string `json:"overrides"`

	GitPath         string `json:"gitPath"` // vuoto = rilevamento automatico
	GitInfo         bool   `json:"gitInfo"`
	GitFetch        bool   `json:"gitFetch"`
	GitFetchMinutes int    `json:"gitFetchMinutes"`
	GitWarningShown bool   `json:"gitWarningShown"`

	Recent []Recent `json:"recent"`

	// Autostart è il vecchio interruttore (config v1): letto solo per la migrazione a StartMode.
	Autostart bool `json:"autostart,omitempty"`
	// StartMode: avvio con il sistema. "off", "window" (apre la finestra) o "tray" (solo tray e ricerca flottante).
	StartMode   string `json:"startMode"`
	CloseToTray bool   `json:"closeToTray"`
	// Hotkey mostra la finestra principale; SpotlightHotkey apre la ricerca flottante. Vuote = disattivate.
	Hotkey          string `json:"hotkey"`
	SpotlightHotkey string `json:"spotlightHotkey"`

	// LastPath e LastSelected: dove eri alla chiusura (cartella corrente e progetto selezionato).
	LastPath     string `json:"lastPath"`
	LastSelected string `json:"lastSelected"`

	SetupDone bool `json:"setupDone"`
}

const (
	currentVersion = 3
	MaxRecent      = 20
)

// Modalità di avvio con il sistema.
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

// Default restituisce la configurazione iniziale. home è la cartella utente.
func Default(home string) Config {
	return Config{
		Version:  currentVersion,
		Language: "en",
		Theme:    "system",
		Roots:    []string{filepath.Join(home, "Development")},
		Launchers: []Launcher{
			{ID: "vscode", Name: "VS Code", Args: `"{path}"`, Enabled: true, Builtin: "vscode"},
			{ID: "intellij", Name: "IntelliJ IDEA", Args: `"{path}"`, Enabled: true, Builtin: "intellij"},
		},
		DefaultLauncher:  "vscode",
		Presets:          []presets.Preset{},
		Rules:            []Rule{},
		ProjectLaunchers: map[string]string{},
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
		Hotkey:           "CmdOrCtrl+Alt+Space",
		SpotlightHotkey:  "Super+Ctrl+K",
	}
}

// normalize ripara i valori mancanti o fuori range (config scritte a mano o vecchie).
func (c *Config) normalize(home string) {
	def := Default(home)
	if c.Language != "en" && c.Language != "it" {
		c.Language = def.Language
	}
	if c.Theme != "light" && c.Theme != "dark" {
		c.Theme = "system"
	}
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
	// migrazione v2 → v3: le regole a pattern diventano preset + regole
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
	if len(c.Recent) > MaxRecent {
		c.Recent = c.Recent[:MaxRecent]
	}
	// migrazione v1 → v2: l'interruttore autostart diventa StartMode, arriva la scorciatoia spotlight
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
	c.Version = currentVersion
}

// oldDefaultRule è la regola che la config v2 preapplicava (Java → IntelliJ).
var oldDefaultRule = []string{"pom.xml", "build.gradle*", "settings.gradle*", "gradlew", "*.iml"}

// migrateLauncherRules converte le regole v2. La vecchia regola di default diventa i preset
// integrati equivalenti; le altre diventano preset dell'utente.
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

// Store tiene la configurazione in memoria e la salva su disco a ogni modifica.
type Store struct {
	mu   sync.RWMutex
	path string
	home string
	cfg  Config
}

// Dir è la cartella dei dati dell'app (%APPDATA%, ~/Library/Application Support, ~/.config).
func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "project-library"), nil
}

// Load legge config.json, o parte dai default se non esiste.
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
		// Config illeggibile: la conserva a parte e riparte dai default.
		_ = os.Rename(path, path+".broken")
		return s, nil
	}
	c.normalize(home)
	s.cfg = c
	return s, nil
}

func (s *Store) Path() string { return s.path }

// Get restituisce una copia della configurazione.
func (s *Store) Get() Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg.clone()
}

// Update applica fn alla configurazione e la salva.
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
	// Scrittura atomica: file temporaneo + rename.
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
	// le liste vuote restano non-nil: in JSON "[]" e non "null"
	out.Rules = append([]Rule{}, c.Rules...)
	out.Presets = make([]presets.Preset, len(c.Presets))
	for i, p := range c.Presets {
		p.Patterns = append([]string(nil), p.Patterns...)
		out.Presets[i] = p
	}
	out.LauncherRules = nil
	return out
}

// AddRecent mette path in cima alla cronologia, senza duplicati.
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

// RenamePath aggiorna override, launcher per progetto e cronologia dopo il rename
// (o l'eliminazione, con newPath vuoto) di oldPath e di tutto ciò che contiene.
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

// under dice se p è dentro dir e restituisce il percorso relativo.
func under(dir, p string) (string, bool) {
	rel, err := filepath.Rel(dir, p)
	if err != nil || rel == "." || rel == ".." || len(rel) >= 3 && rel[:3] == ".."+string(filepath.Separator) {
		return "", false
	}
	return rel, true
}
