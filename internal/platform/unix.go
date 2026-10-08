//go:build darwin || linux

package platform

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// HideConsole is not needed outside Windows.
func HideConsole(cmd *exec.Cmd) {}

func startSetsid(command string, args []string, dir string) error {
	cmd := exec.Command(command, args...)
	cmd.Dir = dir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait() // reap the process when it ends, avoiding zombies
	return nil
}

// ExpandEnv expands $VAR and ~.
func ExpandEnv(s string) string {
	if s == "~" || strings.HasPrefix(s, "~/") {
		s = filepath.Join(home(), s[1:])
	}
	return os.ExpandEnv(s)
}
