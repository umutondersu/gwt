package cmd

import (
	"errors"
	"fmt"

	"github.com/ktr0731/go-fuzzyfinder"
	"github.com/spf13/cobra"

	"github.com/umutondersu/gwt/core"
)

var pickCmd = &cobra.Command{
	Use:   "pick [<name>]",
	Short: "Pick a worktree and connect to its session",
	Long:  `Pick a worktree and connect to its tmux session.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mainRoot, err := core.MainRoot()
		if err != nil {
			return err
		}

		var wtPath string
		if len(args) > 0 {
			p, ok := core.NameToPath(args[0], mainRoot)
			if !ok {
				return fmt.Errorf("no worktree found: %s", args[0])
			}
			wtPath = p
		} else {
			paths, err := core.WorktreePaths()
			if err != nil {
				return err
			}
			names := make([]string, len(paths))
			for i, path := range paths {
				name := core.PathToName(path, mainRoot)
				if path == mainRoot {
					name += " (main)"
				}
				names[i] = name
			}
			if len(names) == 0 {
				return fmt.Errorf("no worktrees found")
			}
			chosen, err := pickWorktree(names, paths)
			if err != nil {
				if errors.Is(err, fuzzyfinder.ErrAbort) {
					return nil
				}
				return err
			}
			wtPath = chosen
		}
		return core.Connect(wtPath)
	},
}

func init() {
	rootCmd.AddCommand(pickCmd)
}
