package core

import (
	"encoding/json"
	"os"
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

// Presets restituisce il catalogo integrato (i preset dell'utente sono nella config).
func (l *Library) Presets() []presets.Preset {
	out := presets.Catalog()
	for i := range out {
		out[i].Builtin = true
	}
	return out
}

// ---------- editor noti ----------

// SyncEditors aggiunge ai launcher, disattivati, gli editor noti installati che non ci sono ancora.
// Restituisce i nomi di quelli aggiunti.
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

// ---------- linguaggi ----------

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

// Languages restituisce la composizione dei linguaggi del progetto (in cache per un minuto).
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

// ---------- export / import della configurazione ----------

// ExportConfig salva la configurazione in un file scelto dall'utente. Restituisce il percorso ("" se annullato).
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

// ImportConfig sostituisce la configurazione con quella di un file scelto dall'utente.
// I file di versioni precedenti passano dalle stesse migrazioni del caricamento.
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
