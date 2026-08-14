package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ktr0731/go-fuzzyfinder"
	"github.com/spf13/cobra"

	"github.com/umutondersu/gwt/core"
)

var (
	rmKeepBranch bool
	rmForce      bool
)

var rmCmd = &cobra.Command{
	Use:   "rm [. | <name>...] [-B] [-f]",
	Short: "Remove worktrees; -B keeps branch, -f forces dirty",
	Long:  `Remove worktrees; -B keeps branch, -f forces dirty.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		names := args

		mainRoot, err := core.MainRoot()
		if err != nil {
			return err
		}

		var paths []string
		var fzfPaths []string

		if len(names) > 0 {
			for _, name := range names {
				switch {
				case name == ".":
					curPath := core.ShowToplevel()
					if curPath == mainRoot {
						return fmt.Errorf("cannot remove the main worktree")
					}
					paths = append(paths, curPath)
				case strings.HasPrefix(name, "/"):
					paths = append(paths, name)
				default:
					if p, ok := core.NameToPath(name, mainRoot); ok {
						paths = append(paths, p)
					} else {
						fmt.Fprintf(os.Stderr, "No worktree found: %s\n", name)
					}
				}
			}
		} else {
			wtPaths, err := core.WorktreePaths()
			if err != nil {
				return err
			}
			var fzfNames []string
			var fzfPathsAll []string
			for _, path := range wtPaths {
				if path == mainRoot {
					continue
				}
				fzfNames = append(fzfNames, core.PathToName(path, mainRoot))
				fzfPathsAll = append(fzfPathsAll, path)
			}
			if len(fzfNames) == 0 {
				fmt.Println("No worktrees to remove.")
				return nil
			}
			selected, err := pickWorktreesToRemove(fzfNames, fzfPathsAll)
			if err != nil {
				if errors.Is(err, fuzzyfinder.ErrAbort) {
					return nil
				}
				return err
			}
			fzfPaths = selected
		}

		for _, path := range paths {
			if !core.IsRegisteredWorktree(path) {
				return fmt.Errorf("not a worktree: %s", path)
			}
			if !rmForce && core.IsDirty(path) {
				return fmt.Errorf("worktree has uncommitted changes (use -f to force): %s", path)
			}
		}
		for _, path := range fzfPaths {
			if !core.IsRegisteredWorktree(path) {
				return fmt.Errorf("not a worktree: %s", path)
			}
		}

		allPaths := append(paths, fzfPaths...)

		if !core.HasTmux() {
			fmt.Fprintln(os.Stderr, "warning: tmux not found; sessions will not be killed or respawned")
		}

		for _, path := range allPaths {
			curPaneID := core.PaneID()
			curPanePath := core.PanePath()
			curSession := core.SessionName()

			var branch string
			if !rmKeepBranch {
				branch = strings.TrimPrefix(core.BranchOfWorktree(path), "refs/heads/")
			}

			var rmArgs []string
			switch {
			case rmForce:
				rmArgs = append(rmArgs, "--force")
			case containsPath(fzfPaths, path):
				if core.IsDirty(path) {
					rmArgs = append(rmArgs, "--force")
				}
			}
			rmArgs = append([]string{"worktree", "remove"}, rmArgs...)
			rmArgs = append(rmArgs, path)
			if err := core.GitQuiet(rmArgs...); err != nil {
				return err
			}
			fmt.Printf("Removed: %s\n", path)

			pruneEmptyParents(path)

			if !rmKeepBranch && branch != "" {
				branchRef := "refs/heads/" + branch
				merged := false
				if core.IsAncestor(mainRoot, branchRef, "HEAD") {
					merged = true
				} else if upstream, ok := core.UpstreamOf(mainRoot, branch); ok {
					if core.IsAncestor(mainRoot, branchRef, upstream) {
						merged = true
					}
				}
				if !merged {
					confirm, err := prompt(fmt.Sprintf("Branch '%s' is not fully merged. Force delete? [y/N] ", branch))
					if err != nil || !strings.EqualFold(confirm, "y") {
						fmt.Printf("Kept branch: %s\n", branch)
						continue
					}
				}
				if err := core.GitQuietIn(mainRoot, "update-ref", "-d", branchRef); err == nil {
					fmt.Printf("Deleted branch: %s\n", branch)
				}
			}

			for _, sess := range core.PanesInPath(path) {
				if sess != curSession {
					core.TmuxQuiet("kill-session", "-t", "="+sess)
					fmt.Printf("Killed session: %s\n", sess)
				}
			}

			if curPanePath != "" && curPanePath == path {
				repoName := filepath.Base(mainRoot)
				wtName := core.PathToName(path, mainRoot)
				if curSession == repoName+"/"+wtName {
					core.TmuxQuiet("kill-session", "-t", "="+curSession)
				} else {
					core.TmuxQuiet("respawn-pane", "-t", curPaneID, "-k", "-c", mainRoot)
				}
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(rmCmd)
	rmCmd.Flags().BoolVarP(&rmKeepBranch, "keep-branch", "B", false, "Keep the branch")
	rmCmd.Flags().BoolVarP(&rmForce, "force", "f", false, "Force removal of dirty worktrees")
}
