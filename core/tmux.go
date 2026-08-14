package core

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func Connect(wtPath string) error {
	mainRoot, err := MainRoot()
	if err != nil {
		return err
	}
	if !HasTmux() {
		fmt.Fprintln(os.Stderr, "warning: tmux not found; no session will be created or attached")
		return nil
	}
	repoName := filepath.Base(mainRoot)
	var sessionName string
	if wtPath == mainRoot {
		sessionName = repoName
	} else {
		sessionName = repoName + "/" + PathToName(wtPath, mainRoot)
	}
	if !SessionExists(sessionName) {
		TmuxQuiet("new-session", "-d", "-s", sessionName, "-c", wtPath)
	}
	TmuxQuiet("switch-client", "-t", "="+sessionName)
	return nil
}

func SessionExists(name string) bool {
	for _, s := range sessionNames() {
		if s == name {
			return true
		}
	}
	return false
}

func PanesInPath(path string) []string {
	out := TmuxOutput("list-panes", "-a", "-F", "#{session_name} #{pane_current_path}")
	seen := map[string]bool{}
	var sessions []string
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] == path && !seen[fields[0]] {
			seen[fields[0]] = true
			sessions = append(sessions, fields[0])
		}
	}
	sort.Strings(sessions)
	return sessions
}

func PanesInSession(sess string) []string {
	out := TmuxOutput("list-panes", "-t", "="+sess, "-F", "#{pane_id}")
	return splitLines(out)
}

func PaneID() string {
	return strings.TrimSpace(TmuxOutput("display-message", "-p", "#{pane_id}"))
}

func PanePath() string {
	return strings.TrimSpace(TmuxOutput("display-message", "-p", "#{pane_current_path}"))
}

func SessionName() string {
	return strings.TrimSpace(TmuxOutput("display-message", "-p", "#{session_name}"))
}

func sessionNames() []string {
	return splitLines(TmuxOutput("list-sessions", "-F", "#{session_name}"))
}

func splitLines(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}
