package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunMissingPath(t *testing.T) {
	var stdout, stderr bytes.Buffer
	missing := filepath.Join(t.TempDir(), "does-not-exist")

	if code := run([]string{missing}, &stdout, &stderr); code != exitFail {
		t.Fatalf("run() = %d, want %d", code, exitFail)
	}
	if !strings.Contains(stderr.String(), "gscan:") {
		t.Errorf("stderr = %q, want gscan prefix", stderr.String())
	}
}

func TestRunPathIsFile(t *testing.T) {
	var stdout, stderr bytes.Buffer
	file := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	if code := run([]string{file}, &stdout, &stderr); code != exitFail {
		t.Fatalf("run() = %d, want %d", code, exitFail)
	}
	if !strings.Contains(stderr.String(), "not a directory") {
		t.Errorf("stderr = %q, want not a directory", stderr.String())
	}
}

func TestRunNoRepositories(t *testing.T) {
	var stdout, stderr bytes.Buffer
	dir := t.TempDir()

	if code := run([]string{"-color", colorNever, dir}, &stdout, &stderr); code != exitOK {
		t.Fatalf("run() = %d, want %d; stderr=%s", code, exitOK, stderr.String())
	}
	if !strings.Contains(stdout.String(), "No git repositories found in "+dir) {
		t.Errorf("stdout = %q, want no repositories message", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Try -hidden") {
		t.Errorf("stdout = %q, want hidden hint", stdout.String())
	}
}

func TestRunNoRepositoriesJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	dir := t.TempDir()

	if code := run([]string{"-json", dir}, &stdout, &stderr); code != exitOK {
		t.Fatalf("run() = %d, want %d; stderr=%s", code, exitOK, stderr.String())
	}

	var report jsonReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout.String())
	}
	if report.Root != dir {
		t.Errorf("Root = %q, want %q", report.Root, dir)
	}
	if report.Summary.Total != 0 {
		t.Errorf("Summary.Total = %d, want 0", report.Summary.Total)
	}
}

func TestRunScansRepository(t *testing.T) {
	requireGit(t)
	dir := initTestRepo(t)
	commitFile(t, dir, "main.go", "package main\n")

	var stdout, stderr bytes.Buffer
	if code := run([]string{"-color", colorNever, dir}, &stdout, &stderr); code != exitOK {
		t.Fatalf("run() = %d, want %d; stderr=%s", code, exitOK, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, dir) {
		t.Errorf("stdout missing repo path:\n%s", out)
	}
	if !strings.Contains(out, "main") || !strings.Contains(out, "no remote") {
		t.Errorf("stdout missing status:\n%s", out)
	}
	if !strings.Contains(out, "Scanned 1 repository") {
		t.Errorf("stdout missing summary:\n%s", out)
	}
}

func TestRunScansRepositoryJSON(t *testing.T) {
	requireGit(t)
	dir := initTestRepo(t)
	commitFile(t, dir, "main.go", "package main\n")

	var stdout, stderr bytes.Buffer
	if code := run([]string{"-json", dir}, &stdout, &stderr); code != exitOK {
		t.Fatalf("run() = %d, want %d; stderr=%s", code, exitOK, stderr.String())
	}

	var report jsonReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout.String())
	}
	if report.Summary.Total != 1 {
		t.Errorf("Summary.Total = %d, want 1", report.Summary.Total)
	}
	if len(report.Repositories) != 1 || report.Repositories[0].Path != dir {
		t.Errorf("Repositories = %+v, want one repo at %s", report.Repositories, dir)
	}
}

func TestRunGitMissing(t *testing.T) {
	t.Setenv("PATH", "")

	var stdout, stderr bytes.Buffer
	if code := run([]string{"-color", colorNever, t.TempDir()}, &stdout, &stderr); code != exitFail {
		t.Fatalf("run() = %d, want %d", code, exitFail)
	}
	if !strings.Contains(stderr.String(), "git not found in PATH") {
		t.Errorf("stderr = %q, want git not found", stderr.String())
	}
}
