package platform

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
)

const (
	createNoWindow        = 0x08000000
	createNewProcessGroup = 0x00000200
)

// HideConsole evita che i processi figli (git, script .cmd) aprano una finestra console.
func HideConsole(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
}

// StartDetached avvia un programma che sopravvive alla chiusura dell'app.
func StartDetached(command string, args []string, dir string) error {
	cmd := exec.Command(command, args...)
	cmd.Dir = dir
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow | createNewProcessGroup}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

func editorCandidates(id string) []string {
	switch id {
	case "vscode":
		return []string{
			`%LOCALAPPDATA%\Programs\Microsoft VS Code\Code.exe`,
			`%ProgramFiles%\Microsoft VS Code\Code.exe`,
			`%ProgramFiles(x86)%\Microsoft VS Code\Code.exe`,
			"code",
		}
	case "intellij":
		return []string{
			`%LOCALAPPDATA%\JetBrains\Toolbox\scripts\idea.cmd`,
			`%ProgramFiles%\JetBrains\IntelliJ IDEA*\bin\idea64.exe`,
			`%LOCALAPPDATA%\Programs\IntelliJ IDEA*\bin\idea64.exe`,
			"idea64", "idea",
		}
	}
	return nil
}

func gitCandidates() []string {
	return []string{
		`%ProgramFiles%\Git\cmd\git.exe`,
		`%LOCALAPPDATA%\Programs\Git\cmd\git.exe`,
		`%ProgramFiles(x86)%\Git\cmd\git.exe`,
	}
}

func GitInstall() InstallInfo {
	return InstallInfo{Command: "winget install --id Git.Git -e --source winget", CanRun: true, URL: "https://git-scm.com/download/win"}
}

// RunGitInstall apre una console visibile con winget, così l'utente vede l'avanzamento e il prompt UAC.
func RunGitInstall() error {
	cmd := exec.Command("cmd.exe", "/c", "start", "Git", "cmd.exe", "/k", GitInstall().Command)
	return cmd.Start()
}

var reservedName = regexp.MustCompile(`(?i)^(con|prn|aux|nul|com[1-9]|lpt[1-9])(\..*)?$`)

func nameProblem(name string) string {
	for _, r := range name {
		if r < 32 || strings.ContainsRune(`<>:"\|?*`, r) {
			return NameBadChars
		}
	}
	if reservedName.MatchString(name) {
		return NameReserved
	}
	if strings.HasSuffix(name, ".") || strings.HasSuffix(name, " ") {
		return NameTrailing
	}
	return ""
}

var winEnv = regexp.MustCompile(`%([^%]+)%`)

// ExpandEnv espande sia %VAR% sia $VAR e ~.
func ExpandEnv(s string) string {
	s = winEnv.ReplaceAllStringFunc(s, func(m string) string {
		if v, ok := os.LookupEnv(m[1 : len(m)-1]); ok {
			return v
		}
		return m
	})
	if strings.HasPrefix(s, "~") {
		s = filepath.Join(home(), s[1:])
	}
	return os.ExpandEnv(s)
}

func CaseInsensitive() bool { return true }
