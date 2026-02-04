package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/zeke-john/komplete/baml_client"
	baml_types "github.com/zeke-john/komplete/baml_client/types"
	ictx "github.com/zeke-john/komplete/internal/context"
)

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "komplete <request>",
	Short: "Convert natural-language requests into a safe shell command plan",
	Long: `Komplete is a CLI assistant. You type a natural-language request,
Komplete proposes a safe shell command plan, asks for confirmation, then runs it.`,
	Args: cobra.MinimumNArgs(1),
	RunE: runRequest,
}

type runOptions struct {
	dryRun  bool
	model   string
	shell   string
	cwd     string
	timeout time.Duration
	verbose bool
}

var opts runOptions

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err == nil {
		return
	}

	var exitErr *exitError
	if errors.As(err, &exitErr) {
		os.Exit(exitErr.code)
	}

	os.Exit(1)
}

func init() {
	rootCmd.SilenceUsage = true
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		loadDotEnv(".env")
		return nil
	}

	rootCmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "print the plan only, do not execute")
	rootCmd.Flags().StringVar(&opts.model, "model", "", "override the BAML client name")
	rootCmd.Flags().StringVar(&opts.shell, "shell", "", "override detected shell")
	rootCmd.Flags().StringVar(&opts.cwd, "cwd", "", "override working directory")
	rootCmd.Flags().DurationVar(&opts.timeout, "timeout", 10*time.Second, "model request timeout")
	rootCmd.Flags().BoolVar(&opts.verbose, "verbose", false, "show request/response metadata")
}

type exitError struct {
	code int
	err  error
}

func (e *exitError) Error() string {
	return e.err.Error()
}

func runRequest(cmd *cobra.Command, args []string) error {
	request := strings.Join(args, " ")

	ctx, cancel := context.WithTimeout(context.Background(), opts.timeout)
	defer cancel()

	contextInfo, err := ictx.BuildContext(opts.shell, opts.cwd)
	if err != nil {
		return &exitError{code: 1, err: err}
	}

	if opts.verbose {
		fmt.Fprintf(os.Stderr, "Request: %s\nOS: %s\nShell: %s\nCWD: %s\nRepo: %s\nGit: %s\n",
			request, contextInfo.OS, contextInfo.Shell, contextInfo.CWD, contextInfo.RepoRoot, contextInfo.GitStatus)
	}

	callOpts := []baml_client.CallOptionFunc{}
	if opts.verbose {
		os.Setenv("BAML_LOG", "info")
	}
	if opts.model != "" {
		callOpts = append(callOpts, baml_client.WithClient(opts.model))
	}

	plan, err := generatePlanWithRepair(ctx, request, contextInfo, callOpts)
	if err != nil {
		return &exitError{code: 3, err: err}
	}

	plan.Commands = filterCommands(plan.Commands)
	if len(plan.Commands) == 0 {
		fmt.Println("No commands to run.")
		return nil
	}

	fmt.Println("Plan:")
	for i, c := range plan.Commands {
		fmt.Printf("%d) %s\n", i+1, c.Cmd)
	}

	if opts.dryRun {
		return nil
	}

	if !confirm(os.Stdin, os.Stdout) {
		return &exitError{code: 2, err: errors.New("aborted")}
	}

	for i, c := range plan.Commands {
		fmt.Printf("[%d/%d] %s\n", i+1, len(plan.Commands), c.Cmd)
		command := exec.Command(contextInfo.Shell, "-lc", c.Cmd)
		command.Dir = contextInfo.CWD
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		if err := command.Run(); err != nil {
			return &exitError{code: 1, err: err}
		}
	}

	return nil
}

func confirm(in *os.File, out *os.File) bool {
	fmt.Fprint(out, "Run these commands? [y/N] ")
	reader := bufio.NewReader(in)
	line, _ := reader.ReadString('\n')
	answer := strings.TrimSpace(strings.ToLower(line))
	return answer == "y" || answer == "yes"
}

func loadDotEnv(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}
		_ = os.Setenv(key, value)
	}
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

func filterCommands(commands []baml_types.Command) []baml_types.Command {
	filtered := make([]baml_types.Command, 0, len(commands))
	for _, cmd := range commands {
		cmd.Cmd = strings.TrimSpace(cmd.Cmd)
		if cmd.Cmd == "" {
			continue
		}
		filtered = append(filtered, cmd)
	}
	return filtered
}

func generatePlanWithRepair(
	ctx context.Context,
	request string,
	contextInfo ictx.Context,
	callOpts []baml_client.CallOptionFunc,
) (baml_types.Plan, error) {
	plan, err := baml_client.GeneratePlan(
		ctx,
		request,
		contextInfo.OS,
		contextInfo.Shell,
		contextInfo.CWD,
		contextInfo.RepoRoot,
		contextInfo.GitStatus,
		callOpts...,
	)
	if err != nil {
		return baml_types.Plan{}, err
	}

	plan.Commands = filterCommands(plan.Commands)
	invalid := invalidCommands(contextInfo.Shell, contextInfo.CWD, plan.Commands)
	if len(invalid) == 0 {
		return plan, nil
	}

	repairRequest := request + "\n\nThe previous plan included invalid commands: " + strings.Join(invalid, ", ") + ". Replace them with valid, standard macOS/Linux shell commands. Do not invent commands."
	plan2, err := baml_client.GeneratePlan(
		ctx,
		repairRequest,
		contextInfo.OS,
		contextInfo.Shell,
		contextInfo.CWD,
		contextInfo.RepoRoot,
		contextInfo.GitStatus,
		callOpts...,
	)
	if err != nil {
		return plan, nil
	}

	plan2.Commands = filterCommands(plan2.Commands)
	invalid2 := invalidCommands(contextInfo.Shell, contextInfo.CWD, plan2.Commands)
	if len(invalid2) == 0 {
		return plan2, nil
	}

	// Last resort: drop commands whose entrypoints don't exist, and keep going.
	plan2.Commands = dropInvalidCommands(contextInfo.Shell, contextInfo.CWD, plan2.Commands)
	return plan2, nil
}

func invalidCommands(shell string, cwd string, commands []baml_types.Command) []string {
	invalid := []string{}
	seen := map[string]struct{}{}
	for _, c := range commands {
		name := commandEntrypoint(c.Cmd)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		if !commandExists(shell, cwd, name) {
			invalid = append(invalid, name)
		}
	}
	return invalid
}

func dropInvalidCommands(shell string, cwd string, commands []baml_types.Command) []baml_types.Command {
	kept := make([]baml_types.Command, 0, len(commands))
	for _, c := range commands {
		name := commandEntrypoint(c.Cmd)
		if name != "" && !commandExists(shell, cwd, name) {
			continue
		}
		kept = append(kept, c)
	}
	return kept
}

func commandExists(shell string, cwd string, name string) bool {
	check := "command -v -- " + shellQuote(name) + " >/dev/null 2>&1"
	c := exec.Command(shell, "-lc", check)
	c.Dir = cwd
	return c.Run() == nil
}

func commandEntrypoint(command string) string {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return ""
	}

	i := 0
	for i < len(fields) && isEnvAssignment(fields[i]) {
		i++
	}
	if i >= len(fields) {
		return ""
	}

	// Handle common wrappers.
	if fields[i] == "sudo" || fields[i] == "env" {
		i++
		for i < len(fields) && isEnvAssignment(fields[i]) {
			i++
		}
		if i >= len(fields) {
			return ""
		}
	}

	return fields[i]
}

func isEnvAssignment(token string) bool {
	// Very small check: NAME=VALUE where NAME is a typical shell identifier.
	if token == "" {
		return false
	}
	eq := strings.IndexByte(token, '=')
	if eq <= 0 {
		return false
	}
	name := token[:eq]
	for i, r := range name {
		if i == 0 {
			if !(r == '_' || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')) {
				return false
			}
		} else {
			if !(r == '_' || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')) {
				return false
			}
		}
	}
	return true
}
