package platform

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// StartDetached avvia un programma staccato dall'app. I bundle .app passano da `open`.
func StartDetached(command string, args []string, dir string) error {
	if strings.HasSuffix(strings.TrimRight(command, "/"), ".app") {
		openArgs := append([]string{"-na", command, "--args"}, args...)
		return startSetsid("open", openArgs, dir)
	}
	return startSetsid(command, args, dir)
}

func editorCandidates(id string) []string {
	h := home()
	app := func(name string) []string {
		return []string{"/Applications/" + name + ".app", filepath.Join(h, "Applications", name+".app")}
	}
	if jb, ok := jetbrains[id]; ok {
		script, name := jb[0], jb[1]
		c := []string{filepath.Join(h, "Library/Application Support/JetBrains/Toolbox/scripts", script)}
		return append(append(c, app(name+"*")...), script)
	}
	switch id {
	case "vscode":
		return []string{
			"/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code",
			filepath.Join(h, "Applications/Visual Studio Code.app/Contents/Resources/app/bin/code"),
			"code",
		}
	case "cursor":
		return append(app("Cursor"), "cursor")
	case "zed":
		return append(app("Zed"), "zed")
	case "sublime":
		return append(app("Sublime Text"), "subl")
	}
	return nil
}

func gitCandidates() []string {
	return []string{"/opt/homebrew/bin/git", "/usr/local/bin/git", "/usr/bin/git"}
}

func GitInstall() InstallInfo {
	return InstallInfo{Command: "xcode-select --install", CanRun: true, URL: "https://git-scm.com/download/mac"}
}

// RunGitInstall apre il dialog di sistema per installare i Command Line Tools (che includono git).
func RunGitInstall() error {
	return exec.Command("xcode-select", "--install").Start()
}

func nameProblem(name string) string {
	if strings.Contains(name, ":") {
		return NameBadChars
	}
	return ""
}

// MoveToTrash usa il Finder, così l'elemento si può ripristinare con "Rimetti a posto".
// Se il Finder non è disponibile (o l'automazione è negata) sposta in ~/.Trash.
func MoveToTrash(path string) error {
	esc := strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(path)
	script := fmt.Sprintf(`tell application "Finder" to delete POSIX file "%s"`, esc)
	if err := exec.Command("osascript", "-e", script).Run(); err == nil {
		return nil
	}
	trash := filepath.Join(home(), ".Trash")
	dest := filepath.Join(trash, filepath.Base(path))
	if _, err := os.Lstat(dest); err == nil {
		dest = fmt.Sprintf("%s %s", dest, time.Now().Format("15.04.05"))
	}
	return os.Rename(path, dest)
}

func CaseInsensitive() bool { return true }
