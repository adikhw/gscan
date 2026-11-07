package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestPrintJSON(t *testing.T) {
	results := []RepoStatus{
		{Path: "/repos/b", Branch: "main", Upstream: "origin/main", HasUpstream: true, Changed: 1},
		{Path: "/repos/a", Branch: "main", Upstream: "origin/main", HasUpstream: true, Behind: 2},
		{Path: "/repos/c", Err: errors.New("boom"), FetchErr: errors.New("network down")},
	}

	var buf bytes.Buffer
	if err := printJSON(&buf, "/repos", results, options{workers: 4}, 1500*time.Millisecond); err != nil {
		t.Fatalf("printJSON() error = %v", err)
	}
	if !json.Valid(buf.Bytes()) {
		t.Fatalf("printJSON() produced invalid JSON:\n%s", buf.String())
	}

	var report jsonReport
	if err := json.Unmarshal(buf.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if report.Schema != jsonSchemaVersion {
		t.Errorf("Schema = %q, want %q", report.Schema, jsonSchemaVersion)
	}
	if report.Root != "/repos" {
		t.Errorf("Root = %q, want /repos", report.Root)
	}
	if report.Workers != 4 {
		t.Errorf("Workers = %d, want 4", report.Workers)
	}
	if report.ElapsedMS != 1500 {
		t.Errorf("ElapsedMS = %d, want 1500", report.ElapsedMS)
	}
	if report.Elapsed != "1.50s" {
		t.Errorf("Elapsed = %q, want 1.50s", report.Elapsed)
	}

	wantSummary := jsonSummary{Total: 3, Attention: 2, Clean: 1, Dirty: 1, Behind: 1, Errors: 1}
	if report.Summary != wantSummary {
		t.Errorf("Summary = %+v, want %+v", report.Summary, wantSummary)
	}

	if len(report.Repositories) != 3 {
		t.Fatalf("len(Repositories) = %d, want 3", len(report.Repositories))
	}
	wantPaths := []string{"/repos/a", "/repos/b", "/repos/c"}
	for i, want := range wantPaths {
		if report.Repositories[i].Path != want {
			t.Errorf("Repositories[%d].Path = %q, want %q", i, report.Repositories[i].Path, want)
		}
	}

	if got := report.Repositories[0].Status; len(got) != 1 || got[0] != "behind 2" {
		t.Errorf("behind status = %v, want [behind 2]", got)
	}
	if report.Repositories[0].Attention {
		t.Error("behind-only repository should not need attention")
	}
	if got := report.Repositories[1].Status; len(got) != 1 || got[0] != "1 changed" {
		t.Errorf("changed status = %v, want [1 changed]", got)
	}
	if report.Repositories[2].Error != "boom" {
		t.Errorf("Error = %q, want boom", report.Repositories[2].Error)
	}
	if report.Repositories[2].FetchError != "network down" {
		t.Errorf("FetchError = %q, want network down", report.Repositories[2].FetchError)
	}

	if bytes.Contains(buf.Bytes(), []byte("\x1b[")) {
		t.Error("JSON output contains ANSI escapes")
	}
}

func TestPrintJSONEmptyRepositories(t *testing.T) {
	var buf bytes.Buffer
	if err := printJSON(&buf, "/empty", nil, options{workers: 1}, 0); err != nil {
		t.Fatalf("printJSON() error = %v", err)
	}

	var report jsonReport
	if err := json.Unmarshal(buf.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if report.Repositories == nil {
		t.Error("Repositories = nil, want an empty array")
	}
	if len(report.Repositories) != 0 {
		t.Errorf("len(Repositories) = %d, want 0", len(report.Repositories))
	}
}
