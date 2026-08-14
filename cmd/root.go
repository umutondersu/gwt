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
	rootCmd.SetUsageTemplate(usageTemplate())
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func usageTemplate() string {
	width := 0
	for _, c := range rootCmd.Commands() {
		if c.Name() == "help" || c.Name() == "completion" {
			continue
		}
		if n := len(c.Use); n > width {
			width = n
		}
	}
	return fmt.Sprintf(`Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}{{$cmds := .Commands}}{{if eq (len .Groups) 0}}

Available Commands:{{range $cmds}}{{if (and .IsAvailableCommand (ne .Name "help") (ne .Name "completion"))}}
  {{rpad .Use %d}} {{.Short}}{{end}}{{end}}{{else}}{{range $group := .Groups}}

{{.Title}}{{range $cmds}}{{if (and (eq .GroupID $group.ID) (and .IsAvailableCommand (ne .Name "help") (ne .Name "completion")))}}
  {{rpad .Use %d}} {{.Short}}{{end}}{{end}}{{end}}{{if not .AllChildCommandsHaveGroup}}

Additional Commands:{{range $cmds}}{{if (and (eq .GroupID "") (and .IsAvailableCommand (ne .Name "help") (ne .Name "completion")))}}
  {{rpad .Use %d}} {{.Short}}{{end}}{{end}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`, width+4, width+4, width+4)
}
