// Package core è il servizio esposto al frontend (binding Wails): tiene la
// configurazione, l'albero dei progetti e coordina git, launcher e file system.
package core

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/jacopofilonzi/project-library/internal/config"
	"github.com/jacopofilonzi/project-library/internal/fsops"
	"github.com/jacopofilonzi/project-library/internal/gitinfo"
	"github.com/jacopofilonzi/project-library/internal/launcher"
	"github.com/jacopofilonzi/project-library/internal/platform"
	"github.com/jacopofilonzi/project-library/internal/readme"
	"github.com/jacopofilonzi/project-library/internal/scanner"
	"github.com/jacopofilonzi/project-library/internal/watcher"
)

// Version è impostata in fase di build (-ldflags "-X …core.Version=…").
var Version = "0.1.0"

// Eventi emessi verso il frontend.
const (
	EventTree          = "tree:updated"
	EventDirty         = "git:dirty"
	EventCloneProgress = "clone:progress"
	EventConfig        = "config:updated"
)

// MainWindow è la finestra principale, impostata da main prima di Run.
var MainWindow application.Window

type Library struct {
	store *config.Store
	git   gitinfo.Git

	mu    sync.RWMutex
	tree  *scanner.Node
	dirty map[string]int

	watch     *watcher.Watcher
	fetchStop context.CancelFunc
	clones    sync.Map // id → context.CancelFunc
	hotkey    string
}

func New(store *config.Store) *Library {
	return &Library{store: store, dirty: map[string]int{}}
}

// ServiceStartup viene chiamata da Wails all'avvio.
func (l *Library) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	cfg := l.store.Get()
	l.git.Detect(cfg.GitPath)
	if w, err := watcher.New(400*time.Millisecond, func() { l.rescan(true) }); err == nil {
		l.watch = w
	} else {
		slog.Warn("watcher unavailable", "err", err)
	}
	l.rescan(false)
	l.applySystem(cfg, config.Config{})
	return nil
}

func (l *Library) ServiceShutdown() error {
	if l.fetchStop != nil {
		l.fetchStop()
	}
	if l.watch != nil {
		return l.watch.Close()
	}
	return nil
}

// AppState è tutto ciò che il frontend deve sapere all'avvio.
type AppState struct {
	Config       config.Config `json:"config"`
	OS           string        `json:"os"`
	Version      string        `json:"version"`
	GitPath      string        `json:"gitPath"`
	GitAvailable bool          `json:"gitAvailable"`
	ConfigPath   string        `json:"configPath"`
	Home         string        `json:"home"`
}

func (l *Library) State() AppState {
	home, _ := os.UserHomeDir()
	return AppState{
		Config:       l.store.Get(),
		OS:           runtime.GOOS,
		Version:      Version,
		GitPath:      l.git.Path(),
		GitAvailable: l.git.Available(),
		ConfigPath:   l.store.Path(),
		Home:         home,
	}
}

// SaveConfig salva la configurazione e applica gli effetti delle modifiche
// (nuova scansione, git, scorciatoia globale, avvio automatico, fetch).
func (l *Library) SaveConfig(next config.Config) (AppState, error) {
	prev := l.store.Get()
	if _, err := l.store.Update(func(c *config.Config) { *c = next }); err != nil {
		return l.State(), err
	}
	cfg := l.store.Get()
	if cfg.GitPath != prev.GitPath {
		l.git.Detect(cfg.GitPath)
	}
	if scanChanged(prev, cfg) {
		go l.rescan(true)
	}
	errs := l.applySystem(cfg, prev)
	return l.State(), errors.Join(errs...)
}

func scanChanged(a, b config.Config) bool {
	eq := func(x, y []string) bool { return strings.Join(x, "\x00") == strings.Join(y, "\x00") }
	if !eq(a.Roots, b.Roots) || !eq(a.Markers, b.Markers) || !eq(a.Ignore, b.Ignore) ||
		a.FilesAsProject != b.FilesAsProject || a.ShowEmpty != b.ShowEmpty || a.MaxDepth != b.MaxDepth ||
		a.FollowLinks != b.FollowLinks || len(a.Overrides) != len(b.Overrides) {
		return true
	}
	for k, v := range a.Overrides {
		if b.Overrides[k] != v {
			return true
		}
	}
	return false
}

// applySystem allinea scorciatoia globale, avvio automatico e fetch periodico alla config.
func (l *Library) applySystem(cfg, prev config.Config) []error {
	app := application.Get()
	var errs []error
	if app == nil {
		return nil
	}
	if cfg.Hotkey != l.hotkey {
		if l.hotkey != "" {
			_ = app.GlobalShortcut.Unregister(l.hotkey)
		}
		l.hotkey = ""
		if cfg.Hotkey != "" {
			if err := app.GlobalShortcut.Register(cfg.Hotkey, ShowMainWindow); err != nil {
				errs = append(errs, &fsops.Error{Code: "hotkey", Detail: err.Error()})
			} else {
				l.hotkey = cfg.Hotkey
			}
		}
	}
	if cfg.Autostart != prev.Autostart {
		var err error
		if cfg.Autostart {
			err = app.Autostart.Enable()
		} else {
			err = app.Autostart.Disable()
		}
		if err != nil {
			errs = append(errs, &fsops.Error{Code: "autostart", Detail: err.Error()})
		}
	}
	if cfg.GitFetch != prev.GitFetch || cfg.GitFetchMinutes != prev.GitFetchMinutes || prev.Version == 0 {
		l.restartFetch(cfg)
	}
	return errs
}

// ShowMainWindow mostra e porta in primo piano la finestra (tray, scorciatoia, seconda istanza).
func ShowMainWindow() {
	if MainWindow == nil {
		return
	}
	MainWindow.Show()
	MainWindow.UnMinimise()
	MainWindow.Focus()
}

// ---------- albero ----------

func (l *Library) Tree() *scanner.Node {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.tree
}

// Rescan rianalizza le radici e restituisce il nuovo albero.
func (l *Library) Rescan() *scanner.Node {
	l.rescan(false)
	return l.Tree()
}

func (l *Library) rescan(emit bool) {
	cfg := l.store.Get()
	tree := scanner.Scan(cfg.Roots, scanner.Options{
		Markers: cfg.Markers, Ignore: cfg.Ignore, FilesAsProject: cfg.FilesAsProject, ShowEmpty: cfg.ShowEmpty,
		FollowLinks: cfg.FollowLinks, MaxDepth: cfg.MaxDepth, Overrides: cfg.Overrides, CaseInsensitive: platform.CaseInsensitive(),
	})
	l.mu.Lock()
	l.tree = tree
	l.mu.Unlock()

	if l.watch != nil {
		var dirs []string
		scanner.Walk(tree, func(n *scanner.Node) {
			if n.Kind != scanner.KindProject && n.Path != "" && !n.Missing {
				dirs = append(dirs, n.Path)
			}
		})
		l.watch.Set(dirs)
	}
	if emit {
		emitEvent(EventTree, tree)
	}
	go l.computeDirty()
}

func emitEvent(name string, data any) {
	if app := application.Get(); app != nil {
		app.Event.Emit(name, data)
	}
}

// Dirty restituisce i file modificati per progetto (calcolati in background dopo ogni scansione).
func (l *Library) Dirty() map[string]int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make(map[string]int, len(l.dirty))
	for k, v := range l.dirty {
		out[k] = v
	}
	return out
}

// RefreshDirty ricalcola lo stato git di tutti i progetti (es. quando la finestra torna in primo piano).
func (l *Library) RefreshDirty() { go l.computeDirty() }

func (l *Library) computeDirty() {
	cfg := l.store.Get()
	if !cfg.GitInfo || !l.git.Available() {
		return
	}
	paths := l.gitProjects()
	result := make(map[string]int, len(paths))
	var mu sync.Mutex
	sem := make(chan struct{}, 6)
	var wg sync.WaitGroup
	for _, p := range paths {
		wg.Add(1)
		sem <- struct{}{}
		go func(p string) {
			defer wg.Done()
			defer func() { <-sem }()
			if n, err := l.git.Dirty(p); err == nil {
				mu.Lock()
				result[p] = n
				mu.Unlock()
			}
		}(p)
	}
	wg.Wait()
	l.mu.Lock()
	l.dirty = result
	l.mu.Unlock()
	emitEvent(EventDirty, result)
}

func (l *Library) gitProjects() []string {
	var paths []string
	scanner.Walk(l.Tree(), func(n *scanner.Node) {
		if n.Kind == scanner.KindProject && n.HasGit {
			paths = append(paths, n.Path)
		}
	})
	return paths
}

func (l *Library) restartFetch(cfg config.Config) {
	if l.fetchStop != nil {
		l.fetchStop()
		l.fetchStop = nil
	}
	if !cfg.GitFetch {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	l.fetchStop = cancel
	go func() {
		t := time.NewTicker(time.Duration(cfg.GitFetchMinutes) * time.Minute)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				for _, p := range l.gitProjects() {
					if ctx.Err() != nil {
						return
					}
					_ = l.git.Fetch(p)
				}
				l.computeDirty()
			}
		}
	}()
}

// ---------- scheda progetto ----------

type ReadmeResult struct {
	Found bool `json:"found"`
	readme.Readme
}

func (l *Library) Readme(path string) (ReadmeResult, error) {
	r, ok, err := readme.Read(path)
	return ReadmeResult{Found: ok, Readme: r}, err
}

// GitInfo restituisce lo stato git di un progetto. Senza git disponibile IsRepo resta vero ma il resto è vuoto.
func (l *Library) GitInfo(path string) (gitinfo.Info, error) {
	info, err := l.git.Info(path)
	if errors.Is(err, gitinfo.ErrNoGit) {
		return info, nil
	}
	if err == nil && info.IsRepo {
		l.mu.Lock()
		l.dirty[path] = info.Dirty
		l.mu.Unlock()
	}
	return info, err
}

// ---------- launcher ----------

// Open apre path con il launcher indicato e lo registra tra i recenti.
func (l *Library) Open(path, launcherID string) error {
	cfg := l.store.Get()
	for _, ln := range cfg.Launchers {
		if ln.ID == launcherID {
			if err := launcher.Launch(ln, launcher.NewTarget(path, cfg.Roots)); err != nil {
				return &fsops.Error{Code: "launch", Detail: err.Error()}
			}
			_, err := l.store.Update(func(c *config.Config) { c.AddRecent(path, launcherID, time.Now().UnixMilli()) })
			emitEvent(EventConfig, l.store.Get())
			return err
		}
	}
	return &fsops.Error{Code: "launcherNotFound"}
}

// TestLauncher prova un launcher non ancora salvato.
func (l *Library) TestLauncher(ln config.Launcher, path string) error {
	cfg := l.store.Get()
	if err := launcher.Launch(ln, launcher.NewTarget(path, cfg.Roots)); err != nil {
		return &fsops.Error{Code: "launch", Detail: err.Error()}
	}
	return nil
}

// LauncherStatus dice dove si trova l'eseguibile di ogni launcher (vuoto = non trovato).
func (l *Library) LauncherStatus() map[string]string {
	out := map[string]string{}
	for _, ln := range l.store.Get().Launchers {
		out[ln.ID] = launcher.Command(ln)
	}
	return out
}

// DetectEditor cerca un editor predefinito ("vscode", "intellij").
func (l *Library) DetectEditor(id string) string { return platform.EditorPath(id) }

// ResolveCommand verifica che un comando esista (nel PATH o come percorso).
func (l *Library) ResolveCommand(cmd string) string { return platform.Resolve(cmd) }

// Reveal mostra path nel file manager.
func (l *Library) Reveal(path string) error {
	return application.Get().Env.OpenFileManager(path, false)
}

func (l *Library) OpenURL(url string) error {
	if !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://") {
		return &fsops.Error{Code: "badUrl"}
	}
	return application.Get().Browser.OpenURL(url)
}

// ---------- file system ----------

func (l *Library) CheckName(parent, name string) string {
	var fe *fsops.Error
	if err := fsops.CheckName(parent, strings.TrimSpace(name)); errors.As(err, &fe) {
		return fe.Code
	}
	return ""
}

func (l *Library) Mkdir(parent, name string) (string, error) {
	p, err := fsops.Mkdir(parent, name)
	if err == nil {
		l.rescan(true)
	}
	return p, err
}

// Rename rinomina e aggiorna override e cronologia che puntano al vecchio percorso.
func (l *Library) Rename(path, newName string) (string, error) {
	if !fsops.Within(path, l.store.Get().Roots) {
		return "", &fsops.Error{Code: "outsideRoots"}
	}
	dest, err := fsops.Rename(path, newName)
	if err != nil {
		return "", err
	}
	l.store.Update(func(c *config.Config) { c.RenamePath(path, dest) })
	emitEvent(EventConfig, l.store.Get())
	l.rescan(true)
	return dest, nil
}

func (l *Library) IsEmptyDir(path string) bool { return fsops.IsEmptyDir(path) }

// Trash sposta path nel Cestino e ripulisce override e cronologia.
func (l *Library) Trash(path string) error {
	if err := fsops.Trash(path, l.store.Get().Roots); err != nil {
		return err
	}
	l.store.Update(func(c *config.Config) { c.RenamePath(path, "") })
	emitEvent(EventConfig, l.store.Get())
	l.rescan(true)
	return nil
}

// SetOverride segna path come "project" o "dir"; "" toglie l'eccezione.
func (l *Library) SetOverride(path, kind string) (AppState, error) {
	_, err := l.store.Update(func(c *config.Config) {
		if kind == "" {
			delete(c.Overrides, path)
		} else {
			c.Overrides[path] = kind
		}
	})
	l.rescan(true)
	return l.State(), err
}

// CreateRoot crea una cartella radice (wizard).
func (l *Library) CreateRoot(path string) error {
	return os.MkdirAll(filepath.Clean(path), 0o755)
}

func (l *Library) Exists(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}

// ---------- clone ----------

type CloneSuggestion struct {
	OK    bool   `json:"ok"`
	Host  string `json:"host"`
	Owner string `json:"owner"`
	Repo  string `json:"repo"`
}

func (l *Library) ParseRepoURL(url string) CloneSuggestion {
	h, o, r, ok := gitinfo.RepoName(url)
	return CloneSuggestion{OK: ok, Host: h, Owner: o, Repo: r}
}

type CloneProgress struct {
	ID      string `json:"id"`
	Phase   string `json:"phase"`
	Percent int    `json:"percent"`
}

// Clone esegue git clone url in parent/name, creando le cartelle mancanti.
// Blocca fino alla fine; l'avanzamento arriva con l'evento clone:progress.
func (l *Library) Clone(id, url, parent, name string) (string, error) {
	if !l.git.Available() {
		return "", &fsops.Error{Code: "noGit"}
	}
	if p := platform.NameProblem(name); p != "" {
		return "", &fsops.Error{Code: "name." + p}
	}
	cfg := l.store.Get()
	if !fsops.Within(filepath.Join(parent, name), cfg.Roots) {
		return "", &fsops.Error{Code: "outsideRoots"}
	}
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return "", &fsops.Error{Code: "io", Detail: err.Error()}
	}
	if err := fsops.CheckName(parent, name); err != nil {
		return "", err
	}
	ctx, cancel := context.WithCancel(context.Background())
	l.clones.Store(id, cancel)
	defer func() { cancel(); l.clones.Delete(id) }()

	dest := filepath.Join(parent, name)
	err := l.git.Clone(ctx, strings.TrimSpace(url), dest, func(p gitinfo.Progress) {
		emitEvent(EventCloneProgress, CloneProgress{ID: id, Phase: p.Phase, Percent: p.Percent})
	})
	if err != nil {
		if ctx.Err() != nil {
			os.RemoveAll(dest) // clone annullato: niente cartelle a metà
			return "", &fsops.Error{Code: "cancelled"}
		}
		return "", &fsops.Error{Code: "clone", Detail: err.Error()}
	}
	l.rescan(true)
	return dest, nil
}

func (l *Library) CancelClone(id string) {
	if c, ok := l.clones.Load(id); ok {
		c.(context.CancelFunc)()
	}
}

// ---------- git (impostazioni e wizard) ----------

// DetectGit cerca git (nel percorso indicato o automaticamente) e lo usa da subito.
func (l *Library) DetectGit(path string) string {
	found := l.git.Detect(path)
	if found != "" {
		go l.computeDirty()
	}
	return found
}

func (l *Library) GitInstallInfo() platform.InstallInfo { return platform.GitInstall() }

func (l *Library) RunGitInstall() error {
	if err := platform.RunGitInstall(); err != nil {
		return &fsops.Error{Code: "gitInstall", Detail: err.Error()}
	}
	return nil
}

// ---------- dialog di sistema ----------

func (l *Library) PickFolder(title, start string) (string, error) {
	d := application.Get().Dialog.OpenFile().CanChooseDirectories(true).CanChooseFiles(false).SetTitle(title)
	if start != "" {
		d.SetDirectory(start)
	}
	return d.PromptForSingleSelection()
}

func (l *Library) PickFile(title string) (string, error) {
	return application.Get().Dialog.OpenFile().CanChooseFiles(true).SetTitle(title).PromptForSingleSelection()
}

// RootsOf restituisce le radici configurate: main le usa per servire le immagini dei README.
// È una funzione e non un metodo per non esporla al frontend.
func RootsOf(l *Library) []string { return l.store.Get().Roots }
