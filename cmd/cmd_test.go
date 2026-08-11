package cmd_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var bin string

func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "gwt-test")
	if err != nil {
		panic(err)
	}
	bin = filepath.Join(tmp, "gwt")
	build := exec.Command("go", "build", "-o", bin, "..")
	build.Stdout = os.Stderr
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		_, _ = os.Stderr.WriteString("failed to build gwt for tests\n")
		os.Exit(1)
	}
	code := m.Run()
	_ = os.RemoveAll(tmp)
	os.Exit(code)
}

type result struct {
	out  string
	code int
}

// gwt runs the compiled binary in dir with a hermetic tmux shim so no real
// tmux server is ever started or touched during tests.
func gwt(t *testing.T, dir, stdin string, args ...string) result {
	t.Helper()
	shim := filepath.Join(t.TempDir(), "tmux")
	script := "#!/bin/sh\nexit 0\n"
	if err := os.WriteFile(shim, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	cmd.Env = replaceEnvPath(os.Environ(), filepath.Dir(shim))
	out, err := cmd.CombinedOutput()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return result{string(out), ee.ExitCode()}
		}
		t.Fatalf("exec gwt %v: %v", args, err)
	}
	return result{string(out), 0}
}

func replaceEnvPath(env []string, path string) []string {
	var filtered []string
	for _, e := range env {
		if !strings.HasPrefix(e, "PATH=") {
			filtered = append(filtered, e)
		}
	}
	return append(filtered, "PATH="+path+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func gitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func newRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	gitIn(t, dir, "init", "-q", "-b", "main")
	gitIn(t, dir, "config", "user.email", "t@example.com")
	gitIn(t, dir, "config", "user.name", "Test")
	gitIn(t, dir, "config", "commit.gpgsign", "false")
	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, dir, "add", ".")
	gitIn(t, dir, "commit", "-qm", "initial")
	return dir
}

func worktreeExists(repo, name string) bool {
	out, err := exec.Command("git", "-C", repo, "worktree", "list").Output()
	return err == nil && strings.Contains(string(out), "worktree/"+name)
}

func branchExists(repo, name string) bool {
	err := exec.Command("git", "-C", repo, "rev-parse", "--verify", "--quiet",
		"refs/heads/"+name).Run()
	return err == nil
}

func runOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmdArgs := append([]string{"-C", dir}, args...)
	out, err := exec.Command("git", cmdArgs...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

func TestInit(t *testing.T) {
	t.Parallel()
	repo := newRepo(t)
	r := gwt(t, repo, "", "init")
	if r.code != 0 {
		t.Fatalf("init: code=%d out=%s", r.code, r.out)
	}
	if !strings.Contains(r.out, "git-ignored") {
		t.Errorf("init output: %q", r.out)
	}
	data, err := os.ReadFile(filepath.Join(repo, ".git", "info", "exclude"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "worktree/") {
		t.Error("exclude file missing worktree/ entry")
	}
}

func TestAdd(t *testing.T) {
	t.Parallel()
	repo := newRepo(t)

	t.Run("by name from HEAD", func(t *testing.T) {
		r := gwt(t, repo, "", "add", "feature")
		if r.code != 0 {
			t.Fatalf("add feature: code=%d out=%s", r.code, r.out)
		}
		if !worktreeExists(repo, "feature") {
			t.Error("worktree/feature not created")
		}
		if !branchExists(repo, "feature") {
			t.Error("branch feature not created")
		}
	})

	t.Run("with -b override", func(t *testing.T) {
		if r := gwt(t, repo, "", "add", "dir1", "-b", "other"); r.code != 0 {
			t.Fatalf("add -b: %d %s", r.code, r.out)
		}
		out, _ := exec.Command("git", "-C", filepath.Join(repo, "worktree", "dir1"),
			"symbolic-ref", "--short", "HEAD").Output()
		if strings.TrimSpace(string(out)) != "other" {
			t.Errorf("expected branch other, got %q", out)
		}
	})

	t.Run("with -f start point", func(t *testing.T) {
		if r := gwt(t, repo, "", "add", "dir2", "-f", "main"); r.code != 0 {
			t.Fatalf("add -f: %d %s", r.code, r.out)
		}
		if !worktreeExists(repo, "dir2") {
			t.Error("worktree/dir2 not created")
		}
	})

	t.Run("existing worktree connects", func(t *testing.T) {
		if r := gwt(t, repo, "", "add", "feature"); r.code != 0 {
			t.Fatalf("add existing: %d %s", r.code, r.out)
		}
		count := 0
		for _, line := range strings.Split(strings.TrimSpace(runOutput(t, repo, "worktree", "list")), "\n") {
			if strings.Contains(line, "worktree/feature") {
				count++
			}
		}
		if count != 1 {
			t.Errorf("expected 1 worktree/feature, got %d", count)
		}
	})

	t.Run("usage error without name but -b", func(t *testing.T) {
		r := gwt(t, repo, "", "add", "-b", "x")
		if r.code == 0 || !strings.Contains(r.out, "usage:") {
			t.Errorf("expected usage error, code=%d out=%s", r.code, r.out)
		}
	})
}

func TestRm(t *testing.T) {
	t.Parallel()
	repo := newRepo(t)
	for _, n := range []string{"one", "two"} {
		if r := gwt(t, repo, "", "add", n); r.code != 0 {
			t.Fatalf("setup add %s: %d %s", n, r.code, r.out)
		}
	}

	t.Run("removes and deletes merged branch", func(t *testing.T) {
		r := gwt(t, repo, "", "rm", "one")
		if r.code != 0 {
			t.Fatalf("rm one: code=%d out=%s", r.code, r.out)
		}
		if !strings.Contains(r.out, "Removed:") {
			t.Errorf("rm output: %q", r.out)
		}
		if worktreeExists(repo, "one") {
			t.Error("worktree/one still exists")
		}
		if branchExists(repo, "one") {
			t.Error("branch one should be deleted")
		}
	})

	t.Run("dirty blocked without -f", func(t *testing.T) {
		dirty := filepath.Join(repo, "worktree", "two", "dirty.txt")
		if err := os.WriteFile(dirty, []byte("x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		r := gwt(t, repo, "", "rm", "two")
		if r.code == 0 || !strings.Contains(r.out, "uncommitted changes") {
			t.Errorf("expected dirty block, code=%d out=%s", r.code, r.out)
		}
		if !worktreeExists(repo, "two") {
			t.Error("worktree/two should still exist")
		}
		r = gwt(t, repo, "", "rm", "two", "-f")
		if r.code != 0 || !strings.Contains(r.out, "Removed:") {
			t.Errorf("rm -f: code=%d out=%s", r.code, r.out)
		}
	})

	t.Run("dot refuses the main worktree", func(t *testing.T) {
		r := gwt(t, repo, "", "rm", ".")
		if r.code == 0 || !strings.Contains(r.out, "cannot remove the main worktree") {
			t.Errorf("expected main-refusal, code=%d out=%s", r.code, r.out)
		}
	})

	t.Run("unknown name", func(t *testing.T) {
		r := gwt(t, repo, "", "rm", "nosuch")
		if !strings.Contains(r.out, "No worktree found") {
			t.Errorf("rm nosuch: out=%s", r.out)
		}
	})

	t.Run("keeps unmerged branch on n", func(t *testing.T) {
		gitIn(t, repo, "checkout", "-qb", "um")
		if err := os.WriteFile(filepath.Join(repo, "um.txt"), []byte("x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		gitIn(t, repo, "add", "um.txt")
		gitIn(t, repo, "commit", "-qm", "unmerged work")
		gitIn(t, repo, "checkout", "-q", "main")
		gitIn(t, repo, "worktree", "add", "-q", "worktree/um", "um")

		r := gwt(t, repo, "n\n", "rm", "um")
		if r.code != 0 || !strings.Contains(r.out, "Kept branch: um") {
			t.Errorf("expected branch kept, code=%d out=%s", r.code, r.out)
		}
		if !branchExists(repo, "um") {
			t.Error("branch um should have been kept")
		}
	})
}

func TestMv(t *testing.T) {
	t.Parallel()
	repo := newRepo(t)
	if r := gwt(t, repo, "", "add", "feature"); r.code != 0 {
		t.Fatalf("setup: %d %s", r.code, r.out)
	}

	r := gwt(t, repo, "", "mv", "feature", "renamed")
	if r.code != 0 {
		t.Fatalf("mv: code=%d out=%s", r.code, r.out)
	}
	if !strings.Contains(r.out, "Renamed branch:") {
		t.Errorf("mv output: %q", r.out)
	}
	if worktreeExists(repo, "feature") || !worktreeExists(repo, "renamed") {
		t.Error("worktree not moved")
	}
	if branchExists(repo, "feature") || !branchExists(repo, "renamed") {
		t.Error("branch not renamed")
	}

	r = gwt(t, repo, "", "mv", "renamed", "other", "-B")
	if r.code != 0 {
		t.Fatalf("mv -B: %d %s", r.code, r.out)
	}
	if !branchExists(repo, "renamed") {
		t.Error("-B should keep branch renamed")
	}
	if branchExists(repo, "other") {
		t.Error("-B should not create branch other")
	}
	if !worktreeExists(repo, "other") {
		t.Error("worktree/other missing")
	}
}

func TestLs(t *testing.T) {
	t.Parallel()
	repo := newRepo(t)
	if r := gwt(t, repo, "", "add", "feature"); r.code != 0 {
		t.Fatalf("setup: %d %s", r.code, r.out)
	}

	r := gwt(t, repo, "", "ls")
	if r.code != 0 {
		t.Fatalf("ls: %d %s", r.code, r.out)
	}
	for _, want := range []string{"(main)", "feature", "Branch", "Worktree"} {
		if !strings.Contains(r.out, want) {
			t.Errorf("ls output missing %q:\n%s", want, r.out)
		}
	}
}

func TestPick(t *testing.T) {
	t.Parallel()
	repo := newRepo(t)
	if r := gwt(t, repo, "", "add", "feature"); r.code != 0 {
		t.Fatalf("setup: %d %s", r.code, r.out)
	}

	if r := gwt(t, repo, "", "pick", "feature"); r.code != 0 {
		t.Fatalf("pick feature: %d %s", r.code, r.out)
	}
	if r := gwt(t, repo, "", "pick", "nosuch"); r.code == 0 {
		t.Error("pick nosuch should fail")
	}
}
