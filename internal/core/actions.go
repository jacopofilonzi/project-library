package core

import (
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/jacopofilonzi/project-library/internal/config"
	"github.com/jacopofilonzi/project-library/internal/fsops"
	"github.com/jacopofilonzi/project-library/internal/gitinfo"
)

// ---------- ricorda dove eri ----------

// SetLastLocation salva la cartella corrente e il progetto selezionato, per riaprire l'app lì.
// Non passa da SaveConfig: non deve scatenare nuove scansioni né riapplicare le impostazioni di sistema.
func (l *Library) SetLastLocation(dir, selected string) error {
	_, err := l.store.Update(func(c *config.Config) {
		c.LastPath = dir
		c.LastSelected = selected
	})
	return err
}

// ---------- inizializza come progetto ----------

// InitProject trasforma una cartella (di solito vuota) in un progetto: git init e/o un README.md minimo.
func (l *Library) InitProject(path string, gitInit, readme bool) error {
	if !fsops.Within(path, l.store.Get().Roots) {
		return &fsops.Error{Code: "outsideRoots"}
	}
	if readme {
		f := filepath.Join(path, "README.md")
		if _, err := os.Stat(f); os.IsNotExist(err) {
			if err := os.WriteFile(f, []byte("# "+filepath.Base(path)+"\n"), 0o644); err != nil {
				return &fsops.Error{Code: "io", Detail: err.Error()}
			}
		}
	}
	if gitInit && !gitinfo.IsRepo(path) {
		if !l.git.Available() {
			return &fsops.Error{Code: "noGit"}
		}
		if err := l.git.Init(path); err != nil {
			return &fsops.Error{Code: "git", Detail: err.Error()}
		}
	}
	l.rescan(true)
	return nil
}

// ---------- azioni git rapide ----------

func (l *Library) GitFetch(path string) error {
	if err := l.git.FetchNow(path); err != nil {
		return &fsops.Error{Code: "git", Detail: err.Error()}
	}
	go l.computeDirty()
	return nil
}

// GitPull fa un pull solo fast-forward e restituisce il messaggio di git.
func (l *Library) GitPull(path string) (string, error) {
	out, err := l.git.Pull(path)
	if err != nil {
		return "", &fsops.Error{Code: "git", Detail: err.Error()}
	}
	go l.computeDirty()
	return out, nil
}

// GitChanges elenca i file modificati (massimo 200).
func (l *Library) GitChanges(path string) ([]gitinfo.Change, error) {
	return l.git.Changes(path, 200)
}

// ---------- finestre ----------

// ShowInMain porta la finestra principale sul percorso indicato (dalla ricerca flottante).
func (l *Library) ShowInMain(path string) {
	emitEvent(EventMainGoto, path)
	ShowMainWindow()
}

// RunInMain esegue nella finestra principale un comando della palette che ne ha bisogno.
func (l *Library) RunInMain(command string) {
	emitEvent(EventMainCommand, command)
	ShowMainWindow()
}

func (l *Library) OpenSpotlight() { ShowSpotlight() }

func (l *Library) HideSpotlight() {
	if SpotlightWindow != nil {
		SpotlightWindow.Hide()
	}
}

// Quit chiude l'app (anche se "resta nella tray" è attivo).
func (l *Library) Quit() {
	Quitting = true
	application.Get().Quit()
}

// Quitting: l'uscita è stata chiesta esplicitamente, la chiusura della finestra non deve finire nella tray.
var Quitting bool
