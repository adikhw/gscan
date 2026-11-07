package main

import (
	"errors"
	"fmt"
	"io"
	"os"
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
	return exitOK
}
