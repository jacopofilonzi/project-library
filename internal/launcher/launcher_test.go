package launcher

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestSplitArgs(t *testing.T) {
	cases := map[string][]string{
		`"{path}"`:                    {"{path}"},
		`-d "{path}"`:                 {"-d", "{path}"},
		`--new-window  "C:\a b\c" x`:  {"--new-window", `C:\a b\c`, "x"},
		`""`:                          {""},
		``:                            nil,
		`--goto "{path}/README.md":1`: {"--goto", "{path}/README.md:1"},
	}
	for in, want := range cases {
		if got := SplitArgs(in); !reflect.DeepEqual(got, want) {
			t.Errorf("SplitArgs(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestExpandAndTarget(t *testing.T) {
	root := filepath.Join("C:", "Users", "x", "Development")
	path := filepath.Join(root, "local", "UNI", "Software Engineering", "Demo")
	tg := NewTarget(path, []string{root})
	if tg.Source != "local" || tg.Group != "UNI/Software Engineering" || tg.Name != "Demo" {
		t.Fatalf("%+v", tg)
	}
	got := Expand(`-d "{path}" --title {name}@{source}`, tg)
	want := []string{"-d", path, "--title", "Demo@local"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q want %q", got, want)
	}
}
