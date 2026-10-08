// Package gitinfo queries the git installed by the user (never a library):
// same behavior as the terminal, same credentials and SSH keys.
package gitinfo

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jacopofilonzi/project-library/internal/platform"
)

var ErrNoGit = errors.New("git not available")

type Commit struct {
	Hash    string `json:"hash"`
	Subject string `json:"subject"`
	Author  string `json:"author"`
	Time    int64  `json:"time"` // unix seconds
}

type Info struct {
	IsRepo      bool    `json:"isRepo"`
	Branch      string  `json:"branch"`
	Detached    bool    `json:"detached"`
	Remote      string  `json:"remote"`    // URL as configured
	RemoteWeb   string  `json:"remoteWeb"` // https URL that opens in the browser, when it can be derived
	HasUpstream bool    `json:"hasUpstream"`
	Ahead       int     `json:"ahead"`
	Behind      int     `json:"behind"`
	Dirty       int     `json:"dirty"`
	Commit      *Commit `json:"commit"`
}

// Git resolves and runs the git binary.
type Git struct {
	mu   sync.RWMutex
	path string
}

// Detect looks for git: first the configured path, then the PATH and the usual system paths.
// It returns the path found (empty if missing).
func (g *Git) Detect(configured string) string {
	var found string
	candidates := platform.GitCandidates()
	if configured != "" {
		candidates = []string{configured}
	}
	for _, c := range candidates {
		p := platform.Resolve(c)
		if p != "" && works(p) {
			found = p
			break
		}
	}
	g.mu.Lock()
	g.path = found
	g.mu.Unlock()
	return found
}

func works(path string) bool {
	cmd := exec.Command(path, "--version")
	platform.HideConsole(cmd)
	out, err := cmd.Output()
	return err == nil && bytes.HasPrefix(out, []byte("git version"))
}

func (g *Git) Path() string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.path
}

func (g *Git) Available() bool { return g.Path() != "" }

func (g *Git) command(ctx context.Context, dir string, args ...string) (*exec.Cmd, error) {
	p := g.Path()
	if p == "" {
		return nil, ErrNoGit
	}
	cmd := exec.CommandContext(ctx, p, args...)
	cmd.Dir = dir
	// No interactive prompt in the terminal: there is no terminal.
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "LC_ALL=C")
	platform.HideConsole(cmd)
	return cmd, nil
}

func (g *Git) run(dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd, err := g.command(ctx, dir, args...)
	if err != nil {
		return "", err
	}
	out, err := cmd.Output()
	return string(out), err
}

// IsRepo: the project has its own .git folder (or file, for worktrees and submodules).
func IsRepo(dir string) bool {
	_, err := os.Lstat(filepath.Join(dir, ".git"))
	return err == nil
}

// Info collects the state of the repository in dir with three git commands.
func (g *Git) Info(dir string) (Info, error) {
	if !IsRepo(dir) {
		return Info{}, nil
	}
	if !g.Available() {
		return Info{IsRepo: true}, ErrNoGit
	}
	info := Info{IsRepo: true}

	// branch, upstream, ahead/behind and changed files in one go
	if out, err := g.run(dir, "status", "--porcelain=v2", "--branch"); err == nil {
		parseStatus(out, &info)
	}
	if out, err := g.run(dir, "log", "-1", "--format=%h%x00%s%x00%an%x00%ct"); err == nil {
		if parts := strings.Split(strings.TrimSpace(out), "\x00"); len(parts) == 4 {
			ts, _ := strconv.ParseInt(parts[3], 10, 64)
			info.Commit = &Commit{Hash: parts[0], Subject: parts[1], Author: parts[2], Time: ts}
		}
	}
	if out, err := g.run(dir, "remote"); err == nil {
		remotes := strings.Fields(out)
		name := ""
		for _, r := range remotes {
			if r == "origin" {
				name = r
			}
		}
		if name == "" && len(remotes) > 0 {
			name = remotes[0]
		}
		if name != "" {
			if url, err := g.run(dir, "remote", "get-url", name); err == nil {
				info.Remote = strings.TrimSpace(url)
				info.RemoteWeb = WebURL(info.Remote)
			}
		}
	}
	return info, nil
}

func parseStatus(out string, info *Info) {
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		l := sc.Text()
		switch {
		case strings.HasPrefix(l, "# branch.head "):
			h := strings.TrimPrefix(l, "# branch.head ")
			if h == "(detached)" {
				info.Detached = true
			} else {
				info.Branch = h
			}
		case strings.HasPrefix(l, "# branch.oid ") && info.Branch == "":
			if oid := strings.TrimPrefix(l, "# branch.oid "); len(oid) >= 7 && oid != "(initial)" {
				info.Branch = oid[:7]
			}
		case strings.HasPrefix(l, "# branch.upstream "):
			info.HasUpstream = true
		case strings.HasPrefix(l, "# branch.ab "):
			for _, f := range strings.Fields(strings.TrimPrefix(l, "# branch.ab ")) {
				n, _ := strconv.Atoi(f[1:])
				if f[0] == '+' {
					info.Ahead = n
				} else {
					info.Behind = n
				}
			}
		case strings.HasPrefix(l, "#") || l == "":
		default:
			info.Dirty++
		}
	}
}

// Dirty counts the changed or untracked files.
func (g *Git) Dirty(dir string) (int, error) {
	out, err := g.run(dir, "status", "--porcelain")
	if err != nil {
		return 0, err
	}
	n := 0
	for _, l := range strings.Split(out, "\n") {
		if strings.TrimSpace(l) != "" {
			n++
		}
	}
	return n, nil
}

// Fetch updates the remote refs (for ahead/behind). Quiet, without tags.
func (g *Git) Fetch(dir string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd, err := g.command(ctx, dir, "fetch", "--quiet", "--no-tags")
	if err != nil {
		return err
	}
	return cmd.Run()
}

// runOut runs git and returns its output; on error the message is git's output.
func (g *Git) runOut(dir string, timeout time.Duration, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd, err := g.command(ctx, dir, args...)
	if err != nil {
		return "", err
	}
	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil {
		if text == "" {
			text = err.Error()
		}
		return "", errors.New(text)
	}
	return text, nil
}

// Pull updates the current branch only if it can fast-forward (no implicit merge).
func (g *Git) Pull(dir string) (string, error) {
	return g.runOut(dir, 2*time.Minute, "pull", "--ff-only")
}

// FetchNow is like Fetch, but returns git's message on error.
func (g *Git) FetchNow(dir string) error {
	_, err := g.runOut(dir, 2*time.Minute, "fetch", "--no-tags")
	return err
}

// Init creates an empty repository in dir.
func (g *Git) Init(dir string) error {
	_, err := g.runOut(dir, 30*time.Second, "init")
	return err
}

// AddRemote adds the remote name with the given URL.
func (g *Git) AddRemote(dir, name, url string) error {
	_, err := g.runOut(dir, 30*time.Second, "remote", "add", name, url)
	return err
}

// PushUpstream pushes the current branch to the remote and sets it as upstream.
func (g *Git) PushUpstream(dir, remote string) error {
	_, err := g.runOut(dir, 5*time.Minute, "push", "--set-upstream", remote, "HEAD")
	return err
}

// Change is a changed, added, deleted or untracked file.
type Change struct {
	Status string `json:"status"` // two-letter git status code, e.g. "M", "??", "A", "D", "R"
	Path   string `json:"path"`
}

// Changes lists the changed files (at most limit).
func (g *Git) Changes(dir string, limit int) ([]Change, error) {
	out, err := g.run(dir, "status", "--porcelain=v1", "-z")
	if err != nil {
		return nil, err
	}
	return parseChanges(out, limit), nil
}

// parseChanges reads the output of `git status --porcelain=v1 -z`.
// With -z renames have the source path as a separate entry, to be skipped.
func parseChanges(out string, limit int) []Change {
	var res []Change
	parts := strings.Split(out, "\x00")
	for i := 0; i < len(parts); i++ {
		e := parts[i]
		if len(e) < 4 {
			continue
		}
		code := strings.TrimSpace(e[:2])
		res = append(res, Change{Status: code, Path: e[3:]})
		if e[0] == 'R' || e[0] == 'C' {
			i++ // source path of the rename
		}
		if len(res) >= limit {
			break
		}
	}
	return res
}

// Progress is a git clone update.
type Progress struct {
	Phase   string `json:"phase"`
	Percent int    `json:"percent"`
}

var reProgress = regexp.MustCompile(`^(?:remote: )?([A-Za-z ]+):\s+(\d+)%`)

// Clone runs git clone and reports the phases with their percentage.
// On error it returns git's last output lines.
func (g *Git) Clone(ctx context.Context, url, dest string, progress func(Progress)) error {
	cmd, err := g.command(ctx, filepath.Dir(dest), "clone", "--progress", url, dest)
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	var last []string
	readLines(stderr, func(line string) {
		line = strings.TrimSpace(line)
		if line == "" {
			return
		}
		if m := reProgress.FindStringSubmatch(line); m != nil {
			pct, _ := strconv.Atoi(m[2])
			progress(Progress{Phase: strings.TrimSpace(m[1]), Percent: pct})
			return
		}
		last = append(last, line)
		if len(last) > 6 {
			last = last[1:]
		}
	})
	if err := cmd.Wait(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if len(last) > 0 {
			return errors.New(strings.Join(last, "\n"))
		}
		return err
	}
	return nil
}

// readLines splits the output on \n and on \r (git updates the progress with \r).
func readLines(r io.Reader, fn func(string)) {
	br := bufio.NewReader(r)
	var buf strings.Builder
	for {
		b, err := br.ReadByte()
		if err != nil {
			if buf.Len() > 0 {
				fn(buf.String())
			}
			return
		}
		if b == '\n' || b == '\r' {
			fn(buf.String())
			buf.Reset()
			continue
		}
		buf.WriteByte(b)
	}
}

var (
	// scp form "user@host:path"; the host must contain a dot, so "C:\repo" is not mistaken for a remote
	reSCP = regexp.MustCompile(`^(?:[\w.-]+@)?([\w-]+\.[\w.-]+):([^\\]+?)(?:\.git)?/?$`)
	reURL = regexp.MustCompile(`^(?:https?|ssh|git)://(?:[^@/]+@)?([^/:]+)(?::\d+)?/(.+?)(?:\.git)?/?$`)
)

// WebURL turns a remote (https, ssh, scp-like) into the https address of the repository.
func WebURL(remote string) string {
	remote = strings.TrimSpace(remote)
	if m := reURL.FindStringSubmatch(remote); m != nil {
		return "https://" + m[1] + "/" + m[2]
	}
	if !strings.Contains(remote, "://") {
		if m := reSCP.FindStringSubmatch(remote); m != nil {
			return "https://" + m[1] + "/" + m[2]
		}
	}
	return ""
}

// RepoName gets owner and repository name from a clone URL (to suggest the destination).
func RepoName(url string) (host, owner, repo string, ok bool) {
	web := WebURL(url)
	if web == "" {
		return "", "", "", false
	}
	parts := strings.Split(strings.TrimPrefix(web, "https://"), "/")
	if len(parts) < 3 {
		return "", "", "", false
	}
	return parts[0], parts[len(parts)-2], parts[len(parts)-1], true
}
