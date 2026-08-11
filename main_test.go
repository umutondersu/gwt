package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestHelperProcess re-invokes this test binary as the gwt CLI so the real
// main() entrypoint (including cobra routing and os.Exit handling) is tested.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	runAsGWT()
	os.Exit(0)
}

func runAsGWT() {
	idx := -1
	for i, a := range os.Args {
		if a == "--" {
			idx = i
			break
		}
	}
	if idx >= 0 {
		os.Args = append([]string{"gwt"}, os.Args[idx+1:]...)
	}
	main()
	os.Exit(0)
}

func execGWT(t *testing.T, dir string, args ...string) (string, int) {
	t.Helper()
	cmdArgs := append([]string{"-test.run=TestHelperProcess", "--"}, args...)
	cmd := exec.Command(os.Args[0], cmdArgs...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return string(out), ee.ExitCode()
		}
		t.Fatalf("exec gwt: %v", err)
	}
	return string(out), 0
}

func TestCLIExitCodes(t *testing.T) {
	repo := newRepo(t)

	tests := []struct {
		name    string
		dir     string
		args    []string
		code    int
		contain string
	}{
		{"help exits 0", repo, []string{"--help"}, 0, "Usage"},
		{"bare exits 0", repo, nil, 0, "Usage"},
		{"unknown subcommand", repo, []string{"bogus"}, 1, "unknown command"},
		{"ls inside repo", repo, []string{"ls"}, 0, "Worktree"},
		{"outside a repo", t.TempDir(), []string{"ls"}, 1, "not inside a git repository"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out, code := execGWT(t, tc.dir, tc.args...)
			if code != tc.code {
				t.Errorf("exit code = %d, want %d (out: %s)", code, tc.code, out)
			}
			if tc.contain != "" && !strings.Contains(out, tc.contain) {
				t.Errorf("output missing %q:\n%s", tc.contain, out)
			}
		})
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

func gitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}
