// Package forge interroga le CLI di GitHub (gh) e GitLab (glab) installate dall'utente,
// con gli account con cui ha già fatto il login: l'app non gestisce token né credenziali.
// Come gitinfo, usa sempre i programmi installati e mai una libreria.
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

// Kind è il servizio: "github" (CLI gh) o "gitlab" (CLI glab).
type Kind string

const (
	GitHub Kind = "github"
	GitLab Kind = "gitlab"
)

// Bin è il nome dell'eseguibile della CLI.
func (k Kind) Bin() string {
	if k == GitHub {
		return platform.CLIGitHub
	}
	return platform.CLIGitLab
}

var ErrNoCLI = errors.New("cli not available")

// Account è un login della CLI su un host (github.com, gitlab.com o un'istanza propria).
type Account struct {
	Host string `json:"host"`
	User string `json:"user"`
	// Protocol è il protocollo che la CLI usa per git ("ssh" o "https"): decide l'URL di clone.
	Protocol string `json:"protocol"`
}

// CLI è una delle due CLI, con il percorso trovato e gli account (in cache).
type CLI struct {
	Kind Kind

	mu       sync.RWMutex
	path     string
	accounts []Account
	authErr  string
	checked  time.Time
}

// accountsTTL: per quanto tempo vale l'elenco degli account (gh auth status passa dalla rete).
const accountsTTL = 5 * time.Minute

// Detect cerca la CLI: prima il percorso configurato, poi PATH e i percorsi tipici del sistema.
// Azzera la cache degli account. Restituisce il percorso trovato (vuoto se non c'è).
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

// run esegue la CLI senza prompt né colori. In caso di errore il messaggio è lo stderr della CLI.
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

// Accounts restituisce gli account con il login fatto (in cache per qualche minuto, refresh la ignora).
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
	// auth status scrive su stderr e termina con errore se un host non è valido: si legge comunque tutto
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

// Account restituisce l'account per host, se c'è.
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
	// gh: "Logged in to github.com account NAME (keyring)", versioni vecchie e glab: "Logged in to HOST as NAME (…)"
	reLogged = regexp.MustCompile(`Logged in to (\S+) (?:account|as) (\S+)`)
	reActive = regexp.MustCompile(`Active account: (true|false)`)
	// gh: "Git operations protocol: ssh"; glab: "Git operations for HOST configured to use ssh protocol."
	reProtoGH   = regexp.MustCompile(`Git operations protocol: (\w+)`)
	reProtoGLab = regexp.MustCompile(`Git operations for \S+ configured to use (\w+) protocol`)
)

// parseAuthStatus legge l'output di `gh auth status` o `glab auth status`.
// Con più account sullo stesso host (gh) tiene quello attivo.
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
	// un account per host: quello attivo, altrimenti il primo
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

// api chiama l'API REST dell'host tramite la CLI (che aggiunge il token) e decodifica il JSON in out.
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
