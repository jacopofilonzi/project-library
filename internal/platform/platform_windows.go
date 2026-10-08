package platform

import (
	"errors"
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

// HideConsole keeps child processes (git, .cmd scripts) from opening a console window.
func HideConsole(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
}

// StartDetached starts a program that outlives the app.
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
	if jb, ok := jetbrains[id]; ok {
		script, dir := jb[0], jb[1]
		c := []string{`%LOCALAPPDATA%\JetBrains\Toolbox\scripts\` + script + `.cmd`}
		if id == "androidstudio" {
			c = append(c, `%ProgramFiles%\Android\Android Studio\bin\studio64.exe`, `%LOCALAPPDATA%\Programs\Android Studio\bin\studio64.exe`)
		} else {
			c = append(c, `%ProgramFiles%\JetBrains\`+dir+`*\bin\`+script+`64.exe`, `%LOCALAPPDATA%\Programs\`+dir+`*\bin\`+script+`64.exe`)
		}
		return append(c, script+"64", script)
	}
	switch id {
	case "vscode":
		return []string{
			`%LOCALAPPDATA%\Programs\Microsoft VS Code\Code.exe`,
			`%ProgramFiles%\Microsoft VS Code\Code.exe`,
			`%ProgramFiles(x86)%\Microsoft VS Code\Code.exe`,
			"code",
		}
	case "cursor":
		return []string{`%LOCALAPPDATA%\Programs\cursor\Cursor.exe`, `%ProgramFiles%\Cursor\Cursor.exe`, "cursor"}
	case "zed":
		return []string{`%LOCALAPPDATA%\Programs\Zed\Zed.exe`, `%ProgramFiles%\Zed\Zed.exe`, "zed"}
	case "sublime":
		return []string{`%ProgramFiles%\Sublime Text\sublime_text.exe`, `%ProgramFiles%\Sublime Text 3\sublime_text.exe`, "subl"}
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

// RunGitInstall opens a visible console with winget, so the user sees the progress and the UAC prompt.
func RunGitInstall() error {
	cmd := exec.Command("cmd.exe", "/c", "start", "Git", "cmd.exe", "/k", GitInstall().Command)
	return cmd.Start()
}

func cliCandidates(id string) []string {
	exe := id + ".exe"
	c := []string{`%LOCALAPPDATA%\Microsoft\WinGet\Links\` + exe, `%USERPROFILE%\scoop\shims\` + exe}
	switch id {
	case CLIGitHub:
		c = append(c, `%ProgramFiles%\GitHub CLI\gh.exe`, `%LOCALAPPDATA%\Programs\GitHub CLI\gh.exe`)
	case CLIGitLab:
		c = append(c, `%ProgramFiles%\glab\glab.exe`, `%LOCALAPPDATA%\Programs\glab\glab.exe`)
	}
	return c
}

var cliWinget = map[string]string{CLIGitHub: "GitHub.cli", CLIGitLab: "GLab.GLab"}

func CLIInstall(id string) InstallInfo {
	return InstallInfo{Command: "winget install --id " + cliWinget[id] + " -e --source winget", CanRun: true, URL: cliURL[id]}
}

// RunCLIInstall opens a visible console with winget, like RunGitInstall.
func RunCLIInstall(id string) error {
	return exec.Command("cmd.exe", "/c", "start", id, "cmd.exe", "/k", CLIInstall(id).Command).Start()
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

// ExpandEnv expands both %VAR% and $VAR and ~.
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

// CrossDevice tells whether a rename failed because source and destination are on different drives.
func CrossDevice(err error) bool { return errors.Is(err, syscall.Errno(17)) } // ERROR_NOT_SAME_DEVICE

// UpdateAsset is the end of the installer's name among the release files.
func UpdateAsset() string { return "-installer.exe" }
