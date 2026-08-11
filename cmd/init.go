package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/umutondersu/gwt/core"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Ensure worktree/ dir is git-ignored",
	Long:  `Ensure the worktree dir inside the repo is git-ignored.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		mainRoot, err := core.MainRoot()
		if err != nil {
			return err
		}
		if err := core.IgnoreWorktreeDir(mainRoot); err != nil {
			return err
		}
		fmt.Printf("worktree/ is git-ignored in %s\n", mainRoot)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
