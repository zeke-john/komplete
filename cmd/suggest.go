package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/zeke-john/komplete/internal/config"
	"github.com/zeke-john/komplete/internal/history"
	"github.com/zeke-john/komplete/internal/suggest"
)

var suggestCwd string

var suggestCmd = &cobra.Command{
	Use:    "suggest [partial command...]",
	Short:  "Suggest a command completion (used by shell plugin)",
	Hidden: true,
	Args:   cobra.MinimumNArgs(1),
	RunE:   runSuggest,
}

func init() {
	suggestCmd.Flags().StringVar(&suggestCwd, "cwd", "", "working directory for context")
	rootCmd.AddCommand(suggestCmd)
}

func runSuggest(cmd *cobra.Command, args []string) error {
	buffer := strings.Join(args, " ")
	if strings.TrimSpace(buffer) == "" {
		return nil
	}

	config.LoadAPIKeysIntoEnv()

	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		return nil
	}

	model := groqModel()

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "zsh"
	}

	cwd := suggestCwd
	if cwd == "" {
		cwd, _ = os.Getwd()
	}

	historyStr := history.GetShellHistory(shell)

	client := suggest.NewClient(apiKey, model)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	suggestion, err := client.Complete(ctx, buffer, cwd, shell, historyStr)
	if err != nil || suggestion == "" {
		return nil
	}

	fmt.Println(suggestion)
	return nil
}

func groqModel() string {
	path, err := config.ConfigPath()
	if err != nil {
		return ""
	}
	cfg, err := config.Load(path)
	if err != nil {
		return ""
	}
	return cfg["groq_model"]
}
