// Package platform collects everything that depends on the operating system.
// Every exported function has an implementation for Windows, macOS and Linux
// in the platform_<os>.go files; the rest of the app uses only these functions.
//
// Per-OS functions (defined in platform_<os>.go):
//
//	MoveToTrash(path string) error
//	StartDetached(command string, args []string, dir string) error
//	HideConsole(cmd *exec.Cmd)
//	editorCandidates(id string) []string
//	gitCandidates() []string
//	GitInstall() InstallInfo
//	RunGitInstall() error
//	cliCandidates(id string) []string
//	CLIInstall(id string) InstallInfo
//	RunCLIInstall(id string) error
//	nameProblem(name string) string
//	ExpandEnv(s string) string
//	CaseInsensitive() bool
//	UpdateAsset() string
//	CrossDevice(err error) bool
//	FileID(path string) string
package platform

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

// InstallInfo describes how to install git on this system.
type InstallInfo struct {
	// Command is the command to show the user (empty if there is no reliable one).
	Command string `json:"command"`
	// CanRun tells whether the app can start the installation by itself.
	CanRun bool `json:"canRun"`
	// URL is the download page, always present.
	URL string `json:"url"`
}

// Problems with a file or folder name, returned by NameProblem.
// They are codes: the frontend translates them.
const (
	NameEmpty     = "empty"
	NameDots      = "dots"
	NameTooLong   = "tooLong"
	NameBadChars  = "badChars"
	NameReserved  = "reserved"
	NameTrailing  = "trailing"
	NameSeparator = "separator"
)

// NameProblem returns the code of the problem with name as a folder name, or "" if it is valid.
func NameProblem(name string) string {
	if strings.TrimSpace(name) == "" {
		return NameEmpty
	}
	if name == "." || name == ".." {
		return NameDots
	}
	if len(name) > 255 || !utf8.ValidString(name) {
		return NameTooLong
	}
	if strings.ContainsAny(name, "/\x00") {
		return NameSeparator
	}
	return nameProblem(name)
}

// Editor is a known editor, detected automatically on every system.
type Editor struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Editors is the list of known editors. The ones found are added to the launchers (disabled).
var Editors = []Editor{
	{"vscode", "VS Code"},
	{"intellij", "IntelliJ IDEA"},
	{"androidstudio", "Android Studio"},
	{"webstorm", "WebStorm"},
	{"pycharm", "PyCharm"},
	{"goland", "GoLand"},
	{"rider", "Rider"},
	{"clion", "CLion"},
	{"phpstorm", "PhpStorm"},
	{"rubymine", "RubyMine"},
	{"rustrover", "RustRover"},
	{"cursor", "Cursor"},
	{"zed", "Zed"},
	{"sublime", "Sublime Text"},
}

// jetbrains: name of the Toolbox script, name of the install folder/app.
var jetbrains = map[string][2]string{
	"intellij":      {"idea", "IntelliJ IDEA"},
	"webstorm":      {"webstorm", "WebStorm"},
	"pycharm":       {"pycharm", "PyCharm"},
	"goland":        {"goland", "GoLand"},
	"rider":         {"rider", "JetBrains Rider"},
	"clion":         {"clion", "CLion"},
	"phpstorm":      {"phpstorm", "PhpStorm"},
	"rubymine":      {"rubymine", "RubyMine"},
	"rustrover":     {"rustrover", "RustRover"},
	"androidstudio": {"studio", "Android Studio"},
}

// EditorPath looks for the executable of a known editor (see Editors).
// It returns "" if it does not find it.
func EditorPath(id string) string {
	for _, c := range editorCandidates(id) {
		if p := resolve(c); p != "" {
			return p
		}
	}
	return ""
}

// GitCandidates returns the paths where to look for git, PATH included.
func GitCandidates() []string {
	return append([]string{"git"}, gitCandidates()...)
}

// resolve turns a candidate (name in the PATH, path or glob) into an existing path.
func resolve(c string) string {
	c = ExpandEnv(c)
	if !strings.ContainsAny(c, `/\`) {
		if p, err := exec.LookPath(c); err == nil {
			return p
		}
		return ""
	}
	if strings.ContainsAny(c, "*?[") {
		matches, _ := filepath.Glob(c)
		sort.Strings(matches)
		for i := len(matches) - 1; i >= 0; i-- { // the most recent version comes last in alphabetical order
			if _, err := os.Stat(matches[i]); err == nil {
				return matches[i]
			}
		}
		return ""
	}
	if _, err := os.Stat(c); err == nil {
		return c
	}
	return ""
}

// Known CLIs besides git: used by internal/forge. The ids are also the executable names.
const (
	CLIGitHub = "gh"
	CLIGitLab = "glab"
)

// cliURL is the page with the install instructions of each CLI.
var cliURL = map[string]string{
	CLIGitHub: "https://cli.github.com",
	CLIGitLab: "https://gitlab.com/gitlab-org/cli#installation",
}

// CLICandidates returns where to look for a CLI (gh, glab), PATH included.
func CLICandidates(id string) []string {
	return append([]string{id}, cliCandidates(id)...)
}

// Resolve is resolve, exported, for the commands of custom launchers.
func Resolve(c string) string { return resolve(c) }

func home() string {
	h, _ := os.UserHomeDir()
	return h
}
