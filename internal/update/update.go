// Package update checks whether a newer release of the app is published on GitHub.
// It reads the public releases API: no login, no token. Drafts and pre-releases are
// never offered (GitHub's "latest" release excludes them).
package update

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Repo is the GitHub repository the releases come from.
const Repo = "jacopofilonzi/project-library"

// API is the base URL of the GitHub API (replaced in the tests).
var API = "https://api.github.com"

// Info is the outcome of a check.
type Info struct {
	Available bool   `json:"available"`
	Current   string `json:"current"`
	Latest    string `json:"latest"`   // e.g. "1.1.0", without the "v"
	NotesURL  string `json:"notesUrl"` // release page on GitHub
	// DownloadURL is the direct link to the installer for this system; the release page if there is none.
	DownloadURL string `json:"downloadUrl"`
}

type release struct {
	TagName    string `json:"tag_name"`
	HTMLURL    string `json:"html_url"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
	Assets     []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	} `json:"assets"`
}

// Check asks GitHub for the latest release and compares it with current.
// assetSuffix picks the installer among the release files (e.g. "-installer.exe"); empty = none for this system.
func Check(ctx context.Context, current, assetSuffix string) (Info, error) {
	info := Info{Current: current}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, API+"/repos/"+Repo+"/releases/latest", nil)
	if err != nil {
		return info, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "project-library/"+current)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return info, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return info, nil // no release published yet
	}
	if resp.StatusCode != http.StatusOK {
		return info, fmt.Errorf("GitHub answered %s", resp.Status)
	}
	var r release
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return info, fmt.Errorf("unexpected response: %w", err)
	}
	if r.Draft || r.Prerelease {
		return info, nil
	}
	info.Latest = strings.TrimPrefix(r.TagName, "v")
	info.NotesURL = r.HTMLURL
	info.DownloadURL = r.HTMLURL
	if assetSuffix != "" {
		for _, a := range r.Assets {
			if strings.HasSuffix(a.Name, assetSuffix) {
				info.DownloadURL = a.URL
				break
			}
		}
	}
	info.Available = Newer(info.Latest, current)
	return info, nil
}

// Newer tells whether version a is newer than b. Both are "x.y.z" (a leading "v" is allowed);
// versions with a pre-release or unreadable part are never newer.
func Newer(a, b string) bool {
	pa, okA := parse(a)
	pb, okB := parse(b)
	if !okA || !okB {
		return false
	}
	for i := range pa {
		if pa[i] != pb[i] {
			return pa[i] > pb[i]
		}
	}
	return false
}

func parse(v string) ([3]int, bool) {
	var out [3]int
	parts := strings.Split(strings.TrimPrefix(strings.TrimSpace(v), "v"), ".")
	if len(parts) != 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return out, false
		}
		out[i] = n
	}
	return out, true
}
