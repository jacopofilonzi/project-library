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

// ---------- GitHub and GitLab (gh and glab) ----------

// forges holds the two CLIs and the response caches.
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

// detect looks for the CLIs with the paths in the config and clears the caches.
func (f *forges) detect(cfg config.Config) {
	f.gh.Detect(cfg.GhPath)
	f.glab.Detect(cfg.GlabPath)
	f.mu.Lock()
	f.repos, f.infos = nil, map[string]infoEntry{}
	f.mu.Unlock()
}

// ForgeStatus is the state of a CLI, for the settings.
type ForgeStatus struct {
	Kind     forge.Kind      `json:"kind"`
	Path     string          `json:"path"`     // empty = not found
	Accounts []forge.Account `json:"accounts"` // empty = not logged in
	// Message: the last line of `auth status` when there is no account (e.g. "run gh auth login").
	Message string `json:"message"`
}

// Forges returns the state of gh and glab. With refresh it detects the CLIs again and re-reads the accounts.
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

// RepoList holds the repositories of all the accounts; Errors collects the accounts that did not answer.
type RepoList struct {
	Repos  []forge.Repo `json:"repos"`
	Errors []string     `json:"errors"`
}

// ForgeRepos lists the repositories of all the gh and glab accounts (cached for a few minutes).
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

// remoteOf gets host and owner/name from a git remote and finds the CLI with an account on that host.
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

// ForgeInfo returns PRs, issues and CI of the remote on the branch. nil if no CLI has an account
// on the remote's host: in that case the card shows nothing.
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

// ForgeOwners lists where the kind account on host can create a repository.
func (l *Library) ForgeOwners(kind, host string) ([]forge.Owner, error) {
	c := l.forge.cli(kind)
	acc, ok := c.Account(host)
	if !ok {
		return nil, &fsops.Error{Code: "forgeNoAccount", Detail: host}
	}
	// the user is always available even if the list of organizations or groups does not arrive
	owners, _ := c.Owners(acc)
	return owners, nil
}

// ForgeNameTaken tells whether owner/name already exists on the host (checked before creating the repository).
func (l *Library) ForgeNameTaken(kind, host, owner, name string) (bool, error) {
	c := l.forge.cli(kind)
	acc, ok := c.Account(host)
	if !ok {
		return false, &fsops.Error{Code: "forgeNoAccount", Detail: host}
	}
	taken, err := c.Exists(acc, owner, strings.TrimSpace(name))
	if err != nil {
		return false, &fsops.Error{Code: "forge", Detail: err.Error()}
	}
	return taken, nil
}

// PublishRequest describes the repository to create for a local project without a remote.
type PublishRequest struct {
	Path        string      `json:"path"`
	Kind        forge.Kind  `json:"kind"`
	Host        string      `json:"host"`
	Owner       forge.Owner `json:"owner"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Private     bool        `json:"private"`
}

// PublishResult: the created repository; Pushed is false if the project has no commits yet
// or the push failed (PushError holds git's message).
type PublishResult struct {
	Web       string `json:"web"`
	Pushed    bool   `json:"pushed"`
	PushError string `json:"pushError"`
}

// Publish creates the repository on GitHub or GitLab, adds it as the origin remote and pushes for the first time.
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
	// a repository without commits still on git's master (init outside the app or before the override):
	// the first push then goes to the configured branch, the one the forge suggests too
	if b := l.initialBranch(); info.Commit == nil && info.Branch == "master" && b != "" && b != "master" {
		if err := l.git.SetUnbornBranch(req.Path, b); err != nil {
			return res, &fsops.Error{Code: "git", Detail: err.Error()}
		}
	}
	if err := l.git.AddRemote(req.Path, "origin", created.CloneURL); err != nil {
		return res, &fsops.Error{Code: "addRemote", Detail: err.Error()}
	}
	l.forge.mu.Lock()
	l.forge.repos = nil // the new repository must show up in the clone list
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
