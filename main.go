package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
)

func run() int {
	j := flag.Int("j", runtime.NumCPU(), "Number of workers")
	flag.Parse()
	args := flag.Args()

	if len(args) > 1 || *j < 1 {
		fmt.Fprintln(os.Stderr, "usage: gscan [-j N] [folder]")
		return exitUsage
	}
	return exitOK
}

func main() {
	os.Exit(run())
}
