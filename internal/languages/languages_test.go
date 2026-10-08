package languages

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, root, rel string, size int) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(strings.Repeat("x", size)), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestAnalyze(t *testing.T) {
	root := t.TempDir()
	write(t, root, "main.go", 7000)
	write(t, root, "frontend/src/App.svelte", 2000)
	write(t, root, "frontend/src/main.ts", 990)
	write(t, root, "frontend/src/tiny.css", 10) // sotto l'1%: va in Other
	write(t, root, "frontend/node_modules/x/index.js", 900000)
	write(t, root, "dist/bundle.min.js", 500000)
	write(t, root, "README.md", 5000)   // prosa: non conta
	write(t, root, ".git/objects/x", 9) // nascosta: saltata

	stats, partial := Analyze(root)
	if partial {
		t.Fatal("small project must not be partial")
	}
	names := make([]string, len(stats))
	for i, s := range stats {
		names[i] = s.Name
	}
	if strings.Join(names, ",") != "Go,Svelte,TypeScript,Other" {
		t.Fatalf("got %v", stats)
	}
	if stats[0].Percent < 69 || stats[0].Percent > 71 {
		t.Fatalf("Go percent %.1f", stats[0].Percent)
	}
}

func TestEmptyProject(t *testing.T) {
	stats, _ := Analyze(t.TempDir())
	if stats != nil {
		t.Fatalf("got %v", stats)
	}
}
