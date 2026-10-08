// Package platform raccoglie tutto ciò che dipende dal sistema operativo.
// Ogni funzione esportata ha un'implementazione per Windows, macOS e Linux
// nei file platform_<os>.go; il resto dell'app usa solo queste funzioni.
//
// Funzioni per OS (definite in platform_<os>.go):
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
package platform

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

// InstallInfo descrive come installare git su questo sistema.
type InstallInfo struct {
	// Command è il comando da mostrare all'utente (vuoto se non ce n'è uno affidabile).
	Command string `json:"command"`
	// CanRun indica se l'app può lanciare l'installazione da sola.
	CanRun bool `json:"canRun"`
	// URL è la pagina di download, sempre presente.
	URL string `json:"url"`
}

// Problemi di un nome di file o cartella, restituiti da NameProblem.
// Sono codici: il frontend li traduce.
const (
	NameEmpty     = "empty"
	NameDots      = "dots"
	NameTooLong   = "tooLong"
	NameBadChars  = "badChars"
	NameReserved  = "reserved"
	NameTrailing  = "trailing"
	NameSeparator = "separator"
)

// NameProblem restituisce il codice del problema di name come nome di cartella, o "" se è valido.
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

// Editor è un editor noto, rilevato automaticamente su ogni sistema.
type Editor struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Editors è l'elenco degli editor noti. Quelli trovati vengono aggiunti ai launcher (disattivati).
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

// jetbrains: nome dello script di Toolbox, nome della cartella/app di installazione.
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

// EditorPath cerca l'eseguibile di un editor noto (vedi Editors).
// Restituisce "" se non lo trova.
func EditorPath(id string) string {
	for _, c := range editorCandidates(id) {
		if p := resolve(c); p != "" {
			return p
		}
	}
	return ""
}

// GitCandidates restituisce i percorsi dove cercare git, PATH compreso.
func GitCandidates() []string {
	return append([]string{"git"}, gitCandidates()...)
}

// resolve trasforma un candidato (nome nel PATH, percorso o glob) in un percorso esistente.
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
		for i := len(matches) - 1; i >= 0; i-- { // la versione più recente per ultima in ordine alfabetico
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

// CLI note oltre a git: le usa internal/forge. Gli id sono anche i nomi degli eseguibili.
const (
	CLIGitHub = "gh"
	CLIGitLab = "glab"
)

// cliURL è la pagina con le istruzioni di installazione di ogni CLI.
var cliURL = map[string]string{
	CLIGitHub: "https://cli.github.com",
	CLIGitLab: "https://gitlab.com/gitlab-org/cli#installation",
}

// CLICandidates restituisce dove cercare una CLI (gh, glab), PATH compreso.
func CLICandidates(id string) []string {
	return append([]string{id}, cliCandidates(id)...)
}

// Resolve è come resolve ma esportata, per i comandi dei launcher custom.
func Resolve(c string) string { return resolve(c) }

func home() string {
	h, _ := os.UserHomeDir()
	return h
}
