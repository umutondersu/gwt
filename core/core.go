package core

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var commonDirRe = regexp.MustCompile(`/\.git(/worktrees/.+)?$`)

func MainRoot() (string, error) {
	common, err := Git("rev-parse", "--git-common-dir")
	if err != nil {
		return "", err
	}
	common = filepath.Clean(strings.TrimSpace(common))
	if !filepath.IsAbs(common) {
		if abs, err := filepath.Abs(common); err == nil {
			common = abs
		}
	}
	if resolved, err := filepath.EvalSymlinks(common); err == nil {
		common = resolved
	}
	return commonDirRe.ReplaceAllString(common, ""), nil
}

func NameToPath(name, mainRoot string) (string, bool) {
	if strings.HasPrefix(name, "/") {
		return name, true
	}
	p := filepath.Join(mainRoot, "worktree", name)
	if IsRegisteredWorktree(p) {
		return p, true
	}
	return "", false
}

func PathToName(path, mainRoot string) string {
	if path == mainRoot {
		return filepath.Base(mainRoot)
	}
	if i := strings.LastIndex(path, "/worktree/"); i >= 0 {
		return path[i+len("/worktree/"):]
	}
	return path
}

func WorktreePaths() ([]string, error) {
	out, err := Git("worktree", "list")
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimLeft(line, " \t")
		if line == "" {
			continue
		}
		var path, rest string
		if idx := strings.IndexByte(line, ' '); idx >= 0 {
			path, rest = line[:idx], line[idx+1:]
		} else {
			path = line
		}
		if !strings.Contains(rest, "(bare)") {
			paths = append(paths, path)
		}
	}
	return paths, nil
}

func IgnoreWorktreeDir(mainRoot string) (err error) {
	exclude := filepath.Join(mainRoot, ".git", "info", "exclude")
	data, readErr := os.ReadFile(exclude)
	if readErr != nil && !os.IsNotExist(readErr) {
		return readErr
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == "worktree/" {
			return nil
		}
	}
	f, err := os.OpenFile(exclude, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()
	if len(data) > 0 && !strings.HasSuffix(string(data), "\n") {
		if _, err := f.WriteString("\n"); err != nil {
			return err
		}
	}
	_, err = f.WriteString("worktree/\n")
	return err
}

func IsRegisteredWorktree(path string) bool {
	set, err := registeredWorktreeSet()
	if err != nil {
		return false
	}
	return set[path]
}

func BranchOfWorktree(path string) string {
	out, err := Git("worktree", "list", "--porcelain")
	if err != nil {
		return ""
	}
	var cur string
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "worktree ") {
			cur = strings.TrimSpace(strings.TrimPrefix(line, "worktree "))
			continue
		}
		if cur == path && strings.HasPrefix(line, "branch ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "branch "))
		}
	}
	return ""
}

func registeredWorktreeSet() (map[string]bool, error) {
	out, err := Git("worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	set := map[string]bool{}
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "worktree ") {
			set[strings.TrimSpace(strings.TrimPrefix(line, "worktree "))] = true
		}
	}
	return set, nil
}
