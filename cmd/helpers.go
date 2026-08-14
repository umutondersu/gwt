package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ktr0731/go-fuzzyfinder"
	"github.com/mattn/go-runewidth"
	"golang.org/x/term"

	"github.com/umutondersu/gwt/core"
)

const (
	colorCyan    = "\x1b[36m"
	colorYellow  = "\x1b[33m"
	colorBrBlack = "\x1b[90m"
	colorBold    = "\x1b[1m"
	colorReset   = "\x1b[0m"
)

func splitLines(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

func isTerminal(f *os.File) bool {
	return term.IsTerminal(int(f.Fd()))
}

func padRight(s string, width int) string {
	if runewidth.StringWidth(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-runewidth.StringWidth(s))
}

func containsPath(paths []string, p string) bool {
	for _, x := range paths {
		if x == p {
			return true
		}
	}
	return false
}

func pruneEmptyParents(path string) {
	parent := filepath.Dir(path)
	for parent != filepath.Dir(parent) {
		entries, err := os.ReadDir(parent)
		if err != nil || len(entries) != 0 {
			break
		}
		_ = os.Remove(parent)
		parent = filepath.Dir(parent)
	}
}

func prompt(message string) (string, error) {
	fmt.Fprint(os.Stderr, message)
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && line == "" {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func pickBranch(refs []string) (string, error) {
	if !isTerminal(os.Stdin) {
		return "", fmt.Errorf("interactive selection requires a terminal; pass a name explicitly")
	}
	idx, err := fuzzyfinder.Find(
		refs,
		func(i int) string { return strings.TrimPrefix(refs[i], "origin/") },
		fuzzyfinder.WithPromptString("Create worktree from branch> "),
		fuzzyfinder.WithHeader("Enter to create worktree"),
		fuzzyfinder.WithPreviewWindow(func(i, w, h int) string {
			return core.GitOutput("log", "--oneline", "--decorate", "--graph", "--color=always", "-20", refs[i])
		}),
	)
	if err != nil {
		return "", err
	}
	return strings.TrimPrefix(refs[idx], "origin/"), nil
}

func pickWorktree(names, paths []string) (string, error) {
	if !isTerminal(os.Stdin) {
		return "", fmt.Errorf("interactive selection requires a terminal; pass a name explicitly")
	}
	idx, err := fuzzyfinder.Find(
		paths,
		func(i int) string { return names[i] },
		fuzzyfinder.WithPromptString("Switch to worktree> "),
		fuzzyfinder.WithHeader("Enter to connect"),
		fuzzyfinder.WithPreviewWindow(func(i, w, h int) string {
			return core.GitOutputIn(paths[i], "log", "--oneline", "--decorate", "--graph", "--color=always", "-20")
		}),
	)
	if err != nil {
		return "", err
	}
	return paths[idx], nil
}

func pickWorktreesToRemove(names, paths []string) ([]string, error) {
	if !isTerminal(os.Stdin) {
		return nil, fmt.Errorf("interactive selection requires a terminal; pass names explicitly")
	}
	indices, err := fuzzyfinder.FindMulti(
		paths,
		func(i int) string { return names[i] },
		fuzzyfinder.WithPromptString("Remove worktrees> "),
		fuzzyfinder.WithHeader("Tab to multi-select, Enter to confirm"),
		fuzzyfinder.WithPreviewWindow(func(i, w, h int) string {
			return worktreePreview(paths[i])
		}),
	)
	if err != nil {
		return nil, err
	}
	var selected []string
	for _, i := range indices {
		selected = append(selected, paths[i])
	}
	return selected, nil
}

func worktreePreview(path string) string {
	var b strings.Builder
	if core.IsDirty(path) {
		b.WriteString("--- Uncommitted changes ---\n")
		b.WriteString(core.GitOutputIn(path, "status", "--short"))
		b.WriteString("\n--- Log ---\n")
	}
	b.WriteString(core.GitOutputIn(path, "log", "--oneline", "--decorate", "--graph", "--color=always", "-20"))
	return b.String()
}
