package core

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

func Git(args ...string) (string, error) {
	return gitIn("", args...)
}

func GitIn(dir string, args ...string) (string, error) {
	return gitIn(dir, args...)
}

func GitQuiet(args ...string) error {
	_, err := gitIn("", args...)
	return err
}

func GitQuietIn(dir string, args ...string) error {
	_, err := gitIn(dir, args...)
	return err
}

func GitOutput(args ...string) string {
	out, _ := gitIn("", args...)
	return out
}

func GitOutputIn(dir string, args ...string) string {
	out, _ := gitIn(dir, args...)
	return out
}

func gitIn(dir string, args ...string) (string, error) {
	out, err := runIn(dir, "git", args...)
	if err != nil {
		msg := strings.TrimSpace(out)
		if msg == "" {
			msg = err.Error()
		}
		return out, fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return out, nil
}

func Tmux(args ...string) (string, error) {
	return runIn("", "tmux", args...)
}

func TmuxOutput(args ...string) string {
	out, _ := Tmux(args...)
	return out
}

func TmuxQuiet(args ...string) {
	_, _ = Tmux(args...)
}

func runIn(dir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return out.String(), err
}

func IsInsideWorkTree() bool {
	out, err := Git("rev-parse", "--is-inside-work-tree")
	if err != nil {
		return false
	}
	return strings.TrimSpace(out) == "true"
}

func RefExists(ref string) bool {
	return GitQuiet("show-ref", "--verify", "--quiet", ref) == nil
}

func RevParseVerify(rev string) bool {
	return GitQuiet("rev-parse", "--verify", "--quiet", rev) == nil
}

func RemoteBranchExists(branch string) bool {
	return GitQuiet("ls-remote", "--exit-code", "origin", branch) == nil
}

func IsAncestor(dir, ancestor, descendant string) bool {
	return GitQuietIn(dir, "merge-base", "--is-ancestor", ancestor, descendant) == nil
}

func UpstreamOf(dir, branch string) (string, bool) {
	out, err := GitIn(dir, "rev-parse", "--quiet", "--abbrev-ref", branch+"@{upstream}")
	if err != nil {
		return "", false
	}
	up := strings.TrimSpace(out)
	if up == "" {
		return "", false
	}
	return up, true
}

func IsDirty(path string) bool {
	out, _ := GitIn(path, "status", "--porcelain")
	return strings.TrimSpace(out) != ""
}

func SymbolicShortHEAD(path string) string {
	out, _ := GitIn(path, "symbolic-ref", "--short", "HEAD")
	return strings.TrimSpace(out)
}

func ShowToplevel() string {
	out, _ := Git("rev-parse", "--show-toplevel")
	return strings.TrimSpace(out)
}

func RevParseShort(path string) string {
	out, _ := GitIn(path, "rev-parse", "--short", "HEAD")
	return strings.TrimSpace(out)
}
