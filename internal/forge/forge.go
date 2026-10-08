// Package forge queries the GitHub (gh) and GitLab (glab) CLIs installed by the user,
// with the accounts they already logged in with: the app handles no tokens or credentials.
// Like gitinfo, it always uses the installed programs and never a library.
package forge

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/jacopofilonzi/project-library/internal/platform"
)

// Kind is the service: "github" (gh CLI) or "gitlab" (glab CLI).
type Kind string

const (
	GitHub Kind = "github"
	GitLab Kind = "gitlab"
)

// Bin is the name of the CLI executable.
func (k Kind) Bin() string {
	if k == GitHub {
		return platform.CLIGitHub
	}
	return platform.CLIGitLab
}

var ErrNoCLI = errors.New("cli not available")

// Account is a CLI login on a host (github.com, gitlab.com or a self-hosted instance).
type Account struct {
	Host string `json:"host"`
	User string `json:"user"`
	// Protocol is the protocol the CLI uses for git ("ssh" or "https"): it decides the clone URL.
	Protocol string `json:"protocol"`
}

// CLI is one of the two CLIs, with the path found and the accounts (cached).
type CLI struct {
	Kind Kind

	mu       sync.RWMutex
	path     string
	accounts []Account
	authErr  string
	checked  time.Time
}

// accountsTTL: how long the account list is valid (gh auth status goes over the network).
const accountsTTL = 5 * time.Minute

// Detect looks for the CLI: first the configured path, then the PATH and the usual system paths.
// It clears the account cache. It returns the path found (empty if missing).
func (c *CLI) Detect(configured string) string {
	candidates := platform.CLICandidates(c.Kind.Bin())
	if configured != "" {
		candidates = []string{configured}
	}
	found := ""
	for _, cand := range candidates {
		if p := platform.Resolve(cand); p != "" && works(p) {
			found = p
			break
		}
	}
	c.mu.Lock()
	c.path = found
	c.accounts, c.authErr, c.checked = nil, "", time.Time{}
	c.mu.Unlock()
	return found
}

func works(path string) bool {
	cmd := exec.Command(path, "--version")
	platform.HideConsole(cmd)
	out, err := cmd.Output()
	return err == nil && len(bytes.TrimSpace(out)) > 0
}

func (c *CLI) Path() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.path
}

// run runs the CLI without prompts or colors. On error the message is the CLI's stderr.
func (c *CLI) run(ctx context.Context, dir string, args ...string) ([]byte, error) {
	p := c.Path()
	if p == "" {
		return nil, ErrNoCLI
	}
	cmd := exec.CommandContext(ctx, p, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GH_PROMPT_DISABLED=1", "GH_NO_UPDATE_NOTIFIER=1", "GH_SPINNER_DISABLED=1",
		"NO_PROMPT=1", "GLAB_CHECK_UPDATE=false", "NO_COLOR=1", "CLICOLOR=0")
	platform.HideConsole(cmd)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return out, errors.New(msg)
	}
	return out, nil
}

// Accounts returns the logged-in accounts (cached for a few minutes, refresh bypasses the cache).
func (c *CLI) Accounts(refresh bool) ([]Account, string) {
	c.mu.RLock()
	fresh := !c.checked.IsZero() && time.Since(c.checked) < accountsTTL
	accs, msg := c.accounts, c.authErr
	c.mu.RUnlock()
	if fresh && !refresh {
		return accs, msg
	}
	if c.Path() == "" {
		return nil, ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	// auth status writes to stderr and exits with an error if a host is invalid: read everything anyway
	p := c.Path()
	cmd := exec.CommandContext(ctx, p, "auth", "status")
	cmd.Env = append(os.Environ(), "NO_COLOR=1", "CLICOLOR=0", "GH_NO_UPDATE_NOTIFIER=1", "GLAB_CHECK_UPDATE=false")
	platform.HideConsole(cmd)
	out, _ := cmd.CombinedOutput()
	accs = parseAuthStatus(string(out))
	msg = ""
	if len(accs) == 0 {
		msg = lastLine(string(out))
	}
	c.mu.Lock()
	c.accounts, c.authErr, c.checked = accs, msg, time.Now()
	c.mu.Unlock()
	return accs, msg
}

// Account returns the account for host, if any.
func (c *CLI) Account(host string) (Account, bool) {
	accs, _ := c.Accounts(false)
	for _, a := range accs {
		if strings.EqualFold(a.Host, host) {
			return a, true
		}
	}
	return Account{}, false
}

var (
	// gh: "Logged in to github.com account NAME (keyring)", older versions and glab: "Logged in to HOST as NAME (…)"
	reLogged = regexp.MustCompile(`Logged in to (\S+) (?:account|as) (\S+)`)
	reActive = regexp.MustCompile(`Active account: (true|false)`)
	// gh: "Git operations protocol: ssh"; glab: "Git operations for HOST configured to use ssh protocol."
	reProtoGH   = regexp.MustCompile(`Git operations protocol: (\w+)`)
	reProtoGLab = regexp.MustCompile(`Git operations for \S+ configured to use (\w+) protocol`)
)

// parseAuthStatus reads the output of `gh auth status` or `glab auth status`.
// With several accounts on the same host (gh) it keeps the active one.
func parseAuthStatus(out string) []Account {
	var accs []Account
	active := map[int]bool{}
	for _, line := range strings.Split(out, "\n") {
		if m := reLogged.FindStringSubmatch(line); m != nil {
			accs = append(accs, Account{Host: m[1], User: m[2], Protocol: "https"})
			continue
		}
		if len(accs) == 0 {
			continue
		}
		last := len(accs) - 1
		if m := reActive.FindStringSubmatch(line); m != nil {
			active[last] = m[1] == "true"
		} else if m := reProtoGH.FindStringSubmatch(line); m != nil {
			accs[last].Protocol = strings.ToLower(m[1])
		} else if m := reProtoGLab.FindStringSubmatch(line); m != nil {
			accs[last].Protocol = strings.ToLower(m[1])
		}
	}
	// one account per host: the active one, otherwise the first
	var out2 []Account
	seen := map[string]int{}
	for i, a := range accs {
		key := strings.ToLower(a.Host)
		if j, ok := seen[key]; ok {
			if active[i] {
				out2[j] = a
			}
			continue
		}
		seen[key] = len(out2)
		out2 = append(out2, a)
	}
	return out2
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}

// api calls the host's REST API through the CLI (which adds the token) and decodes the JSON into out.
func (c *CLI) api(ctx context.Context, host string, out any, args ...string) error {
	full := append([]string{"api", "--hostname", host}, args...)
	data, err := c.run(ctx, "", full...)
	if err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("unexpected response: %w", err)
	}
	return nil
}

func timeout(d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), d)
}
