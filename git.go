package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	aheadRe  = regexp.MustCompile(`ahead (\d+)`)
	behindRe = regexp.MustCompile(`behind (\d+)`)
)

func checkGit() error {
	_, err := exec.LookPath("git")
	return err
}

// runGit executes git in dir. The optional lock and environment settings keep
// gscan read-only and prevent a credential prompt from hanging a scan.
func runGit(ctx context.Context, dir string, timeout time.Duration, args ...string) (string, error) {
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	full := append([]string{"--no-optional-locks"}, args...)
	cmd := exec.CommandContext(cmdCtx, "git", full...)
	cmd.Dir = dir
	cmd.Env = gitEnv()

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		if cmdCtx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("git %s timed out after %s", args[0], timeout)
		}
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("%s", msg)
	}
	return stdout.String(), nil
}

// inspectRepo gathers everything gscan reports about one repository. It never
// fetches unless the caller opted in, so the default scan is read-only and
// offline.
func inspectRepo(ctx context.Context, path string, opts options) RepoStatus {
	st := RepoStatus{Path: path}

	var remoteOut string
	var remoteErr error
	remoteLoaded := false
	loadRemotes := func() {
		if remoteLoaded {
			return
		}
		remoteLoaded = true
		out, err := runGit(ctx, path, opts.timeout, "remote")
		remoteOut = strings.TrimSpace(out)
		remoteErr = err
	}

	if opts.fetch {
		loadRemotes()
		if remoteErr == nil && remoteOut != "" {
			if _, err := runGit(ctx, path, opts.timeout, "fetch", "--quiet", "--prune", "--all"); err != nil {
				st.FetchErr = err
			}
		}
	}

	out, err := runGit(ctx, path, opts.timeout, "status", "--porcelain=v1", "--branch")
	if err != nil {
		st.Err = err
		return st
	}

	parsed := parseStatus(out)
	parsed.Path = path
	parsed.FetchErr = st.FetchErr
	st = parsed

	if !st.HasUpstream && !st.NoCommits {
		loadRemotes()
		if remoteErr == nil && remoteOut == "" {
			st.NoRemote = true
		}
	}
	return st
}

func gitEnv() []string {
	overrides := map[string]string{
		"GIT_OPTIONAL_LOCKS":  "0",
		"GIT_TERMINAL_PROMPT": "0",
		"LC_ALL":              "C",
		"LANG":                "C",
	}

	env := os.Environ()
	out := make([]string, 0, len(env)+len(overrides))
	for _, entry := range env {
		key, _, _ := strings.Cut(entry, "=")
		if _, replaced := overrides[key]; replaced {
			continue
		}
		out = append(out, entry)
	}
	for key, value := range overrides {
		out = append(out, key+"="+value)
	}
	return out
}

// parseStatus turns the output of `git status --porcelain=v1 --branch` into a
// RepoStatus. It is a pure function so the tricky branch parsing is easy to
// test without touching a real repository.
func parseStatus(output string) RepoStatus {
	var st RepoStatus

	trimmed := strings.TrimRight(output, "\n")
	if trimmed == "" {
		return st
	}
	lines := strings.Split(trimmed, "\n")

	firstStatusLine := 0
	if strings.HasPrefix(lines[0], "## ") {
		firstStatusLine = 1
		header := strings.TrimSpace(strings.TrimPrefix(lines[0], "## "))

		switch {
		case strings.HasPrefix(header, "No commits yet on "):
			st.Branch = strings.TrimPrefix(header, "No commits yet on ")
			st.NoCommits = true
		case header == "HEAD (no branch)":
			st.Branch = "HEAD (detached)"
			st.Detached = true
		default:
			branch, rest, found := strings.Cut(header, "...")
			st.Branch = branch
			if !found {
				break
			}

			st.HasUpstream = true
			upstream := rest
			if idx := strings.Index(upstream, " ["); idx >= 0 {
				upstream = upstream[:idx]
			}
			st.Upstream = upstream

			if strings.Contains(rest, "[gone]") {
				st.HasUpstream = false
				st.UpstreamGone = true
			}
			if m := aheadRe.FindStringSubmatch(rest); m != nil {
				st.Ahead, _ = strconv.Atoi(m[1])
			}
			if m := behindRe.FindStringSubmatch(rest); m != nil {
				st.Behind, _ = strconv.Atoi(m[1])
			}
		}
	}

	for _, line := range lines[firstStatusLine:] {
		if line == "" {
			continue
		}
		switch {
		case strings.HasPrefix(line, "??"):
			st.Untracked++
		case strings.HasPrefix(line, "!!"):
			// Ignored files are only present with --ignored; never count them.
		default:
			st.Changed++
		}
	}
	return st
}
