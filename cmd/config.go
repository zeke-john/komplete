package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/zeke-john/komplete/internal/config"
)

// configCmd represents the config command.
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage Komplete configuration",
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a config value",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		if !config.IsAllowedKey(key) {
			return &exitError{code: 1, err: fmt.Errorf("unknown key: %s", key)}
		}
		path, err := config.ConfigPath()
		if err != nil {
			return &exitError{code: 1, err: err}
		}
		cfg, err := config.Load(path)
		if err != nil {
			return &exitError{code: 1, err: err}
		}
		value := cfg[key]
		if value == "" {
			return &exitError{code: 1, err: errors.New("value not set")}
		}
		fmt.Fprintln(os.Stdout, value)
		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a config value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		value := args[1]
		if !config.IsAllowedKey(key) {
			return &exitError{code: 1, err: fmt.Errorf("unknown key: %s", key)}
		}
		path, err := config.ConfigPath()
		if err != nil {
			return &exitError{code: 1, err: err}
		}
		cfg, err := config.Load(path)
		if err != nil {
			return &exitError{code: 1, err: err}
		}
		if cfg == nil {
			cfg = config.Config{}
		}
		cfg[key] = value
		if err := config.Save(path, cfg); err != nil {
			return &exitError{code: 1, err: err}
		}
		fmt.Fprintf(os.Stdout, "Set %s = %s\n", key, value)
		return nil
	},
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured keys",
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := config.ConfigPath()
		if err != nil {
			return &exitError{code: 1, err: err}
		}
		cfg, err := config.Load(path)
		if err != nil {
			return &exitError{code: 1, err: err}
		}
		for _, key := range config.AllowedKeys() {
			if value := cfg[key]; value != "" {
				fmt.Fprintf(os.Stdout, "%s = %s\n", key, value)
			}
		}
		return nil
	},
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Print the config file path",
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := config.ConfigPath()
		if err != nil {
			return &exitError{code: 1, err: err}
		}
		fmt.Fprintln(os.Stdout, path)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configPathCmd)
}
