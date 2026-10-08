// Package scanner builds the folder tree and decides what is a project.
//
// For every folder, in order:
//  1. manual override (project / folder)
//  2. ignored names and hidden folders are skipped
//  3. contains a marker → project (the scan does not go further down)
//  4. contains at least one regular file → project (if FilesAsProject)
//  5. contains only subfolders → folder, the scan goes down
//  6. empty → empty folder (shown if ShowEmpty)
package scanner

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jacopofilonzi/project-library/internal/presets"
	"github.com/jacopofilonzi/project-library/internal/readme"
)

type Kind string

const (
	KindDir     Kind = "dir"
	KindProject Kind = "project"
	KindEmpty   Kind = "empty"
	// KindRoot is the virtual node that holds several root folders.
	KindRoot Kind = "root"
)

type Node struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Kind    Kind   `json:"kind"`
	Lang    string `json:"lang,omitempty"`
	Desc    string `json:"desc,omitempty"`
	HasGit  bool   `json:"hasGit,omitempty"`
	Missing bool   `json:"missing,omitempty"` // root that does not exist on disk
	// RuleLaunchers: launchers of the rules that match the project, in priority order.
	// The frontend uses the first enabled one.
	RuleLaunchers []string `json:"ruleLaunchers,omitempty"`
	Count         int      `json:"count"` // projects in the subtree
	Children      []*Node  `json:"children,omitempty"`
}

type Options struct {
	Markers        []string
	Ignore         []string
	FilesAsProject bool
	ShowEmpty      bool
	FollowLinks    bool
	MaxDepth       int
	// Overrides: absolute path → "project" | "dir".
	Overrides       map[string]string
	CaseInsensitive bool
	// Rules chooses a project's launcher based on the names it contains.
	Rules []Rule
}

type Rule struct {
	Patterns []string
	Launcher string
}

// junk are system files that do not make a folder a project.
var junk = map[string]bool{"desktop.ini": true, "thumbs.db": true, ".ds_store": true, ".localized": true}

// Scan scans the roots. With a single root it returns that root; with several roots
// a virtual node (KindRoot) that contains them.
func Scan(roots []string, opt Options) *Node {
	s := &scan{opt: opt, visited: map[string]bool{}}
	var nodes []*Node
	for _, r := range roots {
		r = filepath.Clean(r)
		n := &Node{Name: filepath.Base(r), Path: r, Kind: KindDir}
		if fi, err := os.Stat(r); err != nil || !fi.IsDir() {
			n.Missing = true
			n.Kind = KindEmpty
		} else {
			s.dir(n, 0)
		}
		nodes = append(nodes, n)
	}
	if len(nodes) == 1 {
		return nodes[0]
	}
	root := &Node{Kind: KindRoot, Children: nodes}
	for _, n := range nodes {
		root.Count += n.Count
	}
	return root
}

type scan struct {
	opt     Options
	visited map[string]bool // with FollowLinks: real folders already visited (avoids loops)
}

func (s *scan) key(name string) string {
	if s.opt.CaseInsensitive {
		return strings.ToLower(name)
	}
	return name
}

func (s *scan) ignored(name string) bool {
	if strings.HasPrefix(name, ".") {
		return true
	}
	for _, p := range s.opt.Ignore {
		if match(p, name) {
			return true
		}
	}
	return false
}

// match compares a glob pattern with a name, case-insensitively.
func match(pattern, name string) bool {
	ok, err := filepath.Match(strings.ToLower(pattern), strings.ToLower(name))
	return err == nil && ok
}

// dir fills n (a folder already classified as "to explore").
func (s *scan) dir(n *Node, depth int) {
	if s.opt.FollowLinks {
		if real, err := filepath.EvalSymlinks(n.Path); err == nil {
			if s.visited[s.key(real)] {
				return
			}
			s.visited[s.key(real)] = true
		}
	}
	entries, err := os.ReadDir(n.Path)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !s.isDir(n.Path, e) || s.ignored(e.Name()) {
			continue
		}
		child := s.classify(filepath.Join(n.Path, e.Name()), e.Name(), depth+1)
		if child == nil {
			continue
		}
		n.Children = append(n.Children, child)
		n.Count += child.Count
	}
	sortNodes(n.Children)
}

// isDir tells whether the entry is a folder to consider. Symlinks and junctions
// (which Go reports as ModeIrregular on Windows) only with FollowLinks.
func (s *scan) isDir(parent string, e fs.DirEntry) bool {
	t := e.Type()
	if t.IsDir() {
		return true
	}
	if t&(fs.ModeSymlink|fs.ModeIrregular) == 0 || !s.opt.FollowLinks {
		return false
	}
	fi, err := os.Stat(filepath.Join(parent, e.Name()))
	return err == nil && fi.IsDir()
}

func (s *scan) classify(path, name string, depth int) *Node {
	n := &Node{Name: name, Path: path}

	if ov, ok := s.override(path); ok && ov == "project" {
		s.makeProject(n, nil)
		return n
	}
	forceDir := false
	if ov, ok := s.override(path); ok && ov == "dir" {
		forceDir = true
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil
	}

	if !forceDir {
		var marker []string
		hasFile := false
		for _, e := range entries {
			nm := e.Name()
			for _, m := range s.opt.Markers {
				if match(m, nm) {
					marker = append(marker, nm)
					break
				}
			}
			if !e.IsDir() && !junk[strings.ToLower(nm)] && !s.ignoredFile(nm) {
				hasFile = true
			}
		}
		if len(marker) > 0 || (hasFile && s.opt.FilesAsProject) {
			s.makeProject(n, entries)
			return n
		}
	}

	hasSub := false
	for _, e := range entries {
		if s.isDir(path, e) && !s.ignored(e.Name()) {
			hasSub = true
			break
		}
	}
	if !hasSub {
		if !s.opt.ShowEmpty {
			return nil
		}
		n.Kind = KindEmpty
		return n
	}
	n.Kind = KindDir
	if depth < s.opt.MaxDepth {
		s.dir(n, depth)
	}
	return n
}

// ignoredFile: the files in the ignore list do not count for the "folder with files" rule.
func (s *scan) ignoredFile(name string) bool {
	for _, p := range s.opt.Ignore {
		if match(p, name) {
			return true
		}
	}
	return false
}

func (s *scan) override(path string) (string, bool) {
	if s.opt.Overrides == nil {
		return "", false
	}
	if v, ok := s.opt.Overrides[path]; ok {
		return v, true
	}
	if s.opt.CaseInsensitive {
		for k, v := range s.opt.Overrides {
			if strings.EqualFold(k, path) {
				return v, true
			}
		}
	}
	return "", false
}

func (s *scan) makeProject(n *Node, entries []fs.DirEntry) {
	n.Kind = KindProject
	n.Count = 1
	if entries == nil {
		entries, _ = os.ReadDir(n.Path)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
		if e.Name() == ".git" {
			n.HasGit = true
		}
	}
	n.Lang = DetectLang(names)
	n.RuleLaunchers = MatchRules(s.opt.Rules, n.Path, names)
	n.Desc = readme.Description(n.Path)
}

// MatchRules returns, without duplicates and in rule order, the launchers of the rules
// whose preset matches the project in dir (names: the names in its top folder).
func MatchRules(rules []Rule, dir string, names []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, r := range rules {
		if r.Launcher == "" || seen[r.Launcher] {
			continue
		}
		if presets.Matches(dir, names, r.Patterns) {
			out = append(out, r.Launcher)
			seen[r.Launcher] = true
		}
	}
	return out
}

// langRules: the first marker found decides the language shown.
var langRules = []struct{ pattern, lang string }{
	{"go.mod", "Go"},
	{"Cargo.toml", "Rust"},
	{"deno.json", "Deno"},
	{"package.json", "Node"},
	{"pom.xml", "Java"}, {"build.gradle*", "Java"}, {"settings.gradle*", "Java"}, {"gradlew", "Java"},
	{"pyproject.toml", "Python"}, {"requirements.txt", "Python"}, {"setup.py", "Python"},
	{"composer.json", "PHP"},
	{"Gemfile", "Ruby"},
	{"*.sln", "C#"}, {"*.csproj", "C#"},
	{"pubspec.yaml", "Dart"},
	{"mix.exs", "Elixir"},
	{"CMakeLists.txt", "C/C++"},
}

func DetectLang(names []string) string {
	for _, r := range langRules {
		for _, n := range names {
			if match(r.pattern, n) {
				return r.lang
			}
		}
	}
	return ""
}

// sortNodes: folders first (full and empty), then projects; by name within each group.
func sortNodes(ns []*Node) {
	sort.SliceStable(ns, func(i, j int) bool {
		pi, pj := ns[i].Kind == KindProject, ns[j].Kind == KindProject
		if pi != pj {
			return !pi
		}
		return strings.ToLower(ns[i].Name) < strings.ToLower(ns[j].Name)
	})
}

// Walk visits all the nodes of the tree.
func Walk(n *Node, fn func(*Node)) {
	if n == nil {
		return
	}
	fn(n)
	for _, c := range n.Children {
		Walk(c, fn)
	}
}
