package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/umutondersu/gwt/core"
)

var mvKeepBranch bool

var mvCmd = &cobra.Command{
	Use:   "mv [<old>] <new> [-B]",
	Short: "Move worktree dir; renames branch if it matches, -B keeps it",
	Long:  `Move worktree dir; renames branch if it matches, -B keeps it.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		mainRoot, err := core.MainRoot()
		if err != nil {
			return err
		}
		repoName := filepath.Base(mainRoot)

		var oldName, newName string
		if len(args) > 0 {
			oldName = args[0]
		}
		if len(args) > 1 {
			newName = args[1]
		}

		if oldName == "" {
			return fmt.Errorf("usage: gwt mv [<old>] <new> [-B]")
		}

		renameCurrent := false
		if newName == "" {
			renameCurrent = true
			newName = oldName
			curPath := core.ShowToplevel()
			if curPath == mainRoot {
				return fmt.Errorf("cannot rename the main worktree")
			}
			oldName = core.PathToName(curPath, mainRoot)
		}

		oldPath, ok := core.NameToPath(oldName, mainRoot)
		if !ok {
			return fmt.Errorf("no worktree found: %s", oldName)
		}

		newPath := filepath.Join(mainRoot, "worktree", newName)
		if _, err := os.Stat(newPath); err == nil {
			return fmt.Errorf("path already exists: %s", newPath)
		}

		branch := core.SymbolicShortHEAD(oldPath)

		oldSess := repoName + "/" + oldName
		newSess := repoName + "/" + newName

		if !core.HasTmux() {
			fmt.Fprintln(os.Stderr, "warning: tmux not found; sessions will not be renamed or respawned")
		}

		renamedOtherSession := false
		var curPaneID string
		if renameCurrent {
			if oldName != newName {
				core.TmuxQuiet("rename-session", "-t", "="+oldSess, newSess)
			}
			curPaneID = core.PaneID()
		} else if core.SessionExists(oldSess) {
			core.TmuxQuiet("rename-session", "-t", "="+oldSess, newSess)
			renamedOtherSession = true
		}

		if err := core.GitQuiet("worktree", "move", oldPath, newPath); err != nil {
			return err
		}

		if !mvKeepBranch && branch != "" && branch == oldName {
			if err := core.GitQuietIn(mainRoot, "branch", "-m", branch, newName); err == nil {
				fmt.Printf("Renamed branch: %s \u2192 %s\n", branch, newName)
			}
		} else if mvKeepBranch && branch != "" {
			fmt.Printf("Kept branch: %s\n", branch)
		}
		fmt.Printf("Moved: %s \u2192 %s\n", oldPath, newPath)

		pruneEmptyParents(oldPath)

		if renameCurrent {
			core.TmuxQuiet("respawn-pane", "-t", curPaneID, "-k", "-c", newPath)
		} else if renamedOtherSession {
			for _, pane := range core.PanesInSession(newSess) {
				core.TmuxQuiet("respawn-pane", "-t", pane, "-k", "-c", newPath)
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(mvCmd)
	mvCmd.Flags().BoolVarP(&mvKeepBranch, "keep-branch", "B", false, "Keep the branch name")
}
