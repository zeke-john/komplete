package history

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

const maxCommands = 5

func GetShellHistory(shell string) string {
	historyFile := getHistoryFile(shell)
	if historyFile == "" {
		return "No shell history available."
	}

	commands, err := readLastCommands(historyFile, shell, maxCommands)
	if err != nil || len(commands) == 0 {
		return "No shell history available."
	}

	return strings.Join(commands, "\n")
}

func getHistoryFile(shell string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	shellName := filepath.Base(shell)
	switch shellName {
	case "zsh":
		return filepath.Join(home, ".zsh_history")
	case "bash":
		if path := filepath.Join(home, ".bash_history"); fileExists(path) {
			return path
		}
		return filepath.Join(home, ".history")
	case "fish":
		return filepath.Join(home, ".local", "share", "fish", "fish_history")
	default:
		return filepath.Join(home, ".history")
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func readLastCommands(path string, shell string, n int) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if len(lines) == 0 {
		return nil, nil
	}

	commands := make([]string, 0, n)
	shellName := filepath.Base(shell)

	for i := len(lines) - 1; i >= 0 && len(commands) < n; i-- {
		line := lines[i]
		cmd := parseHistoryLine(line, shellName)
		if cmd != "" && !isKompleteCommand(cmd) {
			commands = append([]string{cmd}, commands...)
		}
	}

	return commands, nil
}

func parseHistoryLine(line string, shellName string) string {
	switch shellName {
	case "zsh":
		if idx := strings.Index(line, ";"); idx != -1 {
			return strings.TrimSpace(line[idx+1:])
		}
		return strings.TrimSpace(line)
	case "fish":
		if strings.HasPrefix(line, "- cmd: ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "- cmd: "))
		}
		return ""
	default:
		return strings.TrimSpace(line)
	}
}

func isKompleteCommand(cmd string) bool {
	cmd = strings.TrimSpace(cmd)
	return strings.HasPrefix(cmd, "komplete ") ||
		strings.HasPrefix(cmd, "./k ") ||
		strings.HasPrefix(cmd, "k ") ||
		cmd == "komplete" ||
		cmd == "./k" ||
		cmd == "k"
}
