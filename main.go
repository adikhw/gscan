package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	opts, err := parseArgs(args)
	if errors.Is(err, errHelp) {
		fmt.Fprint(stdout, usageText)
		return exitOK
	}
	if err != nil {
		fmt.Fprintf(stderr, "gscan: %v\n\n%s", err, usageText)
		return exitUsage
	}
	if opts.showVersion {
		printVersion(stdout)
		return exitOK
	}

	if err := checkGit(); err != nil {
		fmt.Fprintln(stderr, "gscan: git not found in PATH")
		return exitFail
	}

	root, err := resolveRoot(opts.root)
	if err != nil {
		fmt.Fprintf(stderr, "gscan: %v\n", err)
		return exitFail
	}

	home, _ := os.UserHomeDir()

	start := time.Now()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	paths, err := findRepos(root, opts.hidden)
	if err != nil {
		fmt.Fprintf(stderr, "gscan: %v\n", err)
		return exitFail
	}

	if len(paths) == 0 {
		if opts.json {
			if err := printJSON(stdout, root, nil, opts, time.Since(start)); err != nil {
				fmt.Fprintf(stderr, "gscan: %v\n", err)
				return exitFail
			}
			return exitOK
		}
		fmt.Fprintf(stdout, "No git repositories found in %s\n", root)
		if !opts.hidden {
			fmt.Fprintln(stdout, "Try -hidden to include hidden directories.")
		}
		return exitOK
	}

	results := scanAll(ctx, paths, opts.workers, opts)
	if ctx.Err() != nil {
		fmt.Fprintln(stderr, "gscan: interrupted")
		return exitCancel
	}

	elapsed := time.Since(start)
	if opts.json {
		if err := printJSON(stdout, root, results, opts, elapsed); err != nil {
			fmt.Fprintf(stderr, "gscan: %v\n", err)
			return exitFail
		}
		return exitOK
	}

	printReport(stdout, results, opts, home, elapsed)
	return exitOK
}
