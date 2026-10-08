package forge

import (
	"os"
	"testing"
)

// TestLive really queries the installed CLI, with the user's login. Read-only.
// Enable it with PL_FORGE_LIVE=gh (or glab) and, optionally, PL_FORGE_REPO=owner/name and PL_FORGE_BRANCH.
func TestLive(t *testing.T) {
	kind := map[string]Kind{"gh": GitHub, "glab": GitLab}[os.Getenv("PL_FORGE_LIVE")]
	if kind == "" {
		t.Skip("PL_FORGE_LIVE not set")
	}
	c := &CLI{Kind: kind}
	if c.Detect("") == "" {
		t.Fatalf("%s not found", kind.Bin())
	}
	t.Logf("path: %s", c.Path())
	accs, msg := c.Accounts(true)
	t.Logf("accounts: %+v %s", accs, msg)
	if len(accs) == 0 {
		t.Fatal("no account")
	}
	repos, err := c.Repos(accs[0])
	if err != nil {
		t.Fatalf("repos: %v", err)
	}
	t.Logf("%d repositories, the first: %+v", len(repos), repos[0])
	owners, err := c.Owners(accs[0])
	t.Logf("owners: %+v err=%v", owners, err)
	if taken, err := c.Exists(accs[0], accs[0].User, repos[0].FullName[len(accs[0].User)+1:]); err != nil || !taken {
		t.Errorf("Exists(%s) = %v, %v: should exist", repos[0].FullName, taken, err)
	}
	if taken, err := c.Exists(accs[0], accs[0].User, "project-library-name-that-does-not-exist-7f3a"); err != nil || taken {
		t.Errorf("Exists(missing) = %v, %v", taken, err)
	}
	if repo := os.Getenv("PL_FORGE_REPO"); repo != "" {
		info, err := c.Info(accs[0].Host, repo, os.Getenv("PL_FORGE_BRANCH"))
		if err != nil {
			t.Fatalf("info: %v", err)
		}
		t.Logf("info: prs=%d more=%v issues=%d ci=%+v", info.PRCount, info.PRMore, info.Issues, info.CI)
		for _, pr := range info.PRs {
			t.Logf("  #%d %s (%s) draft=%v", pr.Number, pr.Title, pr.Author, pr.Draft)
		}
	}
}
