package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
}

func gitRun(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test",
		"GIT_COMMITTER_EMAIL=test@example.com",
		"GIT_CONFIG_NOSYSTEM=1",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	gitRun(t, dir, "init", "-b", "main")
	return dir
}

func commitFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	gitRun(t, dir, "add", name)
	gitRun(t, dir, "commit", "-m", "add "+name)
}

func testOptions() options {
	return options{timeout: 5 * time.Second}
}

func TestInspectRepoClean(t *testing.T) {
	requireGit(t)
	dir := initTestRepo(t)
	commitFile(t, dir, "main.go", "package main\n")

	st := inspectRepo(context.Background(), dir, testOptions())
	if st.Err != nil {
		t.Fatalf("inspectRepo() error = %v", st.Err)
	}
	if st.Path != dir {
		t.Errorf("Path = %q, want %q", st.Path, dir)
	}
	if st.Branch != "main" {
		t.Errorf("Branch = %q, want main", st.Branch)
	}
	if st.HasUpstream {
		t.Error("HasUpstream = true, want false")
	}
	if !st.NoRemote {
		t.Error("NoRemote = false, want true")
	}
	if st.Changed != 0 || st.Untracked != 0 {
		t.Errorf("Changed/Untracked = %d/%d, want 0/0", st.Changed, st.Untracked)
	}
}

func TestInspectRepoCountsWorkingTree(t *testing.T) {
	requireGit(t)
	dir := initTestRepo(t)
	commitFile(t, dir, "main.go", "package main\n")

	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main // changed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("todo\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	st := inspectRepo(context.Background(), dir, testOptions())
	if st.Err != nil {
		t.Fatalf("inspectRepo() error = %v", st.Err)
	}
	if st.Changed != 1 {
		t.Errorf("Changed = %d, want 1", st.Changed)
	}
	if st.Untracked != 1 {
		t.Errorf("Untracked = %d, want 1", st.Untracked)
	}
}

func TestInspectRepoNoCommits(t *testing.T) {
	requireGit(t)
	dir := initTestRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "todo.txt"), []byte("todo\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	st := inspectRepo(context.Background(), dir, testOptions())
	if st.Err != nil {
		t.Fatalf("inspectRepo() error = %v", st.Err)
	}
	if !st.NoCommits {
		t.Error("NoCommits = false, want true")
	}
	if st.Branch != "main" {
		t.Errorf("Branch = %q, want main", st.Branch)
	}
	if st.Untracked != 1 {
		t.Errorf("Untracked = %d, want 1", st.Untracked)
	}
}

func TestInspectRepoBroken(t *testing.T) {
	requireGit(t)
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	st := inspectRepo(context.Background(), dir, testOptions())
	if st.Err == nil {
		t.Fatal("inspectRepo() error = nil, want an error")
	}
}

func TestInspectRepoIsReadOnly(t *testing.T) {
	requireGit(t)
	dir := initTestRepo(t)
	commitFile(t, dir, "main.go", "package main\n")

	before := gitRun(t, dir, "--no-optional-locks", "status", "--porcelain")
	st := inspectRepo(context.Background(), dir, testOptions())
	if st.Err != nil {
		t.Fatalf("inspectRepo() error = %v", st.Err)
	}
	after := gitRun(t, dir, "--no-optional-locks", "status", "--porcelain")

	if before != after {
		t.Errorf("working tree changed during inspect: before %q, after %q", before, after)
	}
	if _, err := os.Stat(filepath.Join(dir, ".git", "index.lock")); !os.IsNotExist(err) {
		t.Errorf("index.lock left behind: %v", err)
	}
}

func TestInspectRepoBehindRemote(t *testing.T) {
	requireGit(t)
	base := t.TempDir()
	remote := filepath.Join(base, "remote.git")
	gitRun(t, base, "init", "--bare", "-b", "main", remote)

	work := filepath.Join(base, "work")
	gitRun(t, base, "clone", remote, work)
	commitFile(t, work, "main.go", "one\n")
	gitRun(t, work, "push", "-u", "origin", "main")

	other := filepath.Join(base, "other")
	gitRun(t, base, "clone", remote, other)
	commitFile(t, other, "main.go", "two\n")
	gitRun(t, other, "push")

	// gscan never fetches by default; make the local view current explicitly.
	gitRun(t, work, "fetch", "--quiet")

	st := inspectRepo(context.Background(), work, testOptions())
	if st.Err != nil {
		t.Fatalf("inspectRepo() error = %v", st.Err)
	}
	if !st.HasUpstream {
		t.Error("HasUpstream = false, want true")
	}
	if st.Behind != 1 {
		t.Errorf("Behind = %d, want 1", st.Behind)
	}
	if st.Ahead != 0 {
		t.Errorf("Ahead = %d, want 0", st.Ahead)
	}
	if st.NoRemote {
		t.Error("NoRemote = true, want false")
	}
}

func TestScanAllDeterministic(t *testing.T) {
	requireGit(t)

	clean := initTestRepo(t)
	commitFile(t, clean, "main.go", "package main\n")

	dirty := initTestRepo(t)
	commitFile(t, dirty, "main.go", "package main\n")
	if err := os.WriteFile(filepath.Join(dirty, "notes.txt"), []byte("todo\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	bare := initTestRepo(t)

	// Deliberately unsorted: scanAll must preserve the caller's order.
	paths := []string{dirty, bare, clean}

	one := scanAll(context.Background(), paths, 1, testOptions())
	many := scanAll(context.Background(), paths, 8, testOptions())

	if len(one) != len(many) {
		t.Fatalf("scanAll lengths differ: %d vs %d", len(one), len(many))
	}
	for i := range one {
		if one[i] != many[i] {
			t.Errorf("result %d differs: -j 1 = %+v, -j 8 = %+v", i, one[i], many[i])
		}
		if one[i].Path != paths[i] {
			t.Errorf("result %d path = %q, want %q", i, one[i].Path, paths[i])
		}
	}
}
