package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init [shell]",
	Short: "Print shell function to enable directory navigation",
	Long: `Print a shell function wrapper that enables xx to change directories.

Add this to your shell config:
  eval "$(xx init zsh)"    # for ~/.zshrc
  eval "$(xx init bash)"   # for ~/.bashrc
  eval "$(xx init fish)"   # for ~/.config/fish/config.fish`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		shell := "zsh"
		if len(args) > 0 {
			shell = args[0]
		}

		switch shell {
		case "zsh":
			fmt.Print(zshWrapper())
		case "bash":
			fmt.Print(bashWrapper())
		case "fish":
			fmt.Print(fishWrapper())
		default:
			return fmt.Errorf("unsupported shell: %s (supported: zsh, bash, fish)", shell)
		}
		return nil
	},
}

func shellCoreWrapper() string {
	return `    local xx_bin
    xx_bin="$(command which xx-cli 2>/dev/null || command which xx 2>/dev/null)"
    if [ -z "$xx_bin" ]; then
        echo "xx: binary not found in PATH" >&2
        return 1
    fi

    # Capture the output and check for cd hints
    local output
    output="$("$xx_bin" "$@" 2>&1)"
    local exit_code=$?

    # Check if the output contains a cd instruction from xx
    if echo "$output" | grep -q "^__XX_CD__:"; then
        local target
        target="$(echo "$output" | grep "^__XX_CD__:" | head -1 | cut -d: -f2-)"
        # Expand ~ to home
        target="${target/#\~/$HOME}"
        if [ -d "$target" ]; then
            cd "$target" && echo "  → cd $target"
        else
            echo "  Directory not found: $target" >&2
            return 1
        fi
        # Print remaining output (non-cd lines)
        echo "$output" | grep -v "^__XX_CD__:"
    else
        echo "$output"
    fi

    return $exit_code`
}

func zshWrapper() string {
	return `# xx shell wrapper — enables directory navigation and special character handling
xx() {
` + shellCoreWrapper() + `
}

# Disable glob expansion for xx so ?, *, [] etc. are passed as-is.
# This lets you type: xx is slack running?  (without quoting)
alias xx='noglob xx'
`
}

func bashWrapper() string {
	return `# xx shell wrapper — enables directory navigation and special character handling
xx() {
    # Disable glob expansion so ?, *, [] etc. are passed as-is
    local _old_opts="$(shopt -po noglob 2>/dev/null)"
    set -f
    trap 'eval "$_old_opts"' RETURN 2>/dev/null || true
` + shellCoreWrapper() + `
}
`
}

func fishWrapper() string {
	return `# xx shell wrapper — enables directory navigation
function xx
    set xx_bin (command which xx-cli 2>/dev/null; or command which xx 2>/dev/null)
    if test -z "$xx_bin"
        echo "xx: binary not found in PATH" >&2
        return 1
    end

    set output (eval $xx_bin $argv 2>&1)
    set exit_code $status

    if echo "$output" | grep -q "^__XX_CD__:"
        set target (echo "$output" | grep "^__XX_CD__:" | head -1 | string replace "__XX_CD__:" "")
        set target (string replace "~" "$HOME" "$target")
        if test -d "$target"
            cd "$target"; and echo "  → cd $target"
        else
            echo "  Directory not found: $target" >&2
            return 1
        end
        echo "$output" | grep -v "^__XX_CD__:"
    else
        echo "$output"
    end

    return $exit_code
end
`
}
