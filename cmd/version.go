package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version is set at build time with -ldflags "-X github.com/zeke-john/komplete/cmd.Version=..."
var Version = "dev"

// versionCmd represents the version command.
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the Komplete version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("komplete %s\n", Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
