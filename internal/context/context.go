package context

import (
	"bytes"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

type Context struct {
	OS        string
	Shell     string
	CWD       string
	RepoRoot  string
	GitStatus string
}

func BuildContext(shellOverride string, cwdOverride string) (Context, error) {
	ctx := Context{
		OS: runtime.GOOS,
	}

	shell := shellOverride
	if shell == "" {
		shell = os.Getenv("SHELL")
	}
	if shell == "" {
		shell = "sh"
	}
	ctx.Shell = shell

	cwd := cwdOverride
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return Context{}, err
		}
	}
	ctx.CWD = cwd

	repoRoot, gitStatus := detectGit(cwd)
	ctx.RepoRoot = repoRoot
	ctx.GitStatus = gitStatus

	return ctx, nil
}

func detectGit(cwd string) (string, string) {
	repoRoot, err := runGit(cwd, "rev-parse", "--show-toplevel")
	if err != nil || repoRoot == "" {
		return "", ""
	}

	status, err := runGit(cwd, "status", "--porcelain", "-b")
	if err != nil {
		return repoRoot, ""
	}

	return repoRoot, status
}

func runGit(cwd string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = cwd
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}
