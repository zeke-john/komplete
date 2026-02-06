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
	Short: "Output zsh autocomplete plugin script",
	Long: `Output the zsh autocomplete plugin. Add this to your .zshrc:

  eval "$(komplete init zsh)"`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print(shell.ZshScript)
	},
}

func init() {
	initCmd.AddCommand(initZshCmd)
	rootCmd.AddCommand(initCmd)
}
