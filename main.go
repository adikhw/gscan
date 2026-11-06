package main

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	exitOK    = 0
	exitFail  = 1
	exitUsage = 2
)

var skipDirs = map[string]bool{
	"node_modules": true,
	"venv":         true,
	"__pycache__":  true,
	"target":       true,
}

type RepoStatus struct {
	Path        string
	Branch      string
	HasUpstream bool
	NoCommits   bool
	NoRemote    bool
	Ahead       int
	Behind      int
	Changed     int
	Untracked   int
	Err         error
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
	root = abs

	info, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s: not a directory", root)
	}
	return root, nil
}

func main() {
	fmt.Println("Hello, Go")
}
