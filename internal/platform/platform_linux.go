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
	}
	return nil
}

func gitCandidates() []string {
	return []string{"/usr/bin/git", "/usr/local/bin/git"}
}

// GitInstall propone il comando del gestore pacchetti della distribuzione, letto da /etc/os-release.
// L'app non lo esegue: servirebbe sudo.
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

// MoveToTrash usa `gio trash`; se non c'è implementa la specifica freedesktop del Cestino.
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
