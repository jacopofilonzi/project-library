package forge

import (
	"reflect"
	"testing"
)

func TestParseAuthStatusGitHub(t *testing.T) {
	out := `github.com
  ✓ Logged in to github.com account old-me (keyring)
  - Active account: false
  - Git operations protocol: https
  - Token: gho_************************************

  ✓ Logged in to github.com account jacopofilonzi (keyring)
  - Active account: true
  - Git operations protocol: ssh
  - Token: gho_************************************
  - Token scopes: 'gist', 'read:org', 'repo'

ghe.example.com
  ✓ Logged in to ghe.example.com as worker (oauth_token)
  ✓ Git operations for ghe.example.com configured to use https protocol.
`
	got := parseAuthStatus(out)
	want := []Account{
		{Host: "github.com", User: "jacopofilonzi", Protocol: "ssh"},
		{Host: "ghe.example.com", User: "worker", Protocol: "https"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v\nwant %+v", got, want)
	}
}

func TestParseAuthStatusGitLab(t *testing.T) {
	out := `gitlab.com
  ✓ Logged in to gitlab.com as jacopofilonzi (C:\Users\jacop\.config\glab-cli\config.yml)
  ✓ Git operations for gitlab.com configured to use ssh protocol.
  ✓ API calls for gitlab.com are made over https protocol.
  ✓ REST API Endpoint: https://gitlab.com/api/v4/
  ✓ Token found: **************************
git.example.org
  x git.example.org: API call failed: GET https://git.example.org/api/v4/user: 401 {message: 401 Unauthorized}
`
	got := parseAuthStatus(out)
	want := []Account{{Host: "gitlab.com", User: "jacopofilonzi", Protocol: "ssh"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v\nwant %+v", got, want)
	}
}

func TestParseAuthStatusNotLoggedIn(t *testing.T) {
	if got := parseAuthStatus("You are not logged into any GitHub hosts. To log in, run: gh auth login\n"); len(got) != 0 {
		t.Fatalf("got %+v", got)
	}
}

func TestStates(t *testing.T) {
	gh := map[[2]string]string{
		{"completed", "success"}:   "success",
		{"completed", "failure"}:   "failure",
		{"completed", "timed_out"}: "failure",
		{"completed", "cancelled"}: "cancelled",
		{"completed", "skipped"}:   "skipped",
		{"in_progress", ""}:        "running",
		{"queued", ""}:             "pending",
	}
	for in, want := range gh {
		if got := ghState(in[0], in[1]); got != want {
			t.Errorf("ghState(%q, %q) = %q, want %q", in[0], in[1], got, want)
		}
	}
	gl := map[string]string{"success": "success", "failed": "failure", "running": "running", "canceled": "cancelled", "manual": "pending", "skipped": "skipped"}
	for in, want := range gl {
		if got := glState(in); got != want {
			t.Errorf("glState(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCloneURLFollowsProtocol(t *testing.T) {
	raw := []ghRepo{{FullName: "o/r", SSHURL: "git@github.com:o/r.git", CloneURL: "https://github.com/o/r.git"}}
	if got := ghRepos(raw, Account{Host: "github.com", Protocol: "ssh"})[0].CloneURL; got != "git@github.com:o/r.git" {
		t.Errorf("ssh: %s", got)
	}
	if got := ghRepos(raw, Account{Host: "github.com", Protocol: "https"})[0].CloneURL; got != "https://github.com/o/r.git" {
		t.Errorf("https: %s", got)
	}
	p := []glProject{{PathWithNamespace: "g/sub/r", Visibility: "internal", SSHURL: "git@gitlab.com:g/sub/r.git", HTTPURL: "https://gitlab.com/g/sub/r.git"}}
	r := glRepos(p, Account{Host: "gitlab.com", Protocol: "ssh"})[0]
	if r.CloneURL != "git@gitlab.com:g/sub/r.git" || !r.Private {
		t.Errorf("gitlab: %+v", r)
	}
}

func TestNames(t *testing.T) {
	if got := glPath("group/sub group/repo"); got != "group%2Fsub%20group%2Frepo" {
		t.Errorf("glPath = %s", got)
	}
	if o, n, ok := splitOwner("group/sub/repo"); !ok || o != "group/sub" || n != "repo" {
		t.Errorf("splitOwner = %s %s %v", o, n, ok)
	}
	if _, _, ok := splitOwner("repo"); ok {
		t.Error("splitOwner without owner")
	}
}
