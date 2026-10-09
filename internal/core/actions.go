package core

import (
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/jacopofilonzi/project-library/internal/config"
	"github.com/jacopofilonzi/project-library/internal/fsops"
	"github.com/jacopofilonzi/project-library/internal/gitinfo"
)

// ---------- remember where you were ----------

// SetLastLocation saves the current folder and the selected project, to reopen the app there.
// It does not go through SaveConfig: it must not trigger new scans or re-apply the system settings.
func (l *Library) SetLastLocation(dir, selected string) error {
	_, err := l.store.Update(func(c *config.Config) {
		c.LastPath = dir
		c.LastSelected = selected
	})
	// the other windows need the updated config, or one of their saves would roll it back
	emitEvent(EventConfig, l.store.Get())
	return err
}

// ---------- initialize as a project ----------

// InitProject turns a folder (usually empty) into a project: git init and/or a minimal README.md.
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
		if err := l.git.Init(path, l.initialBranch()); err != nil {
			return &fsops.Error{Code: "git", Detail: err.Error()}
		}
	}
	l.rescan(true)
	return nil
}

// initialBranch is the branch new repositories start on: empty = git's init.defaultBranch.
func (l *Library) initialBranch() string {
	if c := l.store.Get(); c.GitBranchOverride {
		return c.GitBranch
	}
	return ""
}

// ---------- quick git actions ----------

func (l *Library) GitFetch(path string) error {
	if err := l.git.FetchNow(path); err != nil {
		return &fsops.Error{Code: "git", Detail: err.Error()}
	}
	go l.computeDirty()
	return nil
}

// GitPull runs a fast-forward-only pull and returns git's message.
func (l *Library) GitPull(path string) (string, error) {
	out, err := l.git.Pull(path)
	if err != nil {
		return "", &fsops.Error{Code: "git", Detail: err.Error()}
	}
	go l.computeDirty()
	return out, nil
}

// GitChanges lists the changed files (at most 200).
func (l *Library) GitChanges(path string) ([]gitinfo.Change, error) {
	return l.git.Changes(path, 200)
}

// ---------- windows ----------

// ShowInMain takes the main window to the given path (from the floating search).
func (l *Library) ShowInMain(path string) {
	emitEvent(EventMainGoto, path)
	ShowMainWindow()
}

// RunInMain runs in the main window a palette command that needs it.
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

// Quit closes the app (even when "keep in the tray" is on).
func (l *Library) Quit() {
	Quitting = true
	application.Get().Quit()
}

// Quitting: exit was requested explicitly, closing the window must not send the app to the tray.
var Quitting bool
