package forge

import (
	"fmt"
	"net/url"
	"time"
)

// Repo è un repository dell'utente (suo, di un'organizzazione o di un gruppo di cui fa parte).
type Repo struct {
	Kind        Kind   `json:"kind"`
	Host        string `json:"host"`
	FullName    string `json:"fullName"` // owner/nome, su GitLab anche gruppo/sottogruppo/nome
	Description string `json:"description"`
	Private     bool   `json:"private"`
	// CloneURL segue il protocollo scelto nella CLI (ssh o https).
	CloneURL string `json:"cloneUrl"`
	Web      string `json:"web"`
	Updated  string `json:"updated"` // RFC 3339
}

// maxPages limita l'elenco a 500 repository per account.
const maxPages = 5

// Repos elenca i repository dell'account, i più recenti prima.
func (c *CLI) Repos(acc Account) ([]Repo, error) {
	ctx, cancel := timeout(60 * time.Second)
	defer cancel()
	var out []Repo
	for page := 1; page <= maxPages; page++ {
		var batch []Repo
		var err error
		if c.Kind == GitHub {
			var raw []ghRepo
			err = c.api(ctx, acc.Host, &raw, fmt.Sprintf("user/repos?per_page=100&sort=pushed&affiliation=owner,collaborator,organization_member&page=%d", page))
			batch = ghRepos(raw, acc)
		} else {
			var raw []glProject
			err = c.api(ctx, acc.Host, &raw, fmt.Sprintf("projects?membership=true&simple=true&order_by=last_activity_at&per_page=100&page=%d", page))
			batch = glRepos(raw, acc)
		}
		if err != nil {
			return out, err
		}
		out = append(out, batch...)
		if len(batch) < 100 {
			break
		}
	}
	return out, nil
}

type ghRepo struct {
	FullName    string `json:"full_name"`
	Description string `json:"description"`
	Private     bool   `json:"private"`
	SSHURL      string `json:"ssh_url"`
	CloneURL    string `json:"clone_url"`
	HTMLURL     string `json:"html_url"`
	PushedAt    string `json:"pushed_at"`
}

func ghRepos(raw []ghRepo, acc Account) []Repo {
	out := make([]Repo, 0, len(raw))
	for _, r := range raw {
		clone := r.CloneURL
		if acc.Protocol == "ssh" && r.SSHURL != "" {
			clone = r.SSHURL
		}
		out = append(out, Repo{Kind: GitHub, Host: acc.Host, FullName: r.FullName, Description: r.Description,
			Private: r.Private, CloneURL: clone, Web: r.HTMLURL, Updated: r.PushedAt})
	}
	return out
}

type glProject struct {
	ID                int    `json:"id"`
	PathWithNamespace string `json:"path_with_namespace"`
	Description       string `json:"description"`
	Visibility        string `json:"visibility"`
	SSHURL            string `json:"ssh_url_to_repo"`
	HTTPURL           string `json:"http_url_to_repo"`
	WebURL            string `json:"web_url"`
	LastActivityAt    string `json:"last_activity_at"`
}

func glRepos(raw []glProject, acc Account) []Repo {
	out := make([]Repo, 0, len(raw))
	for _, p := range raw {
		clone := p.HTTPURL
		if acc.Protocol == "ssh" && p.SSHURL != "" {
			clone = p.SSHURL
		}
		out = append(out, Repo{Kind: GitLab, Host: acc.Host, FullName: p.PathWithNamespace, Description: p.Description,
			Private: p.Visibility != "public", CloneURL: clone, Web: p.WebURL, Updated: p.LastActivityAt})
	}
	return out
}

// glPath codifica gruppo/progetto per gli URL dell'API di GitLab (projects/gruppo%2Fprogetto).
func glPath(fullName string) string { return url.PathEscape(fullName) }
