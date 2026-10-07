package gitinfo

import "testing"

func TestWebURL(t *testing.T) {
	cases := map[string]string{
		"git@github.com:jacopofilonzi/TimeTable.git":       "https://github.com/jacopofilonzi/TimeTable",
		"https://github.com/jacopofilonzi/NtfyJS":          "https://github.com/jacopofilonzi/NtfyJS",
		"https://user:token@gitlab.com/group/sub/repo.git": "https://gitlab.com/group/sub/repo",
		"ssh://git@github.com:22/owner/repo.git":           "https://github.com/owner/repo",
		"git@github.com:dity-dev/discord-bot-java.git":     "https://github.com/dity-dev/discord-bot-java",
		"C:\\repos\\local.git":                             "",
		"not a url":                                        "",
	}
	for in, want := range cases {
		if got := WebURL(in); got != want {
			t.Errorf("WebURL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRepoName(t *testing.T) {
	host, owner, repo, ok := RepoName("git@github.com:curishi/play.evons.gg.git")
	if !ok || host != "github.com" || owner != "curishi" || repo != "play.evons.gg" {
		t.Fatalf("got %s %s %s %v", host, owner, repo, ok)
	}
}

func TestParseStatus(t *testing.T) {
	out := "# branch.oid 5428c14aaaa\n# branch.head main\n# branch.upstream origin/main\n# branch.ab +1 -2\n1 .M N... 100644 100644 100644 a b file.go\n? new.txt\n"
	var info Info
	parseStatus(out, &info)
	if info.Branch != "main" || !info.HasUpstream || info.Ahead != 1 || info.Behind != 2 || info.Dirty != 2 {
		t.Fatalf("%+v", info)
	}
	var det Info
	parseStatus("# branch.oid 06e157abcdef\n# branch.head (detached)\n", &det)
	if !det.Detached || det.Branch != "06e157a" {
		t.Fatalf("%+v", det)
	}
}
