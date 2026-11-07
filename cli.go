package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
)

const usageText = `gscan - scan local git repositories for work that needs attention

Usage:
  gscan [options] [path]

Arguments:
  path                 directory to scan (default: home directory)

Options:
  -all                 include clean repositories
  -j N                 number of parallel workers (default: number of CPUs)
  -hidden              include hidden directories
  -json                print a machine-readable JSON report
  -color MODE          auto, always, or never (default: auto)
  -timeout DURATION    timeout for each git command (default: 10s)
  -fetch               fetch remotes before checking (network, mutates .git)
  -version             print version information
  -h, --help           show this help

Exit codes:
  0  scan completed
  1  runtime failure (missing path, git not installed, ...)
  2  invalid usage
  130 interrupted (Ctrl+C)
`

var errHelp = errors.New("help requested")

type usageError struct {
	err error
}

func (e usageError) Error() string { return e.err.Error() }
func (e usageError) Unwrap() error { return e.err }

// valueFlags tells splitArgs which options consume the following argument. It
// exists so gscan can accept flags after the path, unlike the standard flag
// package's default "stop at first positional" behaviour.
var valueFlags = map[string]bool{
	"j":       true,
	"color":   true,
	"timeout": true,
}

func parseArgs(args []string) (options, error) {
	opts := defaultOptions()

	fs := flag.NewFlagSet("gscan", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}

	fs.BoolVar(&opts.all, "all", false, "")
	fs.IntVar(&opts.workers, "j", opts.workers, "")
	fs.BoolVar(&opts.hidden, "hidden", false, "")
	fs.BoolVar(&opts.json, "json", false, "")
	fs.StringVar(&opts.colorMode, "color", opts.colorMode, "")
	fs.DurationVar(&opts.timeout, "timeout", opts.timeout, "")
	fs.BoolVar(&opts.fetch, "fetch", false, "")
	fs.BoolVar(&opts.showVersion, "version", false, "")

	flags, positionals, err := splitArgs(args)
	if err != nil {
		return opts, usageError{err}
	}

	if err := fs.Parse(flags); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return opts, errHelp
		}
		return opts, usageError{err}
	}

	if opts.workers < 1 {
		return opts, usageError{fmt.Errorf("-j must be at least 1")}
	}
	if opts.timeout <= 0 {
		return opts, usageError{fmt.Errorf("-timeout must be greater than zero")}
	}
	opts.colorMode = strings.ToLower(opts.colorMode)
	switch opts.colorMode {
	case colorAuto, colorAlways, colorNever:
	default:
		return opts, usageError{fmt.Errorf("-color must be auto, always, or never")}
	}
	if len(positionals) > 1 {
		return opts, usageError{fmt.Errorf("expected at most one path, got %d", len(positionals))}
	}
	if len(positionals) == 1 {
		opts.root = positionals[0]
	}

	return opts, nil
}

// splitArgs separates flags from positional arguments so `gscan ~/src -all`
// works as well as `gscan -all ~/src`.
func splitArgs(args []string) (flags, positionals []string, err error) {
	for i := 0; i < len(args); i++ {
		arg := args[i]

		if arg == "--" {
			positionals = append(positionals, args[i+1:]...)
			break
		}
		if arg == "-" || !strings.HasPrefix(arg, "-") {
			positionals = append(positionals, arg)
			continue
		}

		flags = append(flags, arg)

		name := strings.TrimLeft(arg, "-")
		if strings.Contains(name, "=") || !valueFlags[name] {
			continue
		}
		if i+1 >= len(args) {
			return nil, nil, fmt.Errorf("flag needs an argument: %s", arg)
		}
		i++
		flags = append(flags, args[i])
	}
	return flags, positionals, nil
}

func resolveRoot(arg string) (string, error) {
	root := arg

	if arg == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		root = home
	}

	abs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}

	info, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s: not a directory", abs)
	}
	return abs, nil
}

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func printVersion(w io.Writer) {
	fmt.Fprintf(w, "gscan %s\n", version)
	fmt.Fprintf(w, "  commit: %s\n", commit)
	fmt.Fprintf(w, "  built:  %s\n", date)
	fmt.Fprintf(w, "  go:     %s %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)

	if version != "dev" {
		return
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		fmt.Fprintf(w, "  module: %s\n", info.Main.Version)
	}
}
