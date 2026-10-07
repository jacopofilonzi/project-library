//go:build darwin || linux

package platform

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// HideConsole non serve fuori da Windows.
func HideConsole(cmd *exec.Cmd) {}

func startSetsid(command string, args []string, dir string) error {
	cmd := exec.Command(command, args...)
	cmd.Dir = dir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait() // raccoglie il processo quando termina, evitando zombie
	return nil
}

// ExpandEnv espande $VAR e ~.
func ExpandEnv(s string) string {
	if s == "~" || strings.HasPrefix(s, "~/") {
		s = filepath.Join(home(), s[1:])
	}
	return os.ExpandEnv(s)
}
