package core

import "os/exec"

func HasGit() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

func HasTmux() bool {
	_, err := exec.LookPath("tmux")
	return err == nil
}
