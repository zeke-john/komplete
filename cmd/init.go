package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/zeke-john/komplete/shell"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Output shell initialization script",
}

var initZshCmd = &cobra.Command{
	Use:   "zsh",
	Short: "Output zsh autocomplete plugin script (includes alias k=komplete)",
	Long: `Output the zsh autocomplete plugin. Add this to your .zshrc:

  eval "$(komplete init zsh)"`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print(shell.ZshScript)
	},
}

var initAliasCmd = &cobra.Command{
	Use:   "alias",
	Short: "Output shell alias (alias k=komplete)",
	Long: `Output a short alias for komplete. Add this to your .zshrc or .bashrc:

  eval "$(komplete init alias)"`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("alias k=komplete")
	},
}

func init() {
	initCmd.AddCommand(initZshCmd)
	initCmd.AddCommand(initAliasCmd)
	rootCmd.AddCommand(initCmd)
}
