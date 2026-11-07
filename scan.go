package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// findRepos walks root without following symlinks. A directory containing a
// .git entry (file or directory) is a repository; its contents are skipped so
// nested repos and submodules are not counted twice.
func findRepos(root string, includeHidden bool) ([]string, error) {
	repos := make([]string, 0, 16)

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if d == nil {
				return walkErr
			}
			// A single unreadable directory should not abort the whole scan.
			return nil
		}
		if !d.IsDir() {
			return nil
		}

		name := d.Name()
		if path != root {
			if !includeHidden && strings.HasPrefix(name, ".") {
				return fs.SkipDir
			}
			if skipDirs[name] {
				return fs.SkipDir
			}
		}

		if _, err := os.Stat(filepath.Join(path, ".git")); err == nil {
			repos = append(repos, path)
			return fs.SkipDir
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(repos)
	return repos, nil
}
