package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/zeke-john/komplete/internal/daemon"
)

var daemonPortFile string

var daemonCmd = &cobra.Command{
	Use:    "daemon",
	Short:  "Run the suggestion daemon (used by shell plugin)",
	Hidden: true,
	RunE:   runDaemon,
}

func init() {
	daemonCmd.Flags().StringVar(&daemonPortFile, "port-file", defaultPortFile(), "path to write the listening port")
	rootCmd.AddCommand(daemonCmd)
}

func runDaemon(cmd *cobra.Command, args []string) error {
	if err := os.MkdirAll(filepath.Dir(daemonPortFile), 0o755); err != nil {
		return err
	}

	srv, err := daemon.NewServer(daemonPortFile)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "komplete daemon listening on %s\n", srv.Addr())
	return srv.Run()
}

func defaultPortFile() string {
	tmpDir := os.TempDir()
	return filepath.Join(tmpDir, fmt.Sprintf("komplete-%d.port", os.Getuid()))
}
