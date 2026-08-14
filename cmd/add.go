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
	addBranch string
	addFrom   string
)

var addCmd = &cobra.Command{
	Use:   "add [<name>] [-b <branch>] [-f <from>]",
	Short: "Create worktree (no arg picks remote branch; -b branch name, -f start point)",
	Long:  `Create a worktree. If no name is provided, picks a remote branch.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var name string
		if len(args) > 0 {
			name = args[0]
		}

		if name == "" && (addBranch != "" || addFrom != "") {
			return fmt.Errorf("usage: gwt add <name> [-b <branch>] [-f <from>]")
		}

		mainRoot, err := core.MainRoot()
		if err != nil {
			return err
		}

		alreadyFetched := false
		if name == "" {
			fmt.Println("Fetching origin...")
			if err := core.GitQuiet("fetch", "--prune", "origin"); err != nil {
				return fmt.Errorf("failed to fetch origin")
			}
			alreadyFetched = true

			out, err := core.Git("for-each-ref", "--format=%(refname:short)", "--exclude=refs/remotes/origin/HEAD", "refs/remotes/origin/")
			if err != nil {
				return err
			}
			refs := splitLines(out)
			if len(refs) == 0 {
				return fmt.Errorf("no remote branches found")
			}

			chosen, err := pickBranch(refs)
			if err != nil {
				if errors.Is(err, fuzzyfinder.ErrAbort) {
					return nil
				}
				return err
			}
			name = chosen
		}

		if p, ok := core.NameToPath(name, mainRoot); ok {
			return core.Connect(p)
		}

		wtPath := filepath.Join(mainRoot, "worktree", name)
		wtDir := filepath.Join(mainRoot, "worktree")
		if _, err := os.Stat(wtDir); os.IsNotExist(err) {
			if err := os.MkdirAll(wtDir, 0o755); err != nil {
				return fmt.Errorf("failed to create directory: %s", wtDir)
			}
			_ = core.IgnoreWorktreeDir(mainRoot)
		}

		branchName := addBranch
		if branchName == "" {
			branchName = name
		}

		if core.RefExists("refs/heads/" + branchName) {
			if err := core.GitQuiet("worktree", "add", wtPath, branchName); err != nil {
				return err
			}
			if addFrom != "" {
				fmt.Fprintf(os.Stderr, "Note: branch '%s' already exists; ignoring -f '%s'.\n", branchName, addFrom)
			}
		} else if addFrom != "" {
			startRef := addFrom
			if !core.RevParseVerify(addFrom + "^{commit}") {
				startRef = "refs/remotes/origin/" + addFrom
			}
			if !core.RevParseVerify(startRef) {
				return fmt.Errorf("start point not found: %s", addFrom)
			}
			if err := core.GitQuiet("worktree", "add", "-b", branchName, wtPath, startRef); err != nil {
				return err
			}
		} else if alreadyFetched || core.RemoteBranchExists(branchName) {
			if !alreadyFetched {
				fmt.Printf("Fetching origin/%s...\n", branchName)
				if err := core.GitQuiet("fetch", "origin", branchName); err != nil {
					return fmt.Errorf("failed to fetch origin/%s", branchName)
				}
			}
			if err := core.GitQuiet("worktree", "add", "--track", "-b", branchName, wtPath, "origin/"+branchName); err != nil {
				return err
			}
		} else {
			base, _ := core.GitIn(mainRoot, "symbolic-ref", "--short", "HEAD")
			base = strings.TrimSpace(base)
			if err := core.GitQuiet("worktree", "add", "-b", branchName, wtPath, base); err != nil {
				return err
			}
		}

		return core.Connect(wtPath)
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
	addCmd.Flags().StringVarP(&addBranch, "branch", "b", "", "Branch name")
	addCmd.Flags().StringVarP(&addFrom, "from", "f", "", "Start point")
}
