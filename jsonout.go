package main

import (
	"encoding/json"
	"io"
	"sort"
	"time"
)

const jsonSchemaVersion = "1"

type jsonReport struct {
	Schema       string      `json:"schema"`
	Version      string      `json:"version"`
	Root         string      `json:"root"`
	ElapsedMS    int64       `json:"elapsed_ms"`
	Elapsed      string      `json:"elapsed"`
	Workers      int         `json:"workers"`
	Summary      jsonSummary `json:"summary"`
	Repositories []jsonRepo  `json:"repositories"`
}

type jsonSummary struct {
	Total     int `json:"total"`
	Attention int `json:"attention"`
	Clean     int `json:"clean"`
	Dirty     int `json:"dirty"`
	Unpushed  int `json:"unpushed"`
	Behind    int `json:"behind"`
	Errors    int `json:"errors"`
}

type jsonRepo struct {
	Path         string   `json:"path"`
	Branch       string   `json:"branch,omitempty"`
	Upstream     string   `json:"upstream,omitempty"`
	HasUpstream  bool     `json:"has_upstream"`
	UpstreamGone bool     `json:"upstream_gone,omitempty"`
	Detached     bool     `json:"detached,omitempty"`
	NoCommits    bool     `json:"no_commits,omitempty"`
	NoRemote     bool     `json:"no_remote,omitempty"`
	Ahead        int      `json:"ahead"`
	Behind       int      `json:"behind"`
	Changed      int      `json:"changed"`
	Untracked    int      `json:"untracked"`
	Attention    bool     `json:"attention"`
	Status       []string `json:"status"`
	Error        string   `json:"error,omitempty"`
	FetchError   string   `json:"fetch_error,omitempty"`
}

// printJSON always includes every scanned repository. The attention flag lets
// consumers apply the same filtering that the text report does by default.
func printJSON(w io.Writer, root string, results []RepoStatus, opts options, elapsed time.Duration) error {
	rows := append([]RepoStatus(nil), results...)
	sort.Slice(rows, func(i, j int) bool { return rows[i].Path < rows[j].Path })

	report := jsonReport{
		Schema:       jsonSchemaVersion,
		Version:      version,
		Root:         root,
		ElapsedMS:    elapsed.Milliseconds(),
		Elapsed:      formatDuration(elapsed),
		Workers:      opts.workers,
		Repositories: make([]jsonRepo, 0, len(rows)),
	}

	for _, st := range rows {
		repo := jsonRepo{
			Path:         st.Path,
			Branch:       st.Branch,
			Upstream:     st.Upstream,
			HasUpstream:  st.HasUpstream,
			UpstreamGone: st.UpstreamGone,
			Detached:     st.Detached,
			NoCommits:    st.NoCommits,
			NoRemote:     st.NoRemote,
			Ahead:        st.Ahead,
			Behind:       st.Behind,
			Changed:      st.Changed,
			Untracked:    st.Untracked,
			Attention:    needsAttention(st),
			Status:       summarize(st),
		}
		if st.Err != nil {
			repo.Error = shortError(st.Err)
		}
		if st.FetchErr != nil {
			repo.FetchError = shortError(st.FetchErr)
		}
		report.Repositories = append(report.Repositories, repo)

		report.Summary.Total++
		if repo.Attention {
			report.Summary.Attention++
		} else {
			report.Summary.Clean++
		}
		if st.Changed > 0 || st.Untracked > 0 {
			report.Summary.Dirty++
		}
		if st.Ahead > 0 {
			report.Summary.Unpushed++
		}
		if st.Behind > 0 {
			report.Summary.Behind++
		}
		if st.Err != nil {
			report.Summary.Errors++
		}
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(report)
}
