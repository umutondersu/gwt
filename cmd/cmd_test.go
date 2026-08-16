package cmd_test

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
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

// processAlive reports whether a process with the given pid still exists.
func processAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

func TestRmClosesRunningProcesses(t *testing.T) {
	t.Parallel()
	repo := newRepo(t)
	if r := gwt(t, repo, "", "add", "one"); r.code != 0 {
		t.Fatalf("setup add one: %d %s", r.code, r.out)
	}

	wt, err := filepath.EvalSymlinks(filepath.Join(repo, "worktree", "one"))
	if err != nil {
		t.Fatal(err)
	}
	subDir := filepath.Join(wt, "subdir")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// A tmux shim that reports a session "repo/one" whose pane is running in a
	// subdirectory of the worktree (an ongoing command like nvim or a dev
	// server) and that kills a real background process on kill-session, just
	// like real tmux SIGHUPs the pane's process tree.
	shimDir := t.TempDir()
	pidFile := filepath.Join(shimDir, "pid")
	logFile := filepath.Join(shimDir, "tmux.log")
	shim := filepath.Join(shimDir, "tmux")
	script := fmt.Sprintf(`#!/bin/sh
case "$1" in
  display-message)
    printf 'other\n'
    ;;
  list-panes)
    a=0
    for x in "$@"; do [ "$x" = "-a" ] && a=1; done
    if [ "$a" = 1 ]; then
      printf 'repo/one %s/subdir\n'
    fi
    ;;
  kill-session)
    printf 'kill-session %%s\n' "$*" >>"%s"
    if [ -f "%s" ]; then kill "$(cat "%s")" 2>/dev/null; fi
    ;;
  *) exit 0 ;;
esac
`, wt, logFile, pidFile, pidFile)
	if err := os.WriteFile(shim, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	// Spawn the "ongoing command" with its cwd inside the worktree, as a long
	// running process whose pid the shim will terminate on kill-session.
	srv := exec.Command("sh", "-c", fmt.Sprintf(`echo $$ > %s; exec sleep 300`, pidFile))
	srv.Dir = subDir
	if err := srv.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = srv.Process.Kill() }()

	var pid int
	deadline := time.Now().Add(5 * time.Second)
	for {
		if data, err := os.ReadFile(pidFile); err == nil {
			if _, err := fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &pid); err == nil && pid > 0 {
				break
			}
		}
		if time.Now().After(deadline) {
			t.Fatal("pid file was never written by the emulated command")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if pid == 0 {
		t.Fatal("failed to parse emulated command pid")
	}
	if !processAlive(pid) {
		t.Fatal("emulated command died before rm ran")
	}

	cmd := exec.Command(bin, "rm", "one")
	cmd.Dir = repo
	cmd.Env = replaceEnvPath(os.Environ(), shimDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("rm one with active process: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "Removed:") {
		t.Errorf("rm output: %q", out)
	}
	if worktreeExists(repo, "one") {
		t.Error("worktree/one still exists")
	}

	logData, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(logData), "kill-session") || !strings.Contains(string(logData), "repo/one") {
		t.Errorf("expected kill-session for repo/one, shim log: %q", logData)
	}

	// The ongoing command must have been closed before removal completed.
	done := make(chan error, 1)
	go func() { done <- srv.Wait() }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Error("emulated ongoing command was not closed by rm")
	}
}

func TestRmRespawnsCurrentSessionPanes(t *testing.T) {
	t.Parallel()
	repo := newRepo(t)
	if r := gwt(t, repo, "", "add", "one"); r.code != 0 {
		t.Fatalf("setup add one: %d %s", r.code, r.out)
	}

	mainRoot, err := filepath.EvalSymlinks(repo)
	if err != nil {
		t.Fatal(err)
	}
	wt, err := filepath.EvalSymlinks(filepath.Join(repo, "worktree", "one"))
	if err != nil {
		t.Fatal(err)
	}
	subDir := filepath.Join(wt, "subdir")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Current session "other" is unrelated to the worktree but has a pane
	// running inside it; rm must re-home it to mainRoot, not kill the session.
	shimDir := t.TempDir()
	logFile := filepath.Join(shimDir, "tmux.log")
	shim := filepath.Join(shimDir, "tmux")
	script := fmt.Sprintf(`#!/bin/sh
case "$1" in
  display-message)
    printf 'other\n'
    ;;
  list-panes)
    a=0
    t=0
    for x in "$@"; do
      [ "$x" = "-a" ] && a=1
      case "$x" in -t*) t=1 ;; esac
    done
    if [ "$a" = "1" ]; then
      printf 'other %s/subdir\n'
    elif [ "$t" = "1" ]; then
      printf 'p1 %s/subdir\n'
    fi
    ;;
  respawn-pane)
    printf 'respawn-pane %%s\n' "$*" >>"%s"
    ;;
  kill-session)
    printf 'kill-session %%s\n' "$*" >>"%s"
    ;;
  *) exit 0 ;;
esac
`, wt, wt, logFile, logFile)
	if err := os.WriteFile(shim, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bin, "rm", "one")
	cmd.Dir = repo
	cmd.Env = replaceEnvPath(os.Environ(), shimDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("rm one: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "Removed:") {
		t.Errorf("rm output: %q", out)
	}

	logData, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(logData), "respawn-pane") || !strings.Contains(string(logData), mainRoot) {
		t.Errorf("expected respawn-pane into %s, shim log: %q", mainRoot, logData)
	}
	if strings.Contains(string(logData), "kill-session") {
		t.Errorf("current session must not be killed, shim log: %q", logData)
	}
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

func TestCompletion(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	r := gwt(t, dir, "", "completion", "fish")
	if r.code != 0 {
		t.Fatalf("completion fish: code=%d out=%s", r.code, r.out)
	}
	if !strings.HasPrefix(r.out, "# Fish completions for gwt") {
		t.Errorf("expected handmade fish completion, got %q", r.out)
	}
	if strings.Contains(r.out, "__gwt_debug") {
		t.Error("got cobra-generated fish completion instead of the handmade one")
	}

	r = gwt(t, dir, "", "completion")
	if r.code != 0 {
		t.Fatalf("completion (bare): code=%d out=%s", r.code, r.out)
	}
	if r.out != gwt(t, dir, "", "completion", "fish").out {
		t.Error("bare completion should default to fish output")
	}

	r = gwt(t, dir, "", "completion", "bash")
	if r.code != 0 || !strings.Contains(r.out, "__start_gwt") {
		t.Errorf("completion bash: code=%d", r.code)
	}
}

func TestDependencyBehavior(t *testing.T) {
	t.Parallel()
	repo := newRepo(t)

	t.Run("git missing is reported clearly", func(t *testing.T) {
		t.Parallel()
		shim := filepath.Join(t.TempDir(), "tmux")
		if err := os.WriteFile(shim, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command(bin, "ls")
		cmd.Dir = repo
		cmd.Env = []string{"PATH=" + filepath.Dir(shim)}
		out, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatalf("expected failure with git missing, got %s", out)
		}
		if !strings.Contains(string(out), "git is required") {
			t.Errorf("expected git-required error, got %q", out)
		}
	})

	t.Run("tmux missing warns but worktree ops succeed", func(t *testing.T) {
		t.Parallel()
		gitPath, err := exec.LookPath("git")
		if err != nil {
			t.Fatal(err)
		}
		gitDir := filepath.Dir(gitPath)
		if _, err := exec.LookPath(filepath.Join(gitDir, "tmux")); err == nil {
			t.Skip("tmux resolves inside git's dir; cannot simulate a missing tmux")
		}
		cmd := exec.Command(bin, "add", "feature")
		cmd.Dir = repo
		cmd.Env = []string{"PATH=" + gitDir}
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("add with missing tmux: %v\n%s", err, out)
		}
		if !strings.Contains(string(out), "tmux not found") {
			t.Errorf("expected tmux warning, got %q", out)
		}
		if !worktreeExists(repo, "feature") {
			t.Error("worktree/feature not created")
		}
	})

	t.Run("interactive picker requires a terminal", func(t *testing.T) {
		t.Parallel()
		cmd := exec.Command(bin, "pick")
		cmd.Dir = repo
		cmd.Stdin = strings.NewReader("")
		cmd.Env = replaceEnvPath(os.Environ(), t.TempDir())
		out, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatalf("expected failure for non-tty pick, got %s", out)
		}
		if !strings.Contains(string(out), "requires a terminal") {
			t.Errorf("expected terminal error, got %q", out)
		}
	})
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
