package forge

import (
	"fmt"
	"net/url"
	"sync"
	"time"
)

// PR is an open pull request (GitHub) or merge request (GitLab).
type PR struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	URL    string `json:"url"`
	Author string `json:"author"`
	Draft  bool   `json:"draft"`
}

// CI is the latest CI run on the branch (GitHub Actions workflow or GitLab pipeline).
type CI struct {
	// State: "success", "failure", "running", "pending", "cancelled", "skipped".
	State string `json:"state"`
	Name  string `json:"name"`
	URL   string `json:"url"`
	// Runs is the number of workflows or pipelines of the last commit (state and link are the worst one's).
	Runs int `json:"runs"`
}

// Info is what the project card shows about the remote repository.
type Info struct {
	Kind Kind   `json:"kind"`
	Host string `json:"host"`
	Web  string `json:"web"`
	PRs  []PR   `json:"prs"`
	// PRCount is the total of open PRs; PRMore is true when the total is "at least PRCount" (GitLab).
	PRCount int  `json:"prCount"`
	PRMore  bool `json:"prMore"`
	// Issues is the number of open issues (-1 if not available, e.g. issues disabled).
	Issues int `json:"issues"`
	CI     *CI `json:"ci"`
}

// maxPRs: how many open PRs to list in the card.
const maxPRs = 5

// Info collects PRs, issues and CI of fullName (owner/name) on the given branch.
// The three requests run together; one that fails leaves only its own part empty.
func (c *CLI) Info(host, fullName, branch string) (Info, error) {
	ctx, cancel := timeout(30 * time.Second)
	defer cancel()
	info := Info{Kind: c.Kind, Host: host, Web: "https://" + host + "/" + fullName, PRs: []PR{}, Issues: -1}
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []error
	fail := func(err error) {
		mu.Lock()
		errs = append(errs, err)
		mu.Unlock()
	}
	run := func(fn func() error) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := fn(); err != nil {
				fail(err)
			}
		}()
	}

	if c.Kind == GitHub {
		owner, name, ok := splitOwner(fullName)
		if !ok {
			return info, fmt.Errorf("bad repository name: %s", fullName)
		}
		run(func() error {
			var r ghGraph
			err := c.api(ctx, host, &r, "graphql", "-f", "query="+ghInfoQuery, "-f", "owner="+owner, "-f", "name="+name)
			if err != nil {
				return err
			}
			repo := r.Data.Repository
			mu.Lock()
			info.Issues = repo.Issues.TotalCount
			info.PRCount = repo.PullRequests.TotalCount
			for _, n := range repo.PullRequests.Nodes {
				info.PRs = append(info.PRs, PR{Number: n.Number, Title: n.Title, URL: n.URL, Author: n.Author.Login, Draft: n.IsDraft})
			}
			mu.Unlock()
			return nil
		})
		if branch != "" {
			run(func() error {
				var r struct {
					Runs []ghRun `json:"workflow_runs"`
				}
				// only runs triggered by a push: scheduled or manual ones say nothing about the last commit
				q := "repos/" + fullName + "/actions/runs?per_page=20&event=push&branch=" + url.QueryEscape(branch)
				if err := c.api(ctx, host, &r, q); err != nil {
					return err
				}
				if ci := ghCI(r.Runs); ci != nil {
					mu.Lock()
					info.CI = ci
					mu.Unlock()
				}
				return nil
			})
		}
	} else {
		p := "projects/" + glPath(fullName)
		run(func() error {
			var r struct {
				OpenIssues *int `json:"open_issues_count"`
			}
			if err := c.api(ctx, host, &r, p); err != nil {
				return err
			}
			if r.OpenIssues != nil {
				mu.Lock()
				info.Issues = *r.OpenIssues
				mu.Unlock()
			}
			return nil
		})
		run(func() error {
			var r []struct {
				IID    int    `json:"iid"`
				Title  string `json:"title"`
				WebURL string `json:"web_url"`
				Draft  bool   `json:"draft"`
				WIP    bool   `json:"work_in_progress"`
				Author struct {
					Username string `json:"username"`
				} `json:"author"`
			}
			// up to 20 for the count (GitLab does not return the total in the response body)
			if err := c.api(ctx, host, &r, p+"/merge_requests?state=opened&order_by=updated_at&per_page=20"); err != nil {
				return err
			}
			mu.Lock()
			info.PRCount, info.PRMore = len(r), len(r) == 20
			for i, m := range r {
				if i == maxPRs {
					break
				}
				info.PRs = append(info.PRs, PR{Number: m.IID, Title: m.Title, URL: m.WebURL, Author: m.Author.Username, Draft: m.Draft || m.WIP})
			}
			mu.Unlock()
			return nil
		})
		if branch != "" {
			run(func() error {
				var r []struct {
					ID     int    `json:"id"`
					Status string `json:"status"`
					WebURL string `json:"web_url"`
				}
				if err := c.api(ctx, host, &r, p+"/pipelines?per_page=1&ref="+url.QueryEscape(branch)); err != nil {
					return err
				}
				if len(r) > 0 {
					mu.Lock()
					info.CI = &CI{State: glState(r[0].Status), Name: fmt.Sprintf("#%d", r[0].ID), URL: r[0].WebURL, Runs: 1}
					mu.Unlock()
				}
				return nil
			})
		}
	}
	wg.Wait()
	// error only if nothing arrived: with partial answers the card shows what there is
	if len(errs) > 0 && info.PRCount == 0 && info.Issues < 0 && info.CI == nil {
		return info, errs[0]
	}
	return info, nil
}

const ghInfoQuery = `query($owner: String!, $name: String!) {
  repository(owner: $owner, name: $name) {
    issues(states: OPEN) { totalCount }
    pullRequests(states: OPEN, first: 5, orderBy: {field: UPDATED_AT, direction: DESC}) {
      totalCount
      nodes { number title url isDraft author { login } }
    }
  }
}`

type ghGraph struct {
	Data struct {
		Repository struct {
			Issues struct {
				TotalCount int `json:"totalCount"`
			} `json:"issues"`
			PullRequests struct {
				TotalCount int `json:"totalCount"`
				Nodes      []struct {
					Number  int    `json:"number"`
					Title   string `json:"title"`
					URL     string `json:"url"`
					IsDraft bool   `json:"isDraft"`
					Author  struct {
						Login string `json:"login"`
					} `json:"author"`
				} `json:"nodes"`
			} `json:"pullRequests"`
		} `json:"repository"`
	} `json:"data"`
}

type ghRun struct {
	Name       string `json:"name"`
	HeadSHA    string `json:"head_sha"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	HTMLURL    string `json:"html_url"`
}

// stateRank: for the same commit the worst state wins (a failed workflow outweighs a passed one).
var stateRank = map[string]int{"failure": 5, "running": 4, "pending": 3, "cancelled": 2, "success": 1, "skipped": 0}

// ghCI merges the workflows of the last commit (the first of the runs, which come newest first):
// state, name and link are those of the worst workflow.
func ghCI(runs []ghRun) *CI {
	if len(runs) == 0 {
		return nil
	}
	sha := runs[0].HeadSHA
	var ci *CI
	n := 0
	for _, r := range runs {
		if r.HeadSHA != sha {
			continue
		}
		n++
		st := ghState(r.Status, r.Conclusion)
		if ci == nil || stateRank[st] > stateRank[ci.State] {
			ci = &CI{State: st, Name: r.Name, URL: r.HTMLURL}
		}
	}
	ci.Runs = n
	return ci
}

// ghState reduces the status/conclusion of a GitHub Actions workflow to the common states.
func ghState(status, conclusion string) string {
	if status != "completed" {
		if status == "in_progress" {
			return "running"
		}
		return "pending"
	}
	switch conclusion {
	case "success":
		return "success"
	case "cancelled":
		return "cancelled"
	case "skipped", "neutral":
		return "skipped"
	default: // failure, timed_out, startup_failure, action_required, stale
		return "failure"
	}
}

// glState reduces the state of a GitLab pipeline to the common states.
func glState(status string) string {
	switch status {
	case "success":
		return "success"
	case "failed":
		return "failure"
	case "running":
		return "running"
	case "canceled", "canceling":
		return "cancelled"
	case "skipped":
		return "skipped"
	default: // created, pending, preparing, waiting_for_resource, scheduled, manual
		return "pending"
	}
}

func splitOwner(fullName string) (owner, name string, ok bool) {
	for i := len(fullName) - 1; i >= 0; i-- {
		if fullName[i] == '/' {
			return fullName[:i], fullName[i+1:], i > 0 && i < len(fullName)-1
		}
	}
	return "", "", false
}
