package readme

import "testing"

func TestFirstParagraph(t *testing.T) {
	cases := []struct {
		name, in string
		format   Format
		want     string
	}{
		{"title and paragraph", "# Awake\n\nSelf-hosted Wake-on-LAN over the internet.\nSecond line.\n\n## More", Markdown,
			"Self-hosted Wake-on-LAN over the internet. Second line."},
		{"badges and links", "# X\n[![ci](a.svg)](b)\n\nA [client](https://x) for **ntfy**.", Markdown, "A client for ntfy."},
		{"code fence first", "# X\n```\ncode\n```\n\nReal text.", Markdown, "Real text."},
		{"blockquote note skipped", "# deploy\n\n> [!NOTE]\n> Disclaimer\n\nA tool.", Markdown, "A tool."},
		{"front matter", "---\ntitle: x\n---\nHello.", Markdown, "Hello."},
		{"setext title", "Title\n=====\n\nBody text.", Markdown, "Body text."},
		{"rst", "TimeTable\n=========\n\nSubscribe to timetables.", Text, "Subscribe to timetables."},
		{"empty", "# Only title\n", Markdown, ""},
	}
	for _, c := range cases {
		if got := FirstParagraph(c.in, c.format); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

func TestPriority(t *testing.T) {
	if p, f, _ := priority("readme.md"); p != 0 || f != Markdown {
		t.Fatal("README.md should be preferred markdown")
	}
	if _, f, _ := priority("readme.rst"); f != Text {
		t.Fatal("rst should be plain text")
	}
	if _, _, ok := priority("notes.md"); ok {
		t.Fatal("notes.md is not a readme")
	}
}
