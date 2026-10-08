// Package readme finds and reads a project's README and extracts a short description.
package readme

import (
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const maxSize = 1 << 20 // 1 MB: beyond this the README is truncated

// bom is the UTF-8 byte order mark, removed if the README starts with it.
var bom = string(rune(0xFEFF))

type Format string

const (
	Markdown Format = "markdown"
	Text     Format = "text"
)

type Readme struct {
	Name      string `json:"name"`
	Format    Format `json:"format"`
	Content   string `json:"content"`
	Truncated bool   `json:"truncated"`
}

// priority: lower = preferred when there are several READMEs.
func priority(lower string) (int, Format, bool) {
	switch lower {
	case "readme.md":
		return 0, Markdown, true
	case "readme.markdown", "readme.mdown", "readme.mkd":
		return 1, Markdown, true
	case "readme.mdx":
		return 2, Markdown, true
	case "readme.rst":
		return 3, Text, true
	case "readme.txt", "readme":
		return 4, Text, true
	}
	if strings.HasPrefix(lower, "readme.") {
		return 5, Text, true
	}
	return 0, "", false
}

// Find returns the name of the README in dir and its format, or "" if there is none.
func Find(dir string) (string, Format) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", ""
	}
	best, bestP, bestF := "", 99, Format("")
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if p, f, ok := priority(strings.ToLower(e.Name())); ok && p < bestP {
			best, bestP, bestF = e.Name(), p, f
		}
	}
	return best, bestF
}

// Read reads the README of dir. ok is false if it does not exist.
func Read(dir string) (Readme, bool, error) {
	name, format := Find(dir)
	if name == "" {
		return Readme{}, false, nil
	}
	f, err := os.Open(filepath.Join(dir, name))
	if err != nil {
		return Readme{}, false, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxSize+1))
	if err != nil {
		return Readme{}, false, err
	}
	r := Readme{Name: name, Format: format}
	if len(data) > maxSize {
		data = data[:maxSize]
		r.Truncated = true
	}
	r.Content = strings.TrimPrefix(string(data), bom)
	return r, true, nil
}

// Description returns the first text paragraph of the README of dir (max 240 characters), or "".
func Description(dir string) string {
	name, format := Find(dir)
	if name == "" {
		return ""
	}
	f, err := os.Open(filepath.Join(dir, name))
	if err != nil {
		return ""
	}
	defer f.Close()
	data, _ := io.ReadAll(io.LimitReader(f, 16<<10))
	return FirstParagraph(strings.TrimPrefix(string(data), bom), format)
}

var (
	reImage     = regexp.MustCompile(`!\[[^\]]*\]\([^)]*\)`)
	reLink      = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
	reRefLink   = regexp.MustCompile(`\[([^\]]*)\]\[[^\]]*\]`)
	reTag       = regexp.MustCompile(`<[^>]+>`)
	reEmphasis  = regexp.MustCompile("(\\*\\*|__|\\*|_|`)")
	reSpaces    = regexp.MustCompile(`\s+`)
	reRstHeader = regexp.MustCompile(`^[=\-~^"'#*+]{3,}$`)
)

// FirstParagraph extracts the first "text" paragraph: it skips front matter, headings, badges,
// images, HTML, code blocks, admonition quotes and lists.
func FirstParagraph(content string, format Format) string {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	i := 0
	if format == Markdown && len(lines) > 0 && strings.TrimSpace(lines[0]) == "---" {
		for i = 1; i < len(lines) && strings.TrimSpace(lines[i]) != "---"; i++ {
		}
		i++
	}
	var para []string
	inFence := false
	for ; i < len(lines); i++ {
		raw := lines[i]
		l := strings.TrimSpace(raw)
		if strings.HasPrefix(l, "```") || strings.HasPrefix(l, "~~~") {
			inFence = !inFence
			if len(para) > 0 {
				break
			}
			continue
		}
		if inFence {
			continue
		}
		if l == "" {
			if len(para) > 0 {
				break
			}
			continue
		}
		skip := strings.HasPrefix(l, "#") || strings.HasPrefix(l, "<") || strings.HasPrefix(l, "|") ||
			strings.HasPrefix(l, "- ") || strings.HasPrefix(l, "* ") || strings.HasPrefix(l, "+ ") ||
			strings.HasPrefix(l, ">") || strings.HasPrefix(l, "    ") || strings.HasPrefix(raw, "\t") ||
			reRstHeader.MatchString(l) || strings.HasPrefix(l, "..") || strings.HasPrefix(l, ":")
		// line made only of badges/images
		if !skip && strings.TrimSpace(reLink.ReplaceAllString(reImage.ReplaceAllString(l, ""), "")) == "" {
			skip = true
		}
		// setext heading (line followed by === or ---)
		if !skip && i+1 < len(lines) && reRstHeader.MatchString(strings.TrimSpace(lines[i+1])) && len(para) == 0 {
			i++
			skip = true
		}
		if skip {
			if len(para) > 0 {
				break
			}
			continue
		}
		para = append(para, l)
	}
	s := strings.Join(para, " ")
	s = reImage.ReplaceAllString(s, "")
	s = reLink.ReplaceAllString(s, "$1")
	s = reRefLink.ReplaceAllString(s, "$1")
	s = reTag.ReplaceAllString(s, "")
	s = reEmphasis.ReplaceAllString(s, "")
	s = strings.TrimSpace(reSpaces.ReplaceAllString(s, " "))
	if r := []rune(s); len(r) > 240 {
		s = strings.TrimSpace(string(r[:239])) + "…"
	}
	return s
}
