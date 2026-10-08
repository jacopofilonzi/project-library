package forge

import (
	"strconv"
	"time"
)

// Owner è dove si può creare un repository: l'utente, un'organizzazione (GitHub) o un gruppo (GitLab).
type Owner struct {
	Name     string `json:"name"`
	Personal bool   `json:"personal"`
	// ID è il namespace di GitLab (0 per l'utente e su GitHub).
	ID int `json:"id"`
}

// Owners elenca l'utente e le organizzazioni o i gruppi in cui può creare repository.
func (c *CLI) Owners(acc Account) ([]Owner, error) {
	ctx, cancel := timeout(30 * time.Second)
	defer cancel()
	out := []Owner{{Name: acc.User, Personal: true}}
	if c.Kind == GitHub {
		var orgs []struct {
			Login string `json:"login"`
		}
		if err := c.api(ctx, acc.Host, &orgs, "user/orgs?per_page=100"); err != nil {
			return out, err
		}
		for _, o := range orgs {
			out = append(out, Owner{Name: o.Login})
		}
		return out, nil
	}
	// gruppi in cui l'utente è almeno Maintainer (il livello che di default può creare progetti)
	var groups []struct {
		ID       int    `json:"id"`
		FullPath string `json:"full_path"`
	}
	if err := c.api(ctx, acc.Host, &groups, "groups?min_access_level=40&per_page=100&order_by=path"); err != nil {
		return out, err
	}
	for _, g := range groups {
		out = append(out, Owner{Name: g.FullPath, ID: g.ID})
	}
	return out, nil
}

// Created è il repository appena creato.
type Created struct {
	Web      string `json:"web"`
	CloneURL string `json:"cloneUrl"`
}

// Create crea un repository vuoto (senza README né licenza, così il primo push non va in conflitto).
func (c *CLI) Create(acc Account, owner Owner, name, description string, private bool) (Created, error) {
	ctx, cancel := timeout(60 * time.Second)
	defer cancel()
	if c.Kind == GitHub {
		endpoint := "user/repos"
		if !owner.Personal {
			endpoint = "orgs/" + owner.Name + "/repos"
		}
		var r ghRepo
		err := c.api(ctx, acc.Host, &r, "-X", "POST", endpoint,
			"-f", "name="+name, "-f", "description="+description, "-F", "private="+strconv.FormatBool(private))
		if err != nil {
			return Created{}, err
		}
		clone := r.CloneURL
		if acc.Protocol == "ssh" && r.SSHURL != "" {
			clone = r.SSHURL
		}
		return Created{Web: r.HTMLURL, CloneURL: clone}, nil
	}
	visibility := "public"
	if private {
		visibility = "private"
	}
	args := []string{"-X", "POST", "projects", "-f", "name=" + name, "-f", "path=" + name,
		"-f", "description=" + description, "-f", "visibility=" + visibility}
	if !owner.Personal && owner.ID != 0 {
		args = append(args, "-f", "namespace_id="+strconv.Itoa(owner.ID))
	}
	var p glProject
	if err := c.api(ctx, acc.Host, &p, args...); err != nil {
		return Created{}, err
	}
	clone := p.HTTPURL
	if acc.Protocol == "ssh" && p.SSHURL != "" {
		clone = p.SSHURL
	}
	return Created{Web: p.WebURL, CloneURL: clone}, nil
}
