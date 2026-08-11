package cmd

import (
	"fmt"
	"strings"

	"github.com/mattn/go-runewidth"
	"github.com/spf13/cobra"

	"github.com/umutondersu/gwt/core"
)

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List worktrees",
	RunE: func(cmd *cobra.Command, args []string) error {
		paths, err := core.WorktreePaths()
		if err != nil {
			return err
		}
		mainRoot, err := core.MainRoot()
		if err != nil {
			return err
		}

		type row struct {
			name   string
			branch string
			sync   string
			commit string
			isMain bool
		}
		var rows []row
		for _, path := range paths {
			name := core.PathToName(path, mainRoot)

			branch := core.SymbolicShortHEAD(path)
			if branch == "" {
				branch = "(detached HEAD)"
			}

			sync := "-"
			if abOut, abErr := core.GitIn(path, "rev-list", "--left-right", "--count", "HEAD...@{upstream}"); abErr == nil {
				ab := strings.TrimSpace(abOut)
				if ab != "" {
					parts := strings.Fields(ab)
					if len(parts) == 2 {
						sync = fmt.Sprintf("\u2191%s \u2193%s", parts[0], parts[1])
					}
				}
			}

			hash := core.RevParseShort(path)
			msgOut, _ := core.GitIn(path, "log", "-1", "--format=%s")
			msg := strings.TrimSpace(msgOut)
			if runewidth.StringWidth(msg) > 50 {
				msg = runewidth.Truncate(msg, 50, "\u2026")
			}
			commit := hash + ": " + msg

			isMain := path == mainRoot
			displayName := name
			if isMain {
				displayName += " (main)"
			}
			rows = append(rows, row{displayName, branch, sync, commit, isMain})
		}

		wName, wBranch, wSync, wCommit := 8, 6, 4, 11
		for _, r := range rows {
			if w := runewidth.StringWidth(r.name); w > wName {
				wName = w
			}
			if w := runewidth.StringWidth(r.branch); w > wBranch {
				wBranch = w
			}
			if w := runewidth.StringWidth(r.sync); w > wSync {
				wSync = w
			}
			if w := runewidth.StringWidth(r.commit); w > wCommit {
				wCommit = w
			}
		}

		header := fmt.Sprintf("%s%s  %s  %s  %s%s\n", colorBold,
			padRight("Worktree", wName),
			padRight("Branch", wBranch),
			padRight("Last Commit", wCommit),
			padRight("\u2191\u2193", wSync),
			colorReset)
		sep := fmt.Sprintf("%s%s  %s  %s  %s%s\n", colorBrBlack,
			padRight(strings.Repeat("\u2500", wName), wName),
			padRight(strings.Repeat("\u2500", wBranch), wBranch),
			padRight(strings.Repeat("\u2500", wCommit), wCommit),
			padRight(strings.Repeat("\u2500", wSync), wSync),
			colorReset)

		fmt.Println()
		fmt.Print(header, sep)

		for _, r := range rows {
			color := colorYellow
			if r.isMain {
				color = colorCyan
			}
			fmt.Printf("%s%s  %s  %s  %s%s\n", color,
				padRight(r.name, wName),
				padRight(r.branch, wBranch),
				padRight(r.commit, wCommit),
				padRight(r.sync, wSync),
				colorReset)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(lsCmd)
}
