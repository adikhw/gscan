package main

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func mkdirs(t *testing.T, root string, parts ...string) string {
	t.Helper()
	dir := filepath.Join(append([]string{root}, parts...)...)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func touch(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestFindRepos(t *testing.T) {
	root := t.TempDir()

	repoA := mkdirs(t, root, "repoA")
	mkdirs(t, repoA, ".git")
	mkdirs(t, repoA, "nested")
	mkdirs(t, repoA, "nested", ".git")

	repoB := mkdirs(t, root, "repoB")
	touch(t, filepath.Join(repoB, ".git"))

	hidden := mkdirs(t, root, ".hidden")
	mkdirs(t, hidden, ".git")

	cache := mkdirs(t, root, ".cache")
	mkdirs(t, cache, ".git")

	modules := mkdirs(t, root, "node_modules")
	mkdirs(t, modules, ".git")

	mkdirs(t, root, "plain")

	got, err := findRepos(root, false)
	if err != nil {
		t.Fatalf("findRepos() error = %v", err)
	}
	want := []string{repoA, repoB}
	sort.Strings(want)
	if len(got) != len(want) {
		t.Fatalf("findRepos() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("findRepos()[%d] = %q, want %q", i, got[i], want[i])
		}
	}

	got, err = findRepos(root, true)
	if err != nil {
		t.Fatalf("findRepos(hidden) error = %v", err)
	}
	want = []string{hidden, repoA, repoB}
	sort.Strings(want)
	if len(got) != len(want) {
		t.Fatalf("findRepos(hidden) = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("findRepos(hidden)[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestFindReposRootIsRepo(t *testing.T) {
	root := t.TempDir()
	mkdirs(t, root, ".git")
	mkdirs(t, root, "child", ".git")

	got, err := findRepos(root, false)
	if err != nil {
		t.Fatalf("findRepos() error = %v", err)
	}
	if len(got) != 1 || got[0] != root {
		t.Fatalf("findRepos() = %v, want [%q]", got, root)
	}
}

func TestFindReposDoesNotFollowSymlinks(t *testing.T) {
	root := t.TempDir()
	target := mkdirs(t, root, "repo")
	mkdirs(t, target, ".git")

	linkParent := mkdirs(t, root, "links")
	if err := os.Symlink(target, filepath.Join(linkParent, "loop")); err != nil {
		t.Skipf("symlinks not supported: %v", err)
	}

	got, err := findRepos(root, false)
	if err != nil {
		t.Fatalf("findRepos() error = %v", err)
	}
	if len(got) != 1 || got[0] != target {
		t.Fatalf("findRepos() = %v, want [%q]", got, target)
	}
}

func TestFindReposSkipsUnreadableDirectory(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can read any directory")
	}

	root := t.TempDir()
	repo := mkdirs(t, root, "repo")
	mkdirs(t, repo, ".git")

	locked := mkdirs(t, root, "locked")
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })

	got, err := findRepos(root, false)
	if err != nil {
		t.Fatalf("findRepos() error = %v", err)
	}
	if len(got) != 1 || got[0] != repo {
		t.Fatalf("findRepos() = %v, want [%q]", got, repo)
	}
}

func TestScanAllEmpty(t *testing.T) {
	got := scanAll(context.Background(), nil, 4, testOptions())
	if len(got) != 0 {
		t.Fatalf("scanAll() = %v, want empty", got)
	}
}
