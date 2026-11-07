package main

import (
	"runtime"
	"time"
)

const (
	exitOK     = 0
	exitFail   = 1
	exitUsage  = 2
	exitCancel = 130
)

const (
	colorAuto   = "auto"
	colorAlways = "always"
	colorNever  = "never"

	defaultTimeout = 10 * time.Second
)

// skipDirs contains directory names that are never worth walking. Everything
// here tends to be a build artifact, a package cache, or a dependency tree, so
// descending into it only makes a home-directory scan slower.
var skipDirs = map[string]bool{
	"node_modules":     true,
	"bower_components": true,
	"vendor":           true,
	"Pods":             true,
	"venv":             true,
	".venv":            true,
	"env":              true,
	"__pycache__":      true,
	".tox":             true,
	".mypy_cache":      true,
	".pytest_cache":    true,
	"target":           true,
	"dist":             true,
	"build":            true,
	"out":              true,
	".cache":           true,
	".cargo":           true,
	".rustup":          true,
	".npm":             true,
	".pnpm-store":      true,
	".yarn":            true,
	".gradle":          true,
	".m2":              true,
	".nix-defexpr":     true,
	".next":            true,
	".nuxt":            true,
	".turbo":           true,
	".parcel-cache":    true,
	".svelte-kit":      true,
}

// RepoStatus is the result of inspecting one repository. It deliberately keeps
// raw counts instead of a pre-rendered string so both the text and JSON reports
// can decide how much detail to show.
type RepoStatus struct {
	Path         string
	Branch       string
	Upstream     string
	HasUpstream  bool
	UpstreamGone bool
	Detached     bool
	NoCommits    bool
	NoRemote     bool
	Ahead        int
	Behind       int
	Changed      int
	Untracked    int
	FetchErr     error
	Err          error
}

// options holds the parsed command-line configuration. It is passed by value to
// workers so a scan cannot accidentally mutate shared state.
type options struct {
	all         bool
	workers     int
	hidden      bool
	json        bool
	colorMode   string
	timeout     time.Duration
	fetch       bool
	showVersion bool
	root        string
}

func defaultOptions() options {
	return options{
		workers:   runtime.NumCPU(),
		colorMode: colorAuto,
		timeout:   defaultTimeout,
	}
}
