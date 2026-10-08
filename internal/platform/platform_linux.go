package platform

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func StartDetached(command string, args []string, dir string) error {
	return startSetsid(command, args, dir)
}

func editorCandidates(id string) []string {
	h := home()
	switch id {
	case "vscode":
		return []string{
			"code", "/usr/bin/code", "/snap/bin/code",
			"/var/lib/flatpak/exports/bin/com.visualstudio.code",
			filepath.Join(h, ".local/share/flatpak/exports/bin/com.visualstudio.code"),
		}
	case "intellij":
		return []string{
			filepath.Join(h, ".local/share/JetBrains/Toolbox/scripts/idea"),
			"idea", "intellij-idea-ultimate", "intellij-idea-community",
			"/snap/bin/intellij-idea-ultimate", "/snap/bin/intellij-idea-community",
			"/opt/idea*/bin/idea.sh", "/opt/intellij*/bin/idea.sh",
		}
	case "androidstudio":
		return []string{
			filepath.Join(h, ".local/share/JetBrains/Toolbox/scripts/studio"),
			"android-studio", "/snap/bin/android-studio", "/opt/android-studio/bin/studio.sh",
			filepath.Join(h, "android-studio/bin/studio.sh"),
		}
	case "pycharm":
		return []string{
			filepath.Join(h, ".local/share/JetBrains/Toolbox/scripts/pycharm"),
			"pycharm", "pycharm-professional", "pycharm-community",
			"/snap/bin/pycharm-professional", "/snap/bin/pycharm-community", "/opt/pycharm*/bin/pycharm.sh",
		}
	case "cursor":
		return []string{"cursor", filepath.Join(h, "Applications/cursor.AppImage")}
	case "zed":
		return []string{"zed", "zeditor", filepath.Join(h, ".local/bin/zed")}
	case "sublime":
		return []string{"subl", "/opt/sublime_text/sublime_text"}
	}
	// the other JetBrains IDEs: Toolbox script, PATH, snap, install in /opt
	if jb, ok := jetbrains[id]; ok {
		script := jb[0]
		return []string{
			filepath.Join(h, ".local/share/JetBrains/Toolbox/scripts", script),
			script, "/snap/bin/" + script, "/opt/" + script + "*/bin/" + script + ".sh",
		}
	}
	return nil
}

func gitCandidates() []string {
	return []string{"/usr/bin/git", "/usr/local/bin/git"}
}

// GitInstall suggests the command of the distribution's package manager, read from /etc/os-release.
// The app does not run it: it would need sudo.
func GitInstall() InstallInfo {
	info := InstallInfo{URL: "https://git-scm.com/download/linux"}
	ids := osReleaseIDs()
	switch {
	case ids["debian"] || ids["ubuntu"]:
		info.Command = "sudo apt install git"
	case ids["fedora"] || ids["rhel"] || ids["centos"]:
		info.Command = "sudo dnf install git"
	case ids["arch"]:
		info.Command = "sudo pacman -S git"
	case ids["opensuse"] || ids["suse"] || ids["opensuse-leap"] || ids["opensuse-tumbleweed"]:
		info.Command = "sudo zypper install git"
	case ids["alpine"]:
		info.Command = "sudo apk add git"
	}
	return info
}

func RunGitInstall() error { return fmt.Errorf("not supported") }

func cliCandidates(id string) []string {
	return []string{"/usr/bin/" + id, "/usr/local/bin/" + id, "/snap/bin/" + id, filepath.Join(home(), ".local/bin", id), "/home/linuxbrew/.linuxbrew/bin/" + id}
}

// CLIInstall suggests the distribution's package, where there is one; otherwise just the install page.
func CLIInstall(id string) InstallInfo {
	info := InstallInfo{URL: cliURL[id]}
	ids := osReleaseIDs()
	switch {
	case ids["debian"] || ids["ubuntu"]:
		if id == CLIGitHub {
			info.Command = "sudo apt install gh"
		}
	case ids["fedora"]:
		info.Command = "sudo dnf install " + id
	case ids["arch"]:
		if id == CLIGitHub {
			info.Command = "sudo pacman -S github-cli"
		} else {
			info.Command = "sudo pacman -S glab"
		}
	}
	return info
}

func RunCLIInstall(id string) error { return fmt.Errorf("not supported") }

func osReleaseIDs() map[string]bool {
	ids := map[string]bool{}
	f, err := os.Open("/etc/os-release")
	if err != nil {
		return ids
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		k, v, ok := strings.Cut(sc.Text(), "=")
		if !ok || (k != "ID" && k != "ID_LIKE") {
			continue
		}
		for _, id := range strings.Fields(strings.Trim(v, `"'`)) {
			ids[id] = true
		}
	}
	return ids
}

func nameProblem(name string) string { return "" }

// MoveToTrash uses `gio trash`; if missing it implements the freedesktop trash specification.
func MoveToTrash(path string) error {
	if gio, err := exec.LookPath("gio"); err == nil {
		if err := exec.Command(gio, "trash", path).Run(); err == nil {
			return nil
		}
	}
	return freedesktopTrash(path)
}

func freedesktopTrash(path string) error {
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		dataHome = filepath.Join(home(), ".local/share")
	}
	files := filepath.Join(dataHome, "Trash", "files")
	infos := filepath.Join(dataHome, "Trash", "info")
	if err := os.MkdirAll(files, 0o700); err != nil {
		return err
	}
	if err := os.MkdirAll(infos, 0o700); err != nil {
		return err
	}
	base := filepath.Base(path)
	name := base
	for i := 2; ; i++ {
		if _, err := os.Lstat(filepath.Join(files, name)); os.IsNotExist(err) {
			break
		}
		name = base + "." + strconv.Itoa(i)
	}
	info := fmt.Sprintf("[Trash Info]\nPath=%s\nDeletionDate=%s\n",
		(&url.URL{Path: path}).EscapedPath(), time.Now().Format("2006-01-02T15:04:05"))
	infoPath := filepath.Join(infos, name+".trashinfo")
	if err := os.WriteFile(infoPath, []byte(info), 0o600); err != nil {
		return err
	}
	if err := os.Rename(path, filepath.Join(files, name)); err != nil {
		os.Remove(infoPath)
		return err
	}
	return nil
}

func CaseInsensitive() bool { return false }

// UpdateAsset: no Linux package is published yet, the update links to the release page.
func UpdateAsset() string { return "" }
