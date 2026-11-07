package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want options
	}{
		{
			name: "defaults",
			args: nil,
			want: options{workers: runtime.NumCPU(), colorMode: colorAuto, timeout: 10 * time.Second},
		},
		{
			name: "flags before path",
			args: []string{"-all", "-j", "2", "/tmp/src"},
			want: options{all: true, workers: 2, colorMode: colorAuto, timeout: 10 * time.Second, root: "/tmp/src"},
		},
		{
			name: "flags after path",
			args: []string{"/tmp/src", "-all", "-hidden"},
			want: options{all: true, workers: runtime.NumCPU(), hidden: true, colorMode: colorAuto, timeout: 10 * time.Second, root: "/tmp/src"},
		},
		{
			name: "all options",
			args: []string{"-all", "-j=4", "-hidden", "-json", "-color", "always", "-timeout", "2s", "-fetch", "/tmp/src"},
			want: options{
				all:       true,
				workers:   4,
				hidden:    true,
				json:      true,
				colorMode: colorAlways,
				timeout:   2 * time.Second,
				fetch:     true,
				root:      "/tmp/src",
			},
		},
		{
			name: "double dash flags and separator",
			args: []string{"--all", "--", "-weird-path"},
			want: options{all: true, workers: runtime.NumCPU(), colorMode: colorAuto, timeout: 10 * time.Second, root: "-weird-path"},
		},
		{
			name: "version",
			args: []string{"-version"},
			want: options{workers: runtime.NumCPU(), colorMode: colorAuto, timeout: 10 * time.Second, showVersion: true},
		},
		{
			name: "color mode is case insensitive",
			args: []string{"-color", "ALWAYS"},
			want: options{workers: runtime.NumCPU(), colorMode: colorAlways, timeout: 10 * time.Second},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseArgs(tc.args)
			if err != nil {
				t.Fatalf("parseArgs() error = %v", err)
			}
			if got != tc.want {
				t.Errorf("parseArgs() = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestParseArgsHelp(t *testing.T) {
	for _, arg := range []string{"-h", "--help", "-help"} {
		t.Run(arg, func(t *testing.T) {
			_, err := parseArgs([]string{arg})
			if !errors.Is(err, errHelp) {
				t.Fatalf("parseArgs(%q) error = %v, want errHelp", arg, err)
			}
		})
	}
}

func TestParseArgsErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"zero workers", []string{"-j", "0"}},
		{"bad workers", []string{"-j", "many"}},
		{"bad color", []string{"-color", "rainbow"}},
		{"bad timeout", []string{"-timeout", "0s"}},
		{"missing worker value", []string{"-j"}},
		{"missing color value", []string{"-color"}},
		{"unknown flag", []string{"-nope"}},
		{"two paths", []string{"one", "two"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseArgs(tc.args)
			if err == nil {
				t.Fatal("parseArgs() error = nil, want an error")
			}
			var usageErr usageError
			if !errors.As(err, &usageErr) {
				t.Fatalf("parseArgs() error type = %T, want usageError", err)
			}
		})
	}
}

func TestSplitArgs(t *testing.T) {
	flags, positionals, err := splitArgs([]string{"src", "-j", "8", "other", "-all"})
	if err != nil {
		t.Fatalf("splitArgs() error = %v", err)
	}
	if want := []string{"-j", "8", "-all"}; strings.Join(flags, " ") != strings.Join(want, " ") {
		t.Errorf("flags = %v, want %v", flags, want)
	}
	if want := []string{"src", "other"}; strings.Join(positionals, " ") != strings.Join(want, " ") {
		t.Errorf("positionals = %v, want %v", positionals, want)
	}
}

func TestResolveRoot(t *testing.T) {
	dir := t.TempDir()

	got, err := resolveRoot(dir)
	if err != nil {
		t.Fatalf("resolveRoot(%q) error = %v", dir, err)
	}
	if got != dir {
		t.Errorf("resolveRoot(%q) = %q, want %q", dir, got, dir)
	}

	file := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := resolveRoot(file); err == nil || !strings.Contains(err.Error(), "not a directory") {
		t.Errorf("resolveRoot(file) error = %v, want not a directory", err)
	}

	if _, err := resolveRoot(filepath.Join(dir, "missing")); err == nil {
		t.Error("resolveRoot(missing) error = nil, want an error")
	}
}

func TestRunHelpAndVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer

	if code := run([]string{"-h"}, &stdout, &stderr); code != exitOK {
		t.Fatalf("run(-h) = %d, want %d", code, exitOK)
	}
	if !strings.Contains(stdout.String(), "Usage:") {
		t.Errorf("help output missing Usage: %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"-version"}, &stdout, &stderr); code != exitOK {
		t.Fatalf("run(-version) = %d, want %d", code, exitOK)
	}
	if !strings.Contains(stdout.String(), "gscan ") {
		t.Errorf("version output = %q, want gscan version", stdout.String())
	}
}

func TestRunUsageError(t *testing.T) {
	var stdout, stderr bytes.Buffer

	if code := run([]string{"-j", "0"}, &stdout, &stderr); code != exitUsage {
		t.Fatalf("run(-j 0) = %d, want %d", code, exitUsage)
	}
	if !strings.Contains(stderr.String(), "gscan:") {
		t.Errorf("stderr = %q, want gscan prefix", stderr.String())
	}
}

func TestResolveRootDefaultsToHome(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	got, err := resolveRoot("")
	if err != nil {
		t.Fatalf("resolveRoot(\"\") error = %v", err)
	}
	if got != dir {
		t.Errorf("resolveRoot(\"\") = %q, want %q", got, dir)
	}
}
