package update

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewer(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"1.1.0", "1.0.0", true},
		{"v1.10.0", "1.9.0", true},
		{"1.0.1", "1.0.0", true},
		{"2.0.0", "1.99.99", true},
		{"1.0.0", "1.0.0", false},
		{"1.0.0", "1.1.0", false},
		{"1.1.0-beta", "1.0.0", false},
		{"1.1", "1.0.0", false},
		{"1.1.0", "0.0.0-e2e", false},
	}
	for _, c := range cases {
		if got := Newer(c.a, c.b); got != c.want {
			t.Errorf("Newer(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

func serve(t *testing.T, status int, body string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/"+Repo+"/releases/latest" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.WriteHeader(status)
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	old := API
	API = srv.URL
	t.Cleanup(func() { API = old })
}

const releaseJSON = `{"tag_name":"v1.1.0","html_url":"https://github.com/o/r/releases/tag/v1.1.0","assets":[
 {"name":"notes.txt","browser_download_url":"https://x/notes.txt"},
 {"name":"project-library-amd64-installer.exe","browser_download_url":"https://x/installer.exe"}]}`

func TestCheckFindsInstaller(t *testing.T) {
	serve(t, 200, releaseJSON)
	info, err := Check(context.Background(), "1.0.0", "-installer.exe")
	if err != nil {
		t.Fatal(err)
	}
	if !info.Available || info.Latest != "1.1.0" || info.DownloadURL != "https://x/installer.exe" || info.NotesURL == "" {
		t.Fatalf("got %+v", info)
	}
}

func TestCheckWithoutInstallerFallsBackToReleasePage(t *testing.T) {
	serve(t, 200, releaseJSON)
	info, _ := Check(context.Background(), "1.0.0", "")
	if info.DownloadURL != "https://github.com/o/r/releases/tag/v1.1.0" {
		t.Fatalf("got %+v", info)
	}
}

func TestCheckUpToDateAndNoRelease(t *testing.T) {
	serve(t, 200, releaseJSON)
	if info, _ := Check(context.Background(), "1.1.0", "-installer.exe"); info.Available {
		t.Fatalf("same version offered: %+v", info)
	}
	serve(t, 404, `{"message":"Not Found"}`)
	if info, err := Check(context.Background(), "1.0.0", "-installer.exe"); err != nil || info.Available {
		t.Fatalf("no release: %+v %v", info, err)
	}
	serve(t, 500, `oops`)
	if _, err := Check(context.Background(), "1.0.0", "-installer.exe"); err == nil {
		t.Fatal("server error not reported")
	}
}

func TestCheckSkipsPrerelease(t *testing.T) {
	serve(t, 200, `{"tag_name":"v2.0.0","prerelease":true,"html_url":"h"}`)
	if info, _ := Check(context.Background(), "1.0.0", ""); info.Available {
		t.Fatalf("pre-release offered: %+v", info)
	}
}
