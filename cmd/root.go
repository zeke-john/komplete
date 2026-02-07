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

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	baml "github.com/boundaryml/baml/engine/language_client_go/pkg"
	"github.com/zeke-john/komplete/baml_client"
	baml_types "github.com/zeke-john/komplete/baml_client/types"
	"github.com/zeke-john/komplete/internal/config"
	ictx "github.com/zeke-john/komplete/internal/context"
	"github.com/zeke-john/komplete/internal/history"
)

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("12"))

	commandStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("14")).
			PaddingLeft(2)

	indexStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			Width(4)

	promptStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("11")).
			Bold(true)

	runningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")).
			Bold(true)

	progressStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))
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
		config.LoadAPIKeysIntoEnv()
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

	shellHistory := history.GetShellHistory(contextInfo.Shell)

	if opts.verbose {
		fmt.Fprintf(os.Stderr, "Request: %s\nOS: %s\nShell: %s\nCWD: %s\nRepo: %s\nGit: %s\nShell history:\n%s\n",
			request, contextInfo.OS, contextInfo.Shell, contextInfo.CWD, contextInfo.RepoRoot, contextInfo.GitStatus, shellHistory)
	}

	callOpts := []baml_client.CallOptionFunc{}
	if opts.verbose {
		os.Setenv("BAML_LOG", "info")
	}
	modelOpt := resolveModelOption(opts.model)
	if modelOpt != nil {
		callOpts = append(callOpts, modelOpt)
	}

	plan, err := generatePlanWithRepair(ctx, request, contextInfo, shellHistory, callOpts)
	if err != nil {
		return &exitError{code: 3, err: err}
	}

	plan.Commands = filterCommands(plan.Commands)
	if len(plan.Commands) == 0 {
		fmt.Println("No commands to run.")
		return nil
	}

	printPlan(plan.Commands)

	if opts.dryRun {
		return nil
	}

	selected := selectCommands(os.Stdin, os.Stdout, len(plan.Commands))
	if len(selected) == 0 {
		return &exitError{code: 2, err: errors.New("aborted")}
	}

	fmt.Println()
	for i, idx := range selected {
		c := plan.Commands[idx]
		printRunningCommand(i+1, len(selected), c.Cmd)
		command := exec.Command(contextInfo.Shell, "-lc", c.Cmd)
		command.Dir = contextInfo.CWD
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		if err := command.Run(); err != nil {
			return &exitError{code: 1, err: err}
		}
		fmt.Println()
	}

	return nil
}

func selectCommands(in *os.File, out *os.File, total int) []int {
	promptText := "Run this command?"
	if total > 1 {
		promptText = "Run these commands?"
	}
	prompt := promptStyle.Render(promptText)
	options := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render(" [y/N/#] ")
	fmt.Fprint(out, prompt+options)
	reader := bufio.NewReader(in)
	line, _ := reader.ReadString('\n')
	answer := strings.TrimSpace(strings.ToLower(line))

	if answer == "" || answer == "n" || answer == "no" {
		return nil
	}

	if answer == "y" || answer == "yes" {
		result := make([]int, total)
		for i := range result {
			result[i] = i
		}
		return result
	}

	var num int
	if _, err := fmt.Sscanf(answer, "%d", &num); err == nil && num >= 1 && num <= total {
		return []int{num - 1}
	}

	return nil
}

func printPlan(commands []baml_types.Command) {
	header := "Command ⟶"
	if len(commands) > 1 {
		header = "Commands ⟶"
	}
	fmt.Println(headerStyle.Render(header))
	for i, c := range commands {
		idx := indexStyle.Render(fmt.Sprintf("%d)", i+1))
		cmd := commandStyle.Render(c.Cmd)
		fmt.Println(idx + cmd)
	}
	fmt.Println()
}

func printRunningCommand(current, total int, cmd string) {
	running := runningStyle.Render(cmd)
	if total == 1 {
		fmt.Println(running)
		return
	}
	progress := progressStyle.Render(fmt.Sprintf("[%d/%d]", current, total))
	fmt.Println(progress + " " + running)
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

var bamlClientNames = map[string]bool{
	"OpenRouter": true,
	"OpenAI":     true,
	"Anthropic":  true,
}

func resolveModelOption(flagValue string) baml_client.CallOptionFunc {
	model := flagValue
	if model == "" {
		path, err := config.ConfigPath()
		if err != nil {
			return nil
		}
		cfg, err := config.Load(path)
		if err != nil {
			return nil
		}
		model = cfg["model"]
	}
	if model == "" {
		return nil
	}

	if bamlClientNames[model] {
		return baml_client.WithClient(model)
	}

	registry := baml.NewClientRegistry()
	registry.AddLlmClient("DynamicClient", "openai-generic", map[string]interface{}{
		"base_url": "https://openrouter.ai/api/v1",
		"api_key":  os.Getenv("OPENROUTER_API_KEY"),
		"model":    model,
	})
	registry.SetPrimaryClient("DynamicClient")
	return baml_client.WithClientRegistry(registry)
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
	historyStr string,
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
		historyStr,
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
		historyStr,
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
