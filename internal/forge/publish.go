package forge

import (
	"regexp"
	"strconv"
	"time"
)

// Owner is where a repository can be created: the user, an organization (GitHub) or a group (GitLab).
type Owner struct {
	Name     string `json:"name"`
	Personal bool   `json:"personal"`
	// ID is the GitLab namespace (0 for the user and on GitHub).
	ID int `json:"id"`
}

// Owners lists the user and the organizations or groups where they can create repositories.
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
	// groups where the user is at least Maintainer (the level that can create projects by default)
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

// reNotFound recognizes the 404 answer in the error message of gh ("Not Found (HTTP 404)") and glab.
var reNotFound = regexp.MustCompile(`(?i)not found|\b404\b`)

// Exists tells whether owner/name already exists on the host. GitHub and GitLab are case-insensitive,
// as when the repository is created.
func (c *CLI) Exists(acc Account, owner, name string) (bool, error) {
	ctx, cancel := timeout(20 * time.Second)
	defer cancel()
	path := "repos/" + owner + "/" + name
	if c.Kind == GitLab {
		path = "projects/" + glPath(owner+"/"+name)
	}
	err := c.api(ctx, acc.Host, nil, path)
	if err == nil {
		return true, nil
	}
	if reNotFound.MatchString(err.Error()) {
		return false, nil
	}
	return false, err
}

// Created is the repository just created.
type Created struct {
	Web      string `json:"web"`
	CloneURL string `json:"cloneUrl"`
}

// Create creates an empty repository (no README or license, so the first push does not conflict).
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
