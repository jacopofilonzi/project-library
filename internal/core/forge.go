package core

import (
	"strings"
	"sync"
	"time"

	"github.com/jacopofilonzi/project-library/internal/config"
	"github.com/jacopofilonzi/project-library/internal/forge"
	"github.com/jacopofilonzi/project-library/internal/fsops"
	"github.com/jacopofilonzi/project-library/internal/gitinfo"
	"github.com/jacopofilonzi/project-library/internal/platform"
)

// ---------- GitHub e GitLab (gh e glab) ----------

// forges contiene le due CLI e le cache delle risposte.
type forges struct {
	gh   forge.CLI
	glab forge.CLI

	mu      sync.Mutex
	repos   *RepoList
	reposAt time.Time
	infos   map[string]infoEntry
}

type infoEntry struct {
	info *forge.Info
	at   time.Time
}

const (
	reposTTL = 5 * time.Minute
	infoTTL  = time.Minute
)

func (f *forges) init() {
	f.gh.Kind, f.glab.Kind = forge.GitHub, forge.GitLab
	f.infos = map[string]infoEntry{}
}

func (f *forges) cli(kind string) *forge.CLI {
	if forge.Kind(kind) == forge.GitLab {
		return &f.glab
	}
	return &f.gh
}

// detect cerca le CLI con i percorsi della config e svuota le cache.
func (f *forges) detect(cfg config.Config) {
	f.gh.Detect(cfg.GhPath)
	f.glab.Detect(cfg.GlabPath)
	f.mu.Lock()
	f.repos, f.infos = nil, map[string]infoEntry{}
	f.mu.Unlock()
}

// ForgeStatus è lo stato di una CLI per le impostazioni.
type ForgeStatus struct {
	Kind     forge.Kind      `json:"kind"`
	Path     string          `json:"path"`     // vuoto = non trovata
	Accounts []forge.Account `json:"accounts"` // vuoto = login non fatto
	// Message: l'ultima riga di `auth status` quando non c'è nessun account (es. "run gh auth login").
	Message string `json:"message"`
}

// Forges restituisce lo stato di gh e glab. Con refresh rileva di nuovo le CLI e rilegge gli account.
func (l *Library) Forges(refresh bool) []ForgeStatus {
	if refresh {
		l.forge.detect(l.store.Get())
	}
	out := make([]ForgeStatus, 0, 2)
	for _, c := range []*forge.CLI{&l.forge.gh, &l.forge.glab} {
		accs, msg := c.Accounts(refresh)
		if accs == nil {
			accs = []forge.Account{}
		}
		out = append(out, ForgeStatus{Kind: c.Kind, Path: c.Path(), Accounts: accs, Message: msg})
	}
	return out
}

func (l *Library) ForgeInstallInfo(kind string) platform.InstallInfo {
	return platform.CLIInstall(forge.Kind(kind).Bin())
}

func (l *Library) RunForgeInstall(kind string) error {
	if err := platform.RunCLIInstall(forge.Kind(kind).Bin()); err != nil {
		return &fsops.Error{Code: "cliInstall", Detail: err.Error()}
	}
	return nil
}

// RepoList sono i repository di tutti gli account; Errors raccoglie gli account che non hanno risposto.
type RepoList struct {
	Repos  []forge.Repo `json:"repos"`
	Errors []string     `json:"errors"`
}

// ForgeRepos elenca i repository di tutti gli account di gh e glab (in cache per qualche minuto).
func (l *Library) ForgeRepos(refresh bool) RepoList {
	f := &l.forge
	f.mu.Lock()
	if !refresh && f.repos != nil && time.Since(f.reposAt) < reposTTL {
		r := *f.repos
		f.mu.Unlock()
		return r
	}
	f.mu.Unlock()

	res := RepoList{Repos: []forge.Repo{}, Errors: []string{}}
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, c := range []*forge.CLI{&f.gh, &f.glab} {
		accs, _ := c.Accounts(false)
		for _, acc := range accs {
			wg.Add(1)
			go func(c *forge.CLI, acc forge.Account) {
				defer wg.Done()
				repos, err := c.Repos(acc)
				mu.Lock()
				defer mu.Unlock()
				res.Repos = append(res.Repos, repos...)
				if err != nil {
					res.Errors = append(res.Errors, acc.Host+": "+err.Error())
				}
			}(c, acc)
		}
	}
	wg.Wait()
	f.mu.Lock()
	f.repos, f.reposAt = &res, time.Now()
	f.mu.Unlock()
	return res
}

// remoteOf ricava host e owner/nome da un remote git e trova la CLI con un account su quell'host.
func (l *Library) remoteOf(remote string) (c *forge.CLI, host, fullName string, ok bool) {
	web := gitinfo.WebURL(remote)
	if web == "" {
		return nil, "", "", false
	}
	host, fullName, _ = strings.Cut(strings.TrimPrefix(web, "https://"), "/")
	for _, c := range []*forge.CLI{&l.forge.gh, &l.forge.glab} {
		if _, found := c.Account(host); found {
			return c, host, fullName, true
		}
	}
	return nil, "", "", false
}

// ForgeInfo restituisce PR, issue e CI del remote sul branch. nil se nessuna CLI ha un account
// sull'host del remote: in quel caso la scheda non mostra niente.
func (l *Library) ForgeInfo(remote, branch string) (*forge.Info, error) {
	c, host, fullName, ok := l.remoteOf(remote)
	if !ok {
		return nil, nil
	}
	key := remote + "\x00" + branch
	f := &l.forge
	f.mu.Lock()
	if e, hit := f.infos[key]; hit && time.Since(e.at) < infoTTL {
		f.mu.Unlock()
		return e.info, nil
	}
	f.mu.Unlock()
	info, err := c.Info(host, fullName, branch)
	if err != nil {
		return nil, &fsops.Error{Code: "forge", Detail: err.Error()}
	}
	f.mu.Lock()
	f.infos[key] = infoEntry{&info, time.Now()}
	f.mu.Unlock()
	return &info, nil
}

// ForgeOwners elenca dove l'account di kind su host può creare un repository.
func (l *Library) ForgeOwners(kind, host string) ([]forge.Owner, error) {
	c := l.forge.cli(kind)
	acc, ok := c.Account(host)
	if !ok {
		return nil, &fsops.Error{Code: "forgeNoAccount", Detail: host}
	}
	// l'utente resta sempre disponibile anche se l'elenco di organizzazioni o gruppi non arriva
	owners, _ := c.Owners(acc)
	return owners, nil
}

// PublishRequest descrive il repository da creare per un progetto locale senza remote.
type PublishRequest struct {
	Path        string      `json:"path"`
	Kind        forge.Kind  `json:"kind"`
	Host        string      `json:"host"`
	Owner       forge.Owner `json:"owner"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Private     bool        `json:"private"`
}

// PublishResult: il repository creato; Pushed è falso se il progetto non ha ancora commit
// o se il push non è riuscito (PushError contiene il messaggio di git).
type PublishResult struct {
	Web       string `json:"web"`
	Pushed    bool   `json:"pushed"`
	PushError string `json:"pushError"`
}

// Publish crea il repository su GitHub o GitLab, lo aggiunge come remote origin e fa il primo push.
func (l *Library) Publish(req PublishRequest) (PublishResult, error) {
	if !l.git.Available() {
		return PublishResult{}, &fsops.Error{Code: "noGit"}
	}
	if !fsops.Within(req.Path, l.store.Get().Roots) {
		return PublishResult{}, &fsops.Error{Code: "outsideRoots"}
	}
	info, err := l.git.Info(req.Path)
	if err != nil || !info.IsRepo {
		return PublishResult{}, &fsops.Error{Code: "notRepo"}
	}
	if info.Remote != "" {
		return PublishResult{}, &fsops.Error{Code: "hasRemote", Detail: info.Remote}
	}
	c := l.forge.cli(string(req.Kind))
	acc, ok := c.Account(req.Host)
	if !ok {
		return PublishResult{}, &fsops.Error{Code: "forgeNoAccount", Detail: req.Host}
	}
	created, err := c.Create(acc, req.Owner, strings.TrimSpace(req.Name), strings.TrimSpace(req.Description), req.Private)
	if err != nil {
		return PublishResult{}, &fsops.Error{Code: "forgeCreate", Detail: err.Error()}
	}
	res := PublishResult{Web: created.Web}
	if err := l.git.AddRemote(req.Path, "origin", created.CloneURL); err != nil {
		return res, &fsops.Error{Code: "addRemote", Detail: err.Error()}
	}
	l.forge.mu.Lock()
	l.forge.repos = nil // il nuovo repository deve comparire nell'elenco del clone
	l.forge.mu.Unlock()
	if info.Commit != nil {
		if err := l.git.PushUpstream(req.Path, "origin"); err != nil {
			res.PushError = err.Error()
		} else {
			res.Pushed = true
		}
	}
	return res, nil
}
