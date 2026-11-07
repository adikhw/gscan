package main

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestNeedsAttention(t *testing.T) {
	inSync := RepoStatus{Branch: "main", HasUpstream: true}

	tests := []struct {
		name string
		st   RepoStatus
		want bool
	}{
		{"clean and in sync", inSync, false},
		{"behind only", RepoStatus{Branch: "main", HasUpstream: true, Behind: 3}, false},
		{"changed", RepoStatus{Branch: "main", HasUpstream: true, Changed: 1}, true},
		{"untracked", RepoStatus{Branch: "main", HasUpstream: true, Untracked: 1}, true},
		{"ahead", RepoStatus{Branch: "main", HasUpstream: true, Ahead: 1}, true},
		{"no remote", RepoStatus{Branch: "main", NoRemote: true}, true},
		{"no upstream", RepoStatus{Branch: "main"}, true},
		{"no commits and no remote", RepoStatus{Branch: "main", NoCommits: true}, false},
		{"no commits with untracked", RepoStatus{Branch: "main", NoCommits: true, Untracked: 1}, true},
		{"detached", RepoStatus{Branch: "HEAD (detached)", Detached: true}, true},
		{"upstream gone", RepoStatus{Branch: "main", UpstreamGone: true}, true},
		{"error", RepoStatus{Err: errors.New("boom")}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := needsAttention(tc.st); got != tc.want {
				t.Errorf("needsAttention(%+v) = %v, want %v", tc.st, got, tc.want)
			}
		})
	}
}

func TestSummarize(t *testing.T) {
	tests := []struct {
		name string
		st   RepoStatus
		want string
	}{
		{"clean", RepoStatus{Branch: "main", HasUpstream: true}, "clean"},
		{"working tree", RepoStatus{Branch: "main", HasUpstream: true, Changed: 2, Untracked: 1}, "2 changed, 1 untracked"},
		{"ahead and behind", RepoStatus{Branch: "main", HasUpstream: true, Ahead: 1, Behind: 3}, "ahead 1, behind 3"},
		{"no commits", RepoStatus{Branch: "main", NoCommits: true}, "no commits"},
		{"no remote", RepoStatus{Branch: "main", NoRemote: true}, "no remote"},
		{"no upstream", RepoStatus{Branch: "main"}, "no upstream"},
		{"upstream gone", RepoStatus{Branch: "main", UpstreamGone: true}, "upstream gone"},
		{"detached", RepoStatus{Branch: "HEAD (detached)", Detached: true}, "detached"},
		{"fetch failed", RepoStatus{Branch: "main", HasUpstream: true, FetchErr: errors.New("network down")}, "fetch failed"},
		{"error", RepoStatus{Err: errors.New("fatal: bad repo\nsecond line")}, "error: fatal: bad repo"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := strings.Join(summarize(tc.st), ", ")
			if got != tc.want {
				t.Errorf("summarize() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestShortPath(t *testing.T) {
	home := "/home/user"
	tests := []struct {
		path string
		want string
	}{
		{"/home/user/projects/app", "~/projects/app"},
		{"/home/user", "~"},
		{"/home/user2/app", "/home/user2/app"},
		{"/opt/app", "/opt/app"},
	}

	for _, tc := range tests {
		if got := shortPath(tc.path, home); got != tc.want {
			t.Errorf("shortPath(%q) = %q, want %q", tc.path, got, tc.want)
		}
	}
	if got := shortPath("/opt/app", ""); got != "/opt/app" {
		t.Errorf("shortPath with empty home = %q, want /opt/app", got)
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		in   time.Duration
		want string
	}{
		{0, "0ms"},
		{500 * time.Millisecond, "500ms"},
		{1500 * time.Millisecond, "1.50s"},
	}

	for _, tc := range tests {
		if got := formatDuration(tc.in); got != tc.want {
			t.Errorf("formatDuration(%s) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestColorEnabled(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("CLICOLOR", "")
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("FORCE_COLOR", "")

	if !colorEnabled(colorAlways, io.Discard) {
		t.Error("always should enable color")
	}
	if colorEnabled(colorNever, io.Discard) {
		t.Error("never should disable color")
	}
	if colorEnabled(colorAuto, io.Discard) {
		t.Error("auto with a non-terminal writer should disable color")
	}

	t.Setenv("FORCE_COLOR", "1")
	if !colorEnabled(colorAuto, io.Discard) {
		t.Error("FORCE_COLOR should enable color")
	}

	t.Setenv("NO_COLOR", "1")
	if colorEnabled(colorAuto, io.Discard) {
		t.Error("NO_COLOR should win over FORCE_COLOR")
	}

	t.Setenv("NO_COLOR", "")
	t.Setenv("CLICOLOR", "0")
	if colorEnabled(colorAuto, io.Discard) {
		t.Error("CLICOLOR=0 should disable color")
	}

	t.Setenv("CLICOLOR", "")
	t.Setenv("TERM", "dumb")
	if colorEnabled(colorAuto, io.Discard) {
		t.Error("TERM=dumb should disable color")
	}
}

func TestPrintReportFiltersAndSummarizes(t *testing.T) {
	home := "/home/user"
	results := []RepoStatus{
		{Path: "/home/user/clean", Branch: "main", HasUpstream: true},
		{Path: "/home/user/dirty", Branch: "main", HasUpstream: true, Changed: 1},
	}

	var buf bytes.Buffer
	printReport(&buf, results, options{colorMode: colorNever}, home, time.Second)

	out := buf.String()
	if strings.Contains(out, "~/clean") {
		t.Errorf("clean repo should be hidden without -all:\n%s", out)
	}
	if !strings.Contains(out, "~/dirty") || !strings.Contains(out, "1 changed") {
		t.Errorf("dirty repo missing from report:\n%s", out)
	}
	if !strings.Contains(out, "Scanned 2 repositories") || !strings.Contains(out, "1 needs attention") {
		t.Errorf("summary missing:\n%s", out)
	}
	if strings.Contains(out, "\x1b[") {
		t.Errorf("color never produced ANSI escapes:\n%q", out)
	}
}

func TestPrintReportAll(t *testing.T) {
	home := "/home/user"
	results := []RepoStatus{
		{Path: "/home/user/clean", Branch: "main", HasUpstream: true},
		{Path: "/home/user/dirty", Branch: "main", HasUpstream: true, Changed: 1},
	}

	var buf bytes.Buffer
	printReport(&buf, results, options{all: true, colorMode: colorNever}, home, time.Second)

	out := buf.String()
	if !strings.Contains(out, "~/clean") || !strings.Contains(out, "clean") {
		t.Errorf("clean repo missing with -all:\n%s", out)
	}
	if !strings.Contains(out, "1 clean") {
		t.Errorf("clean count missing from summary:\n%s", out)
	}
}

func TestPrintReportClean(t *testing.T) {
	results := []RepoStatus{{Path: "/tmp/repo", Branch: "main", HasUpstream: true}}

	var buf bytes.Buffer
	printReport(&buf, results, options{colorMode: colorNever}, "", time.Second)

	out := buf.String()
	if !strings.Contains(out, "Everything is clean and pushed.") {
		t.Errorf("clean message missing:\n%s", out)
	}
}

func TestPrintReportBehindOnly(t *testing.T) {
	results := []RepoStatus{{Path: "/tmp/repo", Branch: "main", HasUpstream: true, Behind: 2}}

	var buf bytes.Buffer
	printReport(&buf, results, options{colorMode: colorNever}, "", time.Second)

	out := buf.String()
	if !strings.Contains(out, "No local work pending") {
		t.Errorf("behind-only message missing:\n%s", out)
	}
	if !strings.Contains(out, "1 behind remote") {
		t.Errorf("behind count missing:\n%s", out)
	}
}

func TestPrintReportErrorRow(t *testing.T) {
	results := []RepoStatus{{Path: "/tmp/broken", Err: errors.New("fatal: not a git repository")}}

	var buf bytes.Buffer
	printReport(&buf, results, options{colorMode: colorNever}, "", time.Second)

	out := buf.String()
	if !strings.Contains(out, "error: fatal: not a git repository") {
		t.Errorf("error row missing:\n%s", out)
	}
	if !strings.Contains(out, "1 error") {
		t.Errorf("error count missing:\n%s", out)
	}
}

func TestPrintReportColor(t *testing.T) {
	results := []RepoStatus{{Path: "/tmp/repo", Branch: "main", HasUpstream: true, Changed: 1}}

	var buf bytes.Buffer
	printReport(&buf, results, options{colorMode: colorAlways}, "", time.Second)

	if !strings.Contains(buf.String(), "\x1b[") {
		t.Errorf("color always produced no ANSI escapes:\n%q", buf.String())
	}
}

func TestPrintReportRuneAlignment(t *testing.T) {
	results := []RepoStatus{
		{Path: "/tmp/aa", Branch: "main", HasUpstream: true},
		{Path: "/tmp/éé", Branch: "main", HasUpstream: true},
	}

	var buf bytes.Buffer
	printReport(&buf, results, options{all: true, colorMode: colorNever}, "", time.Second)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines, got:\n%s", buf.String())
	}
	first := strings.Index(lines[0], "clean")
	second := strings.Index(lines[1], "clean")
	if first < 0 || second < 0 {
		t.Fatalf("clean status missing:\n%s", buf.String())
	}
	firstRunes := utf8.RuneCountInString(lines[0][:first])
	secondRunes := utf8.RuneCountInString(lines[1][:second])
	if firstRunes != secondRunes {
		t.Errorf("status columns are not aligned: %d vs %d runes\n%s", firstRunes, secondRunes, buf.String())
	}
}
