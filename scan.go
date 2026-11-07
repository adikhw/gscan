package main

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
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
		if path != root && !includeHidden && strings.HasPrefix(name, ".") {
			return fs.SkipDir
		}
		if path != root && skipDirs[name] {
			return fs.SkipDir
		}

		if _, err := os.Stat(filepath.Join(path, ".git")); err == nil {
			repos = append(repos, path)
			return fs.SkipDir
		}

		// A project directory named "target" or "build" is still a repo, but
		// if it is not, it is almost certainly a build artifact.
		if path != root && softSkipDirs[name] {
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

// scanAll inspects repositories with a bounded worker pool. Each worker writes
// to its own index in results, so no mutex is needed. The returned slice keeps
// the same order as paths regardless of which worker finishes first.
func scanAll(ctx context.Context, paths []string, workers int, opts options) []RepoStatus {
	results := make([]RepoStatus, len(paths))
	if len(paths) == 0 {
		return results
	}
	if workers < 1 {
		workers = 1
	}
	if workers > len(paths) {
		workers = len(paths)
	}

	jobs := make(chan int)
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobs {
				results[idx] = inspectRepo(ctx, paths[idx], opts)
			}
		}()
	}

	for idx := range paths {
		select {
		case jobs <- idx:
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return results
		}
	}
	close(jobs)
	wg.Wait()
	return results
}
