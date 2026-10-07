package scanner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jacopofilonzi/project-library/internal/config"
)

// build crea un albero di prova: i percorsi che finiscono con "/" sono cartelle.
func build(t *testing.T, paths ...string) string {
	t.Helper()
	root := t.TempDir()
	for _, p := range paths {
		full := filepath.Join(root, filepath.FromSlash(p))
		if p[len(p)-1] == '/' {
			if err := os.MkdirAll(full, 0o755); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("# x\n\nhello"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func opts() Options {
	return Options{Markers: config.DefaultMarkers, Ignore: config.DefaultIgnore, FilesAsProject: true, ShowEmpty: true, MaxDepth: 20}
}

func find(n *Node, rel ...string) *Node {
	for _, name := range rel {
		var next *Node
		for _, c := range n.Children {
			if c.Name == name {
				next = c
			}
		}
		if next == nil {
			return nil
		}
		n = next
	}
	return n
}

func TestClassification(t *testing.T) {
	root := build(t,
		"github/owner/withgit/.git/",
		"github/owner/node/package.json",
		"github/owner/node/node_modules/x/package.json",
		"local/gradle/settings.gradle.kts",
		"local/onlyfiles/notes.txt",
		"local/empty/",
		"local/onlyjunk/desktop.ini",
		"local/UNI/Ingegneria/Demo/build.gradle.kts",
		"local/.hidden/project/go.mod",
		"local/node_modules/pkg/package.json",
	)
	tree := Scan([]string{root}, opts())

	check := func(want Kind, rel ...string) *Node {
		t.Helper()
		n := find(tree, rel...)
		if n == nil {
			t.Fatalf("%v: not found", rel)
		}
		if n.Kind != want {
			t.Fatalf("%v: kind %s, want %s", rel, n.Kind, want)
		}
		return n
	}
	check(KindDir, "github")
	check(KindDir, "github", "owner")
	if n := check(KindProject, "github", "owner", "withgit"); !n.HasGit {
		t.Error("withgit should have HasGit")
	}
	if n := check(KindProject, "github", "owner", "node"); n.Lang != "Node" || len(n.Children) != 0 {
		t.Errorf("node: lang %q, children %d", n.Lang, len(n.Children))
	}
	if n := check(KindProject, "local", "gradle"); n.Lang != "Java" {
		t.Errorf("gradle lang %q", n.Lang)
	}
	check(KindProject, "local", "onlyfiles")
	check(KindEmpty, "local", "empty")
	check(KindEmpty, "local", "onlyjunk")
	check(KindDir, "local", "UNI", "Ingegneria")
	check(KindProject, "local", "UNI", "Ingegneria", "Demo")
	if find(tree, "local", ".hidden") != nil || find(tree, "local", "node_modules") != nil {
		t.Error("hidden and ignored folders must be skipped")
	}
	if tree.Count != 5 {
		t.Errorf("count %d, want 5", tree.Count)
	}
	// ordine: cartelle prima dei progetti
	local := find(tree, "local")
	if local.Children[0].Kind == KindProject {
		t.Error("dirs must come before projects")
	}
}

func TestOptions(t *testing.T) {
	root := build(t, "a/onlyfiles/notes.txt", "a/empty/", "a/proj/go.mod", "a/b/c/d/go.mod")

	o := opts()
	o.FilesAsProject = false
	o.ShowEmpty = false
	tree := Scan([]string{root}, o)
	if find(tree, "a", "empty") != nil {
		t.Error("empty folder hidden with ShowEmpty=false")
	}
	if find(tree, "a", "onlyfiles") != nil {
		t.Error("folder with only files and no subfolders is empty when FilesAsProject=false")
	}

	o = opts()
	o.Overrides = map[string]string{
		filepath.Join(root, "a", "empty"): config.OverrideProject,
		filepath.Join(root, "a", "proj"):  config.OverrideDir,
	}
	tree = Scan([]string{root}, o)
	if n := find(tree, "a", "empty"); n == nil || n.Kind != KindProject {
		t.Error("override project")
	}
	if n := find(tree, "a", "proj"); n == nil || n.Kind != KindEmpty {
		t.Error("override dir on folder without subfolders gives empty dir")
	}

	o = opts()
	o.MaxDepth = 2
	tree = Scan([]string{root}, o)
	if n := find(tree, "a", "b"); n == nil || len(n.Children) != 0 {
		t.Error("max depth must stop descending")
	}
}

func TestLauncherRules(t *testing.T) {
	root := build(t, "a/javaproj/build.gradle.kts", "a/web/package.json", "a/mixed/pom.xml", "a/mixed/package.json")
	o := opts()
	o.Rules = []Rule{
		{Patterns: []string{"pom.xml", "build.gradle*"}, Launcher: "intellij"},
		{Patterns: []string{"package.json"}, Launcher: "webstorm"},
		{Patterns: []string{"*.json"}, Launcher: "intellij"}, // duplicato: ignorato
	}
	tree := Scan([]string{root}, o)
	check := func(name string, want []string) {
		t.Helper()
		got := find(tree, "a", name).RuleLaunchers
		if len(got) != len(want) {
			t.Fatalf("%s: got %v want %v", name, got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("%s: got %v want %v", name, got, want)
			}
		}
	}
	check("javaproj", []string{"intellij"})
	check("web", []string{"webstorm", "intellij"})
	check("mixed", []string{"intellij", "webstorm"})
}

func TestMultipleRoots(t *testing.T) {
	r1 := build(t, "p/go.mod")
	r2 := build(t, "q/go.mod")
	tree := Scan([]string{r1, r2, filepath.Join(r1, "missing")}, opts())
	if tree.Kind != KindRoot || len(tree.Children) != 3 || tree.Count != 2 {
		t.Fatalf("root: kind %s children %d count %d", tree.Kind, len(tree.Children), tree.Count)
	}
	if !tree.Children[2].Missing {
		t.Error("missing root must be flagged")
	}
}
