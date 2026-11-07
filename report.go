package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

type palette struct {
	reset   string
	red     string
	yellow  string
	green   string
	cyan    string
	blue    string
	magenta string
	dim     string
}

func newPalette(enabled bool) palette {
	if !enabled {
		return palette{}
	}
	return palette{
		reset:   "\x1b[0m",
		red:     "\x1b[31m",
		yellow:  "\x1b[33m",
		green:   "\x1b[32m",
		cyan:    "\x1b[36m",
		blue:    "\x1b[34m",
		magenta: "\x1b[35m",
		dim:     "\x1b[2m",
	}
}

func (p palette) paint(code, text string) string {
	if code == "" {
		return text
	}
	return code + text + p.reset
}

func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func colorEnabled(mode string, w io.Writer) bool {
	switch mode {
	case colorAlways:
		return true
	case colorNever:
		return false
	}

	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if os.Getenv("CLICOLOR") == "0" {
		return false
	}
	if os.Getenv("TERM") == "dumb" {
		return false
	}
	if os.Getenv("FORCE_COLOR") != "" {
		return true
	}
	return isTerminal(w)
}

func needsAttention(st RepoStatus) bool {
	if st.Err != nil {
		return true
	}
	if st.Detached {
		return true
	}
	if st.Changed > 0 || st.Untracked > 0 || st.Ahead > 0 {
		return true
	}
	if st.NoRemote {
		return true
	}
	if !st.HasUpstream && !st.NoCommits {
		return true
	}
	return false
}

func summarize(st RepoStatus) []string {
	if st.Err != nil {
		return []string{"error: " + shortError(st.Err)}
	}

	parts := make([]string, 0, 5)
	if st.Changed > 0 {
		parts = append(parts, fmt.Sprintf("%d changed", st.Changed))
	}
	if st.Untracked > 0 {
		parts = append(parts, fmt.Sprintf("%d untracked", st.Untracked))
	}
	if st.Ahead > 0 {
		parts = append(parts, fmt.Sprintf("ahead %d", st.Ahead))
	}
	if st.Behind > 0 {
		parts = append(parts, fmt.Sprintf("behind %d", st.Behind))
	}
	if st.NoCommits {
		parts = append(parts, "no commits")
	}

	switch {
	case st.Detached:
		parts = append(parts, "detached")
	case st.UpstreamGone:
		parts = append(parts, "upstream gone")
	case st.NoRemote:
		parts = append(parts, "no remote")
	case !st.HasUpstream && !st.NoCommits:
		parts = append(parts, "no upstream")
	}

	if st.FetchErr != nil {
		parts = append(parts, "fetch failed")
	}
	if len(parts) == 0 {
		parts = append(parts, "clean")
	}
	return parts
}

func shortError(err error) string {
	msg := strings.TrimSpace(err.Error())
	if idx := strings.IndexByte(msg, '\n'); idx >= 0 {
		msg = msg[:idx]
	}
	return msg
}

func shortPath(path, home string) string {
	if home == "" {
		return path
	}
	if path == home {
		return "~"
	}
	prefix := home + string(os.PathSeparator)
	if strings.HasPrefix(path, prefix) {
		return "~" + strings.TrimPrefix(path, home)
	}
	return path
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

func renderStatus(pal palette, st RepoStatus) string {
	parts := summarize(st)
	colored := make([]string, len(parts))
	for i, part := range parts {
		switch {
		case strings.HasPrefix(part, "error:"):
			colored[i] = pal.paint(pal.red, part)
		case part == "clean":
			colored[i] = pal.paint(pal.green, part)
		case strings.HasPrefix(part, "ahead"):
			colored[i] = pal.paint(pal.cyan, part)
		case strings.HasPrefix(part, "behind"):
			colored[i] = pal.paint(pal.blue, part)
		case part == "fetch failed":
			colored[i] = pal.paint(pal.dim, part)
		case part == "no remote", part == "no upstream", part == "upstream gone", part == "detached", part == "no commits":
			colored[i] = pal.paint(pal.magenta, part)
		default:
			colored[i] = pal.paint(pal.yellow, part)
		}
	}
	return strings.Join(colored, ", ")
}

func printReport(w io.Writer, results []RepoStatus, opts options, home string, elapsed time.Duration) {
	rows := append([]RepoStatus(nil), results...)
	sort.Slice(rows, func(i, j int) bool { return rows[i].Path < rows[j].Path })

	shown := make([]RepoStatus, 0, len(rows))
	for _, st := range rows {
		if opts.all || needsAttention(st) {
			shown = append(shown, st)
		}
	}

	pal := newPalette(colorEnabled(opts.colorMode, w))

	if len(shown) > 0 {
		pathWidth := 0
		branchWidth := 0
		for _, st := range shown {
			if n := utf8.RuneCountInString(shortPath(st.Path, home)); n > pathWidth {
				pathWidth = n
			}
			if n := utf8.RuneCountInString(st.Branch); n > branchWidth {
				branchWidth = n
			}
		}

		for _, st := range shown {
			path := shortPath(st.Path, home)
			branch := st.Branch
			if branch == "" {
				branch = "-"
			}

			pathCell := fmt.Sprintf("%-*s", pathWidth, path)
			branchCell := fmt.Sprintf("%-*s", branchWidth, branch)

			switch {
			case st.Err != nil:
				pathCell = pal.paint(pal.red, pathCell)
			case needsAttention(st):
				pathCell = pal.paint(pal.yellow, pathCell)
			default:
				pathCell = pal.paint(pal.green, pathCell)
			}
			branchCell = pal.paint(pal.dim, branchCell)

			fmt.Fprintf(w, "%s  %s  %s\n", pathCell, branchCell, renderStatus(pal, st))
		}
	} else if len(rows) > 0 {
		if countBehind(rows) > 0 {
			fmt.Fprintln(w, "No local work pending, but some repositories are behind their remote.")
		} else {
			fmt.Fprintln(w, "Everything is clean and pushed.")
		}
	}

	fmt.Fprintln(w)
	printSummary(w, rows, elapsed, pal)
}

func countBehind(results []RepoStatus) int {
	n := 0
	for _, st := range results {
		if st.Behind > 0 {
			n++
		}
	}
	return n
}

func printSummary(w io.Writer, results []RepoStatus, elapsed time.Duration, pal palette) {
	total := len(results)
	attention := 0
	clean := 0
	errors := 0
	dirty := 0
	unpushed := 0
	behind := 0

	for _, st := range results {
		if st.Err != nil {
			errors++
		}
		if needsAttention(st) {
			attention++
		} else {
			clean++
		}
		if st.Changed > 0 || st.Untracked > 0 {
			dirty++
		}
		if st.Ahead > 0 {
			unpushed++
		}
		if st.Behind > 0 {
			behind++
		}
	}

	repoWord := "repositories"
	if total == 1 {
		repoWord = "repository"
	}
	summary := fmt.Sprintf("Scanned %d %s in %s", total, repoWord, formatDuration(elapsed))

	switch {
	case attention > 0:
		attentionWord := "need"
		if attention == 1 {
			attentionWord = "needs"
		}
		summary += fmt.Sprintf(" - %d %s attention", attention, attentionWord)

		details := make([]string, 0, 3)
		if dirty > 0 {
			details = append(details, fmt.Sprintf("%d dirty", dirty))
		}
		if unpushed > 0 {
			details = append(details, fmt.Sprintf("%d unpushed", unpushed))
		}
		if errors > 0 {
			errorWord := "errors"
			if errors == 1 {
				errorWord = "error"
			}
			details = append(details, fmt.Sprintf("%d %s", errors, errorWord))
		}
		if len(details) > 0 {
			summary += " (" + strings.Join(details, ", ") + ")"
		}
		if clean > 0 {
			summary += fmt.Sprintf(", %d clean", clean)
		}
	case behind > 0:
		summary += " - no local work pending"
	default:
		summary += " - everything is clean and pushed"
	}

	if behind > 0 {
		summary += fmt.Sprintf(", %d behind remote", behind)
	}
	fmt.Fprintln(w, pal.paint(pal.dim, summary))
}
