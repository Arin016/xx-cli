// Package executor handles safe execution of shell commands.
package executor

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Run executes a shell command and returns its combined output.
// It uses the system's default shell for proper command interpretation.
func Run(command string) (string, error) {
	shell, flag := shellAndFlag()

	cmd := exec.Command(shell, flag, command)
	cmd.Env = os.Environ()
	cmd.Dir, _ = os.Getwd()

	// For cd commands, emit a special marker that the shell wrapper can intercept.
	// If running without the wrapper, it falls back to a helpful hint.
	if isCdCommand(command) {
		dir := extractCdTarget(command)
		expanded := expandHome(dir)
		return fmt.Sprintf("__XX_CD__:%s", expanded), nil
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	output := stdout.String()
	if errOut := stderr.String(); errOut != "" {
		if output != "" {
			output += "\n"
		}
		output += errOut
	}

	return output, err
}

func shellAndFlag() (string, string) {
	if runtime.GOOS == "windows" {
		return "powershell", "-Command"
	}
	if shell := os.Getenv("SHELL"); shell != "" {
		return shell, "-c"
	}
	return "/bin/sh", "-c"
}

func isCdCommand(command string) bool {
	trimmed := strings.TrimSpace(command)
	return trimmed == "cd" || strings.HasPrefix(trimmed, "cd ")
}

func extractCdTarget(command string) string {
	parts := strings.SplitN(strings.TrimSpace(command), " ", 2)
	if len(parts) < 2 {
		return "~"
	}
	return strings.TrimSpace(parts[1])
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			return strings.Replace(path, "~", home, 1)
		}
	}
	return path
}
