package forge

import (
	"os"
	"testing"
)

// TestLive interroga davvero la CLI installata, con il login dell'utente. Solo in lettura.
// Si attiva con PL_FORGE_LIVE=gh (o glab) e, facoltativo, PL_FORGE_REPO=owner/nome e PL_FORGE_BRANCH.
func TestLive(t *testing.T) {
	kind := map[string]Kind{"gh": GitHub, "glab": GitLab}[os.Getenv("PL_FORGE_LIVE")]
	if kind == "" {
		t.Skip("PL_FORGE_LIVE non impostata")
	}
	c := &CLI{Kind: kind}
	if c.Detect("") == "" {
		t.Fatalf("%s non trovata", kind.Bin())
	}
	t.Logf("path: %s", c.Path())
	accs, msg := c.Accounts(true)
	t.Logf("accounts: %+v %s", accs, msg)
	if len(accs) == 0 {
		t.Fatal("nessun account")
	}
	repos, err := c.Repos(accs[0])
	if err != nil {
		t.Fatalf("repos: %v", err)
	}
	t.Logf("%d repository, il primo: %+v", len(repos), repos[0])
	owners, err := c.Owners(accs[0])
	t.Logf("owners: %+v err=%v", owners, err)
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
