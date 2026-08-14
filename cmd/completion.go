package cmd

import (
	_ "embed"
	"fmt"

	"github.com/spf13/cobra"
)

//go:embed completions/gwt.fish
var fishCompletion string

var completionCmd = &cobra.Command{
	Use:   "completion",
	Short: "Generate the autocompletion script for the specified shell",
	Long: `Generate the autocompletion script for gwt for the specified shell.
See each sub-command's help for details on how to use the generated script.
`,
	Args: cobra.NoArgs,
}

var completionBashCmd = &cobra.Command{
	Use:   "bash",
	Short: "Generate the autocompletion script for bash",
	Long: `Generate the autocompletion script for the bash shell.

To load completions in your current shell session:

	source <(gwt completion bash)

To load completions for every new session, execute once:

	gwt completion bash > /etc/bash_completion.d/gwt

You will need to start a new shell for this setup to take effect.
`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return rootCmd.GenBashCompletionV2(cmd.OutOrStdout(), true)
	},
}

var completionZshCmd = &cobra.Command{
	Use:   "zsh",
	Short: "Generate the autocompletion script for zsh",
	Long: `Generate the autocompletion script for the zsh shell.

To load completions in your current shell session:

	source <(gwt completion zsh)

To load completions for every new session, execute once:

	gwt completion zsh > "${fpath[1]}/_gwt"

You will need to start a new shell for this setup to take effect.
`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return rootCmd.GenZshCompletion(cmd.OutOrStdout())
	},
}

var completionFishCmd = &cobra.Command{
	Use:   "fish",
	Short: "Generate the autocompletion script for fish",
	Long: `Generate the autocompletion script for the fish shell.

To load completions in your current shell session:

	gwt completion fish | source

To load completions for every new session, execute once:

	gwt completion fish > ~/.config/fish/completions/gwt.fish

You will need to start a new shell for this setup to take effect.
`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		_, err := fmt.Fprint(cmd.OutOrStdout(), fishCompletion)
		return err
	},
}

var completionPowershellCmd = &cobra.Command{
	Use:   "powershell",
	Short: "Generate the autocompletion script for powershell",
	Long: `Generate the autocompletion script for powershell.

To load completions in your current shell session:

	gwt completion powershell | Out-String | Invoke-Expression

To load completions for every new session, add the output of the above command
to your powershell profile.
`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return rootCmd.GenPowerShellCompletion(cmd.OutOrStdout())
	},
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	completionCmd.AddCommand(completionBashCmd, completionZshCmd, completionFishCmd, completionPowershellCmd)
	rootCmd.AddCommand(completionCmd)
}
