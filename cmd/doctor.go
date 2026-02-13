package cmd

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/arin/xx-cli/internal/config"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check system health and configuration",
	Long: `Run a comprehensive health check on your xx setup.
Verifies Ollama connectivity, model availability, shell wrapper,
PATH configuration, and system resources.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		green := color.New(color.FgGreen)
		red := color.New(color.FgRed)
		yellow := color.New(color.FgYellow)
		dim := color.New(color.FgHiBlack)
		cyan := color.New(color.FgCyan, color.Bold)

		cyan.Fprintf(os.Stderr, "\n  🩺 xx doctor\n\n")

		pass, fail, warn := 0, 0, 0

		check := func(name string, fn func() (string, error)) {
			detail, err := fn()
			if err != nil {
				if strings.HasPrefix(err.Error(), "warn:") {
					yellow.Fprintf(os.Stderr, "  ⚠ %s\n", name)
					dim.Fprintf(os.Stderr, "    %s\n", strings.TrimPrefix(err.Error(), "warn:"))
					warn++
				} else {
					red.Fprintf(os.Stderr, "  ✗ %s\n", name)
					dim.Fprintf(os.Stderr, "    %s\n", err.Error())
					fail++
				}
			} else {
				green.Fprintf(os.Stderr, "  ✓ %s", name)
				if detail != "" {
					dim.Fprintf(os.Stderr, " — %s", detail)
				}
				fmt.Fprintln(os.Stderr)
				pass++
			}
		}

		// 1. Go binary
		check("xx binary installed", func() (string, error) {
			path, err := os.Executable()
			if err != nil {
				return "", fmt.Errorf("could not find xx binary")
			}
			return path, nil
		})

		// 2. GOPATH/bin in PATH
		check("$GOPATH/bin in PATH", func() (string, error) {
			gopath := os.Getenv("GOPATH")
			if gopath == "" {
				home, _ := os.UserHomeDir()
				gopath = filepath.Join(home, "go")
			}
			goBin := filepath.Join(gopath, "bin")
			pathEnv := os.Getenv("PATH")
			if strings.Contains(pathEnv, goBin) {
				return goBin, nil
			}
			return "", fmt.Errorf("add to your shell config: export PATH=\"$HOME/go/bin:$PATH\"")
		})

		// 3. Ollama installed
		check("Ollama installed", func() (string, error) {
			out, err := exec.Command("ollama", "--version").CombinedOutput()
			if err != nil {
				return "", fmt.Errorf("ollama not found — install from https://ollama.com")
			}
			return strings.TrimSpace(string(out)), nil
		})

		// 4. Ollama reachable
		check("Ollama server reachable", func() (string, error) {
			client := &http.Client{Timeout: 3 * time.Second}
			resp, err := client.Get("http://localhost:11434/api/tags")
			if err != nil {
				return "", fmt.Errorf("could not connect — run: ollama serve")
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				return "", fmt.Errorf("unexpected status %d", resp.StatusCode)
			}
			return "localhost:11434", nil
		})

		// 5. Model pulled
		cfg, _ := config.Load()
		check(fmt.Sprintf("Model available (%s)", cfg.Model), func() (string, error) {
			out, err := exec.Command("ollama", "list").CombinedOutput()
			if err != nil {
				return "", fmt.Errorf("could not list models")
			}
			if strings.Contains(string(out), strings.Split(cfg.Model, ":")[0]) {
				return "ready", nil
			}
			return "", fmt.Errorf("model not found — run: ollama pull %s", cfg.Model)
		})

		// 6. Shell wrapper
		check("Shell wrapper configured", func() (string, error) {
			shell := detectDoctorShell()
			home, _ := os.UserHomeDir()
			var rcFile string
			switch shell {
			case "zsh":
				rcFile = filepath.Join(home, ".zshrc")
			case "bash":
				rcFile = filepath.Join(home, ".bashrc")
			case "fish":
				rcFile = filepath.Join(home, ".config", "fish", "config.fish")
			default:
				return "", fmt.Errorf("warn:unknown shell %q — add eval \"$(xx init <shell>)\" to your config", shell)
			}
			data, err := os.ReadFile(rcFile)
			if err != nil {
				return "", fmt.Errorf("warn:could not read %s", rcFile)
			}
			if strings.Contains(string(data), "xx init") {
				return shell, nil
			}
			return "", fmt.Errorf("warn:add to %s: eval \"$(xx init %s)\"", rcFile, shell)
		})

		// 7. Config directory
		check("Config directory", func() (string, error) {
			dir := config.Dir()
			info, err := os.Stat(dir)
			if err != nil {
				return "", fmt.Errorf("warn:~/.xx-cli not found — will be created on first use")
			}
			if !info.IsDir() {
				return "", fmt.Errorf("~/.xx-cli exists but is not a directory")
			}
			return dir, nil
		})

		// 8. OS and arch
		check("System info", func() (string, error) {
			return fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH), nil
		})

		// Summary
		fmt.Fprintln(os.Stderr)
		total := pass + fail + warn
		if fail == 0 && warn == 0 {
			green.Fprintf(os.Stderr, "  All %d checks passed. You're good to go.\n\n", total)
		} else if fail == 0 {
			yellow.Fprintf(os.Stderr, "  %d passed, %d warnings. Everything works, but some things could be better.\n\n", pass, warn)
		} else {
			red.Fprintf(os.Stderr, "  %d passed, %d failed, %d warnings. Fix the failures above.\n\n", pass, fail, warn)
		}

		return nil
	},
}

func detectDoctorShell() string {
	if runtime.GOOS == "windows" {
		return "powershell"
	}
	if shell := os.Getenv("SHELL"); shell != "" {
		parts := strings.Split(shell, "/")
		return parts[len(parts)-1]
	}
	return "sh"
}
