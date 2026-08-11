package core_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/umutondersu/gwt/core"
)

var _ = Describe("Path resolution", func() {
	var dir string
	var mainRoot string
	var oldWd string

	gitIn := func(wd string, args ...string) string {
		cmd := exec.Command("git", args...)
		cmd.Dir = wd
		out, err := cmd.CombinedOutput()
		Expect(err).NotTo(HaveOccurred(), "git %v: %s", args, out)
		return string(out)
	}

	BeforeEach(func() {
		var err error
		dir, err = os.MkdirTemp("", "gwt-core-test")
		Expect(err).NotTo(HaveOccurred())
		mainRoot, err = filepath.EvalSymlinks(dir)
		Expect(err).NotTo(HaveOccurred())
		oldWd, err = os.Getwd()
		Expect(err).NotTo(HaveOccurred())

		gitIn(dir, "init", "-q", "-b", "main")
		gitIn(dir, "config", "user.email", "test@example.com")
		gitIn(dir, "config", "user.name", "Test")
		Expect(os.WriteFile(filepath.Join(dir, "file.txt"), []byte("hello\n"), 0o644)).To(Succeed())
		gitIn(dir, "add", ".")
		gitIn(dir, "commit", "-qm", "initial")
		Expect(os.Chdir(mainRoot)).To(Succeed())
	})

	AfterEach(func() {
		_ = os.Chdir(oldWd)
		_ = os.RemoveAll(dir)
	})

	Describe("MainRoot", func() {
		It("resolves the main repo root from inside the main worktree", func() {
			root, err := core.MainRoot()
			Expect(err).NotTo(HaveOccurred())
			Expect(root).To(Equal(mainRoot))
		})

		It("resolves the same root from inside a linked worktree", func() {
			gitIn(dir, "worktree", "add", "-b", "feature", filepath.Join(dir, "worktree", "feature"))
			Expect(os.Chdir(filepath.Join(dir, "worktree", "feature"))).To(Succeed())

			root, err := core.MainRoot()
			Expect(err).NotTo(HaveOccurred())
			Expect(root).To(Equal(mainRoot))
		})
	})

	Describe("NameToPath", func() {
		It("passes absolute paths through", func() {
			p, ok := core.NameToPath("/some/abs/path", mainRoot)
			Expect(ok).To(BeTrue())
			Expect(p).To(Equal("/some/abs/path"))
		})

		It("maps a registered name under worktree/ to its absolute path", func() {
			gitIn(dir, "worktree", "add", "-b", "feature", filepath.Join(dir, "worktree", "feature"))
			p, ok := core.NameToPath("feature", mainRoot)
			Expect(ok).To(BeTrue())
			Expect(p).To(Equal(filepath.Join(mainRoot, "worktree", "feature")))
		})

		It("rejects unregistered names", func() {
			_, ok := core.NameToPath("nope", mainRoot)
			Expect(ok).To(BeFalse())
		})
	})

	Describe("PathToName", func() {
		It("uses the repo basename for the main worktree", func() {
			Expect(core.PathToName(mainRoot, mainRoot)).To(Equal(filepath.Base(mainRoot)))
		})

		It("uses the path relative to worktree/ for linked worktrees", func() {
			gitIn(dir, "worktree", "add", "-b", "feature", filepath.Join(dir, "worktree", "feature"))
			name := core.PathToName(filepath.Join(mainRoot, "worktree", "feature"), mainRoot)
			Expect(name).To(Equal("feature"))
		})

		It("round-trips through nested names", func() {
			gitIn(dir, "worktree", "add", "-b", "feat/x", filepath.Join(dir, "worktree", "feat", "x"))
			wtPath := filepath.Join(mainRoot, "worktree", "feat", "x")
			name := core.PathToName(wtPath, mainRoot)
			Expect(name).To(Equal("feat/x"))
			p, ok := core.NameToPath(name, mainRoot)
			Expect(ok).To(BeTrue())
			Expect(p).To(Equal(wtPath))
		})
	})

	Describe("WorktreePaths", func() {
		It("lists the main worktree first and skips bare repos", func() {
			bare := filepath.Join(dir, "bare.git")
			Expect(exec.Command("git", "init", "-q", "--bare", bare).Run()).To(Succeed())

			paths, err := core.WorktreePaths()
			Expect(err).NotTo(HaveOccurred())
			Expect(paths).To(HaveLen(1))
			Expect(paths[0]).To(Equal(mainRoot))
		})

		It("includes linked worktrees after the main one", func() {
			gitIn(dir, "worktree", "add", "-b", "feature", filepath.Join(dir, "worktree", "feature"))

			paths, err := core.WorktreePaths()
			Expect(err).NotTo(HaveOccurred())
			Expect(paths).To(Equal([]string{
				mainRoot,
				filepath.Join(mainRoot, "worktree", "feature"),
			}))
		})
	})

	Describe("IgnoreWorktreeDir", func() {
		It("appends worktree/ to .git/info/exclude idempotently", func() {
			Expect(core.IgnoreWorktreeDir(mainRoot)).To(Succeed())
			Expect(core.IgnoreWorktreeDir(mainRoot)).To(Succeed())

			data, err := os.ReadFile(filepath.Join(mainRoot, ".git", "info", "exclude"))
			Expect(err).NotTo(HaveOccurred())
			count := 0
			for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
				if line == "worktree/" {
					count++
				}
			}
			Expect(count).To(Equal(1))
		})
	})

	Describe("BranchOfWorktree", func() {
		It("returns the refs/heads branch of a linked worktree", func() {
			gitIn(dir, "worktree", "add", "-b", "feature", filepath.Join(dir, "worktree", "feature"))
			Expect(core.BranchOfWorktree(filepath.Join(mainRoot, "worktree", "feature"))).To(Equal("refs/heads/feature"))
		})

		It("returns empty for a detached worktree", func() {
			gitIn(dir, "worktree", "add", "--detach", filepath.Join(dir, "worktree", "detached"), "HEAD")
			Expect(core.BranchOfWorktree(filepath.Join(mainRoot, "worktree", "detached"))).To(BeEmpty())
		})
	})
})
