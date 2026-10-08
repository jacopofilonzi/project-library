package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/jacopofilonzi/project-library/internal/config"
	"github.com/jacopofilonzi/project-library/internal/fsops"
	"github.com/jacopofilonzi/project-library/internal/languages"
	"github.com/jacopofilonzi/project-library/internal/platform"
	"github.com/jacopofilonzi/project-library/internal/presets"
)

// ---------- preset ----------

// Presets returns the built-in catalog (the user's presets are in the config).
func (l *Library) Presets() []presets.Preset {
	out := presets.Catalog()
	for i := range out {
		out[i].Builtin = true
	}
	return out
}

// ---------- known editors ----------

// SyncEditors adds to the launchers, disabled, the installed known editors that are not there yet.
// It returns the names of the ones it added.
func (l *Library) SyncEditors() []string {
	cfg := l.store.Get()
	have := map[string]bool{}
	for _, ln := range cfg.Launchers {
		if ln.Builtin != "" {
			have[ln.Builtin] = true
		}
	}
	var add []config.Launcher
	var names []string
	for _, e := range platform.Editors {
		if have[e.ID] || platform.EditorPath(e.ID) == "" {
			continue
		}
		add = append(add, config.Launcher{ID: e.ID, Name: e.Name, Args: `"{path}"`, Enabled: false, Builtin: e.ID})
		names = append(names, e.Name)
	}
	if len(add) == 0 {
		return []string{}
	}
	l.store.Update(func(c *config.Config) { c.Launchers = append(c.Launchers, add...) })
	emitEvent(EventConfig, l.store.Get())
	return names
}

// ---------- languages ----------

type LanguageResult struct {
	Stats   []languages.Stat `json:"stats"`
	Partial bool             `json:"partial"`
}

var langCache = struct {
	sync.Mutex
	m map[string]langEntry
}{m: map[string]langEntry{}}

type langEntry struct {
	res LanguageResult
	at  time.Time
}

// Languages returns the language breakdown of the project (cached for a minute).
func (l *Library) Languages(path string) LanguageResult {
	langCache.Lock()
	e, ok := langCache.m[path]
	langCache.Unlock()
	if ok && time.Since(e.at) < time.Minute {
		return e.res
	}
	stats, partial := languages.Analyze(path)
	if stats == nil {
		stats = []languages.Stat{}
	}
	res := LanguageResult{Stats: stats, Partial: partial}
	langCache.Lock()
	langCache.m[path] = langEntry{res, time.Now()}
	langCache.Unlock()
	return res
}

// ---------- configuration export / import ----------

// ExportConfig saves the configuration to a file chosen by the user. It returns the path ("" if cancelled).
func (l *Library) ExportConfig(title string) (string, error) {
	path, err := application.Get().Dialog.SaveFile().
		SetMessage(title).
		SetFilename("project-library-config.json").
		AddFilter("JSON", "*.json").
		PromptForSingleSelection()
	if err != nil || path == "" {
		return "", err
	}
	if !strings.HasSuffix(strings.ToLower(path), ".json") {
		path += ".json"
	}
	data, err := json.MarshalIndent(l.store.Get(), "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", &fsops.Error{Code: "io", Detail: err.Error()}
	}
	return path, nil
}

// ImportConfig replaces the configuration with the one in a file chosen by the user.
// Files from older versions go through the same migrations as loading.
func (l *Library) ImportConfig(title string) (AppState, error) {
	path, err := application.Get().Dialog.OpenFile().
		CanChooseFiles(true).
		SetTitle(title).
		AddFilter("JSON", "*.json").
		PromptForSingleSelection()
	if err != nil || path == "" {
		return l.State(), err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return l.State(), &fsops.Error{Code: "io", Detail: err.Error()}
	}
	var next config.Config
	if err := json.Unmarshal(data, &next); err != nil || next.Version == 0 {
		return l.State(), &fsops.Error{Code: "badConfig"}
	}
	return l.SaveConfig(next)
}

// ResetConfig brings the configuration back to the initial values: the frontend then reopens the wizard.
// Only the language is kept, so the wizard speaks the user's language. The previous configuration
// is saved to config.backup.json, next to config.json, and can be imported again.
func (l *Library) ResetConfig() (AppState, error) {
	prev := l.store.Get()
	data, err := json.MarshalIndent(prev, "", "  ")
	if err == nil {
		err = os.WriteFile(l.backupPath(), data, 0o644)
	}
	if err != nil {
		return l.State(), &fsops.Error{Code: "io", Detail: err.Error()}
	}
	home, _ := os.UserHomeDir()
	next := config.Default(home)
	next.Language = prev.Language
	if _, err := l.SaveConfig(next); err != nil {
		return l.State(), err
	}
	l.SyncEditors() // the installed known editors come back to the list, disabled, as on the first run
	return l.State(), nil
}

// backupPath is the file where ResetConfig saves the previous configuration.
func (l *Library) backupPath() string {
	return filepath.Join(filepath.Dir(l.store.Path()), "config.backup.json")
}
