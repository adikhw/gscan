package main

import (
	"strings"
	"testing"
)

func TestParseStatus(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want RepoStatus
	}{
		{
			name: "empty output",
			in:   "",
			want: RepoStatus{},
		},
		{
			name: "clean and in sync",
			in:   "## main...origin/main\n",
			want: RepoStatus{Branch: "main", Upstream: "origin/main", HasUpstream: true},
		},
		{
			name: "ahead with changed and untracked files",
			in:   "## main...origin/main [ahead 2]\n M main.go\n?? notes.txt\n",
			want: RepoStatus{Branch: "main", Upstream: "origin/main", HasUpstream: true, Ahead: 2, Changed: 1, Untracked: 1},
		},
		{
			name: "ahead and behind",
			in:   "## dev...origin/dev [ahead 1, behind 3]\n",
			want: RepoStatus{Branch: "dev", Upstream: "origin/dev", HasUpstream: true, Ahead: 1, Behind: 3},
		},
		{
			name: "behind only",
			in:   "## main...origin/main [behind 2]\n",
			want: RepoStatus{Branch: "main", Upstream: "origin/main", HasUpstream: true, Behind: 2},
		},
		{
			name: "no upstream",
			in:   "## feature\n",
			want: RepoStatus{Branch: "feature"},
		},
		{
			name: "upstream gone",
			in:   "## main...origin/main [gone]\n",
			want: RepoStatus{Branch: "main", Upstream: "origin/main", UpstreamGone: true},
		},
		{
			name: "no commits yet",
			in:   "## No commits yet on main\n?? a.txt\n",
			want: RepoStatus{Branch: "main", NoCommits: true, Untracked: 1},
		},
		{
			name: "detached HEAD",
			in:   "## HEAD (no branch)\n",
			want: RepoStatus{Branch: "HEAD (detached)", Detached: true},
		},
		{
			name: "branch with slash",
			in:   "## feature/login...origin/feature/login\n",
			want: RepoStatus{Branch: "feature/login", Upstream: "origin/feature/login", HasUpstream: true},
		},
		{
			name: "rename counts as changed",
			in:   "## main...origin/main\nR  old.go -> new.go\n",
			want: RepoStatus{Branch: "main", Upstream: "origin/main", HasUpstream: true, Changed: 1},
		},
		{
			name: "all tracked change kinds",
			in:   "## main...origin/main\nM  staged.go\n M unstaged.go\nA  added.go\n D deleted.go\nUU conflict.go\n",
			want: RepoStatus{Branch: "main", Upstream: "origin/main", HasUpstream: true, Changed: 5},
		},
		{
			name: "untracked only",
			in:   "## main...origin/main\n?? one.txt\n?? two.txt\n",
			want: RepoStatus{Branch: "main", Upstream: "origin/main", HasUpstream: true, Untracked: 2},
		},
		{
			name: "ignored lines are not counted",
			in:   "## main...origin/main\n!! build/\n?? real.txt\n",
			want: RepoStatus{Branch: "main", Upstream: "origin/main", HasUpstream: true, Untracked: 1},
		},
		{
			name: "status without a branch header",
			in:   " M a.go\n?? b.go\n",
			want: RepoStatus{Changed: 1, Untracked: 1},
		},
		{
			name: "blank lines are ignored",
			in:   "## main...origin/main\n\n M a.go\n\n",
			want: RepoStatus{Branch: "main", Upstream: "origin/main", HasUpstream: true, Changed: 1},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseStatus(tc.in)
			if got != tc.want {
				t.Errorf("parseStatus() = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestParseStatusDoesNotSetPathOrRemote(t *testing.T) {
	got := parseStatus("## main...origin/main\n")
	if got.Path != "" {
		t.Errorf("Path = %q, want empty", got.Path)
	}
	if got.NoRemote {
		t.Error("NoRemote = true, want false")
	}
	if got.Err != nil {
		t.Errorf("Err = %v, want nil", got.Err)
	}
}

func TestGitEnvOverrides(t *testing.T) {
	t.Setenv("GIT_OPTIONAL_LOCKS", "1")
	t.Setenv("GIT_TERMINAL_PROMPT", "1")
	t.Setenv("LC_ALL", "en_US.UTF-8")

	want := map[string]string{
		"GIT_OPTIONAL_LOCKS":  "0",
		"GIT_TERMINAL_PROMPT": "0",
		"LC_ALL":              "C",
		"LANG":                "C",
	}

	env := gitEnv()
	counts := make(map[string]int)
	values := make(map[string]string)
	for _, entry := range env {
		key, value, _ := strings.Cut(entry, "=")
		counts[key]++
		values[key] = value
	}

	for key, value := range want {
		if counts[key] != 1 {
			t.Errorf("%s appears %d times, want 1", key, counts[key])
		}
		if values[key] != value {
			t.Errorf("%s = %q, want %q", key, values[key], value)
		}
	}
}
