package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/umutondersu/gwt/core"
)

var rootCmd = &cobra.Command{
	Use:   "gwt",
	Short: "Git worktree manager",
	Long:  `A fast, portable Git worktree orchestrator wrapping Git and Tmux.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Name() != "gwt" && !core.IsInsideWorkTree() {
			return fmt.Errorf("not inside a git repository")
		}
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
