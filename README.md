<p align="center">
  <h1 align="center">xx</h1>
  <p align="center">A natural language CLI assistant that translates plain English into shell commands.</p>
  <p align="center">
    <a href="#setup-from-scratch">Setup</a> •
    <a href="#how-it-works">How It Works</a> •
    <a href="#usage">Usage</a> •
    <a href="#configuration">Configuration</a> •
    <a href="#development">Development</a> •
    <a href="#roadmap">Roadmap</a>
  </p>
</p>

---

**xx** lets you talk to your terminal in plain English. Instead of remembering obscure flags and piping syntax, just describe what you want and get results instantly.

It runs 100% locally using [Ollama](https://ollama.com) — no API keys, no cloud calls, no costs.

```bash
$ xx is chrome running

  Yes, Chrome is running on your system with multiple active processes.

$ xx kill slack

  → pkill Slack
  Terminates the Slack application

Execute? [y/N] y

  ✓ Done.

$ xx show me disk usage

Filesystem   Size  Used  Avail  Capacity  Mounted on
/dev/disk3   460G  402G   28G     94%     /System/Volumes/Data
...

$ xx explain "tar -xzf archive.tar.gz"

  tar -xzf archive.tar.gz

  tar: command for managing archives
  -x: extract the contents
  -z: decompress using gzip
  -f: read from the specified file
  archive.tar.gz: the compressed file to extract

$ xx run tests              # in a Go project
  → go test ./...           # auto-detects project type

$ xx run tests              # in a Node project
  → npm test

$ xx go to my downloads     # actually cd's there (with shell wrapper)
  → cd /Users/you/Downloads

$ cat error.log | xx what went wrong    # pipe input for analysis

  The errors are caused by a missing database connection.
  Check your DB_HOST environment variable.

$ xx stage everything commit with a good message and push

  📋 Workflow (3 steps):

  1. git add -A
  2. git commit -m "feat: add multi-step workflow engine with git context"
  3. git push origin main

  Run all? [y/N] y

  ✓ Step 1: git add -A
  ✓ Step 2: git commit -m "feat: add multi-step workflow engine with git context"
  ✓ Step 3: git push origin main

  ✓ All 3 steps completed.
```

## How It Works

```
┌──────────────────────┐
│  You type in English  │
│  "is chrome running"  │
└──────────┬───────────┘
           │
           ▼
┌──────────────────────┐
│   Ollama (local AI)  │
│  Translates to shell │
│  command + intent    │
└──────────┬───────────┘
           │
           ▼
┌──────────────────────┐
│  Smart UX based on   │
│  intent:             │
│                      │
│  query    → auto-run,│
│             summarize│
│  execute  → confirm, │
│             then run │
│  display  → auto-run,│
│             raw out  │
│  workflow → show plan│
│             confirm, │
│             run steps│
└──────────────────────┘
```

**xx** classifies every request into one of four intents:

| Intent | When | Confirmation | Output |
|---|---|---|---|
| `query` | You're asking a question ("is X running?", "how much RAM?") | No — runs automatically | Friendly plain-English summary |
| `execute` | You want an action ("kill Slack", "delete temp files") | Yes — asks `y/N` | ✓ Done / ✗ Failed |
| `display` | You want to see data ("show disk usage", "list files") | No — runs automatically | Raw command output |
| `workflow` | You want multiple steps ("commit and push", "clean build and test") | Yes — asks `Run all? [y/N]` once | Step-by-step ✓/✗ progress |

For `query` and `display` intents, the underlying command is hidden for a cleaner experience. Use `--verbose` or `-v` to see it.

## Setup (from scratch)

Complete guide to get `xx` running from a fresh machine. If you already have Go and Ollama installed, skip to [Step 3](#step-3-install-xx).

### Step 1: Install Go

Go is the programming language `xx` is built with. You need it to compile the project.

**macOS (Homebrew):**
```bash
brew install go
```

**macOS (manual):**
Download the installer from [go.dev/dl](https://go.dev/dl/) and run it.

**Linux (Ubuntu/Debian):**
```bash
sudo apt update
sudo apt install -y golang-go
```

**Linux (manual — any distro):**
```bash
wget https://go.dev/dl/go1.23.4.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.23.4.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
```

**Windows:**
Download the MSI installer from [go.dev/dl](https://go.dev/dl/) and run it.

**Verify Go is installed:**
```bash
go version
# Should print something like: go version go1.23.4 darwin/arm64
```

After installing Go, make sure `$GOPATH/bin` is in your PATH so you can run Go-installed binaries from anywhere:

```bash
# Add to your ~/.zshrc (macOS) or ~/.bashrc (Linux)
export PATH="$HOME/go/bin:$PATH"

# Reload your shell
source ~/.zshrc   # or source ~/.bashrc
```

### Step 2: Install Ollama

Ollama runs AI models locally on your machine. It's what powers `xx` — no cloud, no API keys, no costs.

**macOS:**
```bash
# Option A: Homebrew
brew install ollama

# Option B: Direct download
# Go to https://ollama.com/download and download the macOS app
```

**Linux:**
```bash
curl -fsSL https://ollama.com/install.sh | sh
```

**Windows:**
Download from [ollama.com/download](https://ollama.com/download).

**Verify Ollama is installed and running:**
```bash
ollama --version
# Should print something like: ollama version is 0.15.0
```

Now pull the default model (Llama 3.2 — ~2GB download, one-time):

```bash
ollama pull llama3.2
```

> Ollama runs as a background service. On macOS it starts automatically after installation. On Linux, you may need to run `ollama serve` in a separate terminal if it's not running as a service.

### Step 3: Install xx

```bash
# Clone the repo
git clone https://github.com/Arin016/xx-cli.git
cd xx-cli

# Build and install
make install

# Create the short alias (so you can type 'xx' instead of 'xx-cli')
cp $(go env GOPATH)/bin/xx-cli $(go env GOPATH)/bin/xx
```

### Step 4: Enable Shell Wrapper (recommended)

The shell wrapper lets `xx` change directories in your actual shell — something a subprocess normally can't do. It also enables seamless `cd` navigation like `xx go to my downloads`.

Add this to your shell config:

```bash
# For zsh (~/.zshrc)
eval "$(xx init zsh)"

# For bash (~/.bashrc)
eval "$(xx init bash)"

# For fish (~/.config/fish/config.fish)
eval "$(xx init fish)"
```

Then reload your shell:

```bash
source ~/.zshrc   # or source ~/.bashrc
```

> Without the wrapper, `xx` still works perfectly for everything else — you just won't get automatic `cd` navigation.

### Step 5: Verify everything works

```bash
xx is ollama running
```

You should see a plain-English answer like "Yes, Ollama is running." If you see a connection error, make sure Ollama is running (`ollama serve`).

That's it. You're ready to go.

## Usage

### Basic

Just type `xx` followed by what you want in plain English:

```bash
# Ask questions — get instant answers, no confirmation needed
xx is chrome running
xx how much free ram do i have
xx what's my public ip address
xx is port 3000 in use

# Perform actions — confirms before executing
xx kill the Slack app
xx delete all .tmp files
xx compress this folder into a tar.gz
xx empty the trash

# See data — runs and shows output directly
xx show me disk usage
xx list all running docker containers
xx show me the top 10 largest files here
xx find all .log files larger than 100mb
```

> **Tip:** Avoid `?` in your prompt — zsh treats it as a wildcard. Write `xx is slack running` instead of `xx is slack running?`, or use quotes: `xx "is slack running?"`

### Explain Commands

Don't know what a command does? Ask `xx` to explain it:

```bash
xx explain "tar -xzf archive.tar.gz"
xx explain "awk '{print $1}' file.txt"
xx explain "find / -name '*.log' -size +100M -delete"
xx explain "chmod 755 script.sh"
```

### Context-Aware Commands

`xx` automatically detects your project type and tailors commands accordingly:

```bash
# In a Go project (detects go.mod)
xx run tests        → go test ./...
xx build this       → go build .

# In a Node project (detects package.json)
xx run tests        → npm test
xx install deps     → npm install

# In a Python project (detects requirements.txt / pyproject.toml)
xx run tests        → pytest
xx install deps     → pip install -r requirements.txt

# In a Rust project (detects Cargo.toml)
xx run tests        → cargo test
xx build this       → cargo build
```

Detected project types: Go, Node.js, Python, Rust, Ruby, Java, Docker, Terraform.

### Shell Navigation (cd)

With the [shell wrapper](#step-4-enable-shell-wrapper-recommended) enabled, `xx` can navigate directories for you:

```bash
xx go to my downloads
# → cd /Users/you/Downloads

xx take me to the sodms project
# → cd ~/AAG/SODMS/sodms

xx go home
# → cd ~
```

Under the hood, the Go binary emits a `__XX_CD__` marker that the shell wrapper intercepts and runs `cd` in your actual shell session. This is the same technique used by tools like `zoxide` and `nvm`.

### Pipe Input (Analyze Data)

Pipe any data into `xx` and ask questions about it:

```bash
# Analyze logs
cat error.log | xx what went wrong
cat server.log | xx summarize the last 10 errors

# Understand files
cat package.json | xx what dependencies does this project use
cat Dockerfile | xx explain this dockerfile

# Process command output
ps aux | xx which process is using the most memory
df -h | xx am i running low on disk space
git log --oneline -20 | xx summarize recent changes
```

When `xx` detects piped input, it switches to analysis mode — the AI reads the data and answers your question directly, no command translation involved.

### Multi-Step Workflows

Describe a complex task in plain English, and `xx` breaks it into a step-by-step pipeline:

```bash
$ xx stage everything commit with a good message and push

  📋 Workflow (3 steps):

  1. git add -A
  2. git commit -m "feat: add multi-step workflow engine with git context"
  3. git push origin main

  Run all? [y/N] y

  ✓ Step 1: git add -A
  ✓ Step 2: git commit -m "feat: add multi-step workflow engine with git context"
  ✓ Step 3: git push origin main

  ✓ All 3 steps completed.
```

More examples:
```bash
xx clean build and run tests          # go clean → go build → go test ./...
xx create a new branch called feature-login and switch to it
xx stop the server on port 3000 and restart it
```

`xx` is git-aware — it reads your current branch, uncommitted changes, and recent commit history to generate meaningful commit messages and accurate git commands. No more generic "update files" commits.

If any step fails, the workflow stops immediately and shows you what went wrong.

### Chat Mode

Start an interactive conversation with `xx` — it remembers context between messages:

```bash
$ xx chat

  xx chat
  Your friendly terminal buddy. Ask me anything.
  Type 'exit' to quit.

  you → how do i check which ports are open on my mac
  xx → You can use lsof to check open ports:
       lsof -i -P | grep LISTEN

  you → how do i kill the one on port 3000
  xx → First find the PID: lsof -i :3000
       Then kill it: kill <PID>

  you → thanks bye
  xx → Later! 👋
```

Great for when you're learning, troubleshooting, or need step-by-step guidance.

### Smart Retry

When a command fails, `xx` automatically diagnoses the error and suggests a fix:

```bash
$ xx install tensorflow

  → pip install tensorflow
  Execute? [y/N] y

  ✗ Failed: ERROR: Could not find a version that satisfies the requirement...

  🔧 Suggested fix:
  → pip3 install tensorflow

  Retry? [y/N] y

  ✓ Done.
```

### WTF — Error Diagnosis

Paste any error message and get an instant diagnosis:

```bash
$ xx wtf "EACCES: permission denied, open /usr/local/lib/node_modules"

  🔍 Diagnosis

  1. What happened: Permission denied when accessing node_modules
  2. Why: The directory is owned by root, not your user
  3. Fix: sudo chown -R $USER /usr/local/lib/node_modules

# Also works with piped input
$ npm install 2>&1 | xx wtf
```

### Learn — Teach xx Your Preferences

When the AI gets a command wrong, correct it. `xx` remembers and uses your corrections as few-shot examples:

```bash
$ xx learn "run tests" "make test"
  ✓ Learned: "run tests" → make test

$ xx learn "deploy" "./scripts/deploy.sh"
  ✓ Learned: "deploy" → ./scripts/deploy.sh

# Now xx always uses your preferred commands
$ xx run tests
  → make test

# View all corrections
$ xx learn --list
```

### Diff Explain — PR Descriptions in Seconds

Reads your git diff and explains what changed in plain English:

```bash
$ xx diff-explain

  📝 Diff Summary

  Added provider interface for pluggable AI backends. Extracted Ollama HTTP
  logic into separate module. Added 18 new tests covering all intent types,
  workflow splitting, and error paths.

$ xx diff-explain --staged    # Only staged changes
```

### Watch — Monitor and Alert

Poll a query and get alerted when the status changes:

```bash
$ xx watch is my server still running
  👁 Watching: is my server still running
  Command: curl -s -o /dev/null -w "%{http_code}" localhost:3000
  Interval: 10s (Ctrl+C to stop)

  [14:23:01] Initial: 200
  [14:23:11] No change
  [14:23:21] ⚠ CHANGED: 000 (connection refused)

$ xx watch --interval 5 is port 3000 in use
```

### Recap — AI-Powered Standup

Summarize your terminal activity into a standup-ready recap:

```bash
$ xx recap

  📋 Today's Recap

  • Built and tested provider abstraction for AI backends
  • 3 git pushes to main branch (xx-cli project)
  • Ran gradle clean build in SODMS project (2 failures, 1 success)
  • Debugged permission issues with node_modules
```

### Flags

| Flag | Short | Description |
|---|---|---|
| `--dry-run` | | Show the generated command without executing it |
| `--yolo` | | Skip confirmation even for destructive commands |
| `--verbose` | `-v` | Show the underlying shell command for all intents |
| `--version` | | Print the version of xx |

```bash
# See what command it would run without executing
xx --dry-run delete all node_modules folders

# Skip confirmation for actions
xx --yolo kill slack

# See the command even for queries
xx -v is chrome running
```

### Subcommands

```bash
# Interactive chat mode
xx chat

# Explain a command
xx explain "tar -xzf archive.tar.gz"

# Diagnose an error
xx wtf "EACCES: permission denied"

# Explain your git diff
xx diff-explain
xx diff-explain --staged

# Monitor something
xx watch is my server running
xx watch --interval 5 is port 3000 in use

# Daily standup recap
xx recap

# Teach xx your preferred commands
xx learn "run tests" "make test"
xx learn --list

# Enable shell wrapper (add to ~/.zshrc)
xx init zsh
xx init bash
xx init fish

# View command history
xx history
xx history -n 5          # Last 5 commands

# Configuration
xx config show           # Show current config
xx config set-model llama3.1:latest   # Change model
```

## Configuration

Config is stored in `~/.xx-cli/config.json`.

| Setting | Environment Variable | Default | Description |
|---|---|---|---|
| Model | `XX_MODEL` | `llama3.2:latest` | Ollama model to use |

Environment variables override the config file.

### Changing the model

```bash
# Use a different Ollama model
xx config set-model llama3.1:latest

# Or via environment variable
export XX_MODEL=llama3.1:latest
```

Any model available in Ollama works. Smaller models are faster, larger models are more accurate:

```bash
ollama list                    # See installed models
ollama pull llama3.1:latest    # Pull a new model
```

## Safety

**xx** is designed with safety as a priority:

- **Smart confirmation** — Only asks for confirmation on state-changing commands (kill, delete, etc.). Questions and data display run automatically since they're read-only
- **Dry run mode** — Use `--dry-run` to see the command without executing it
- **No sudo by default** — The AI never adds `sudo` unless you explicitly ask for it
- **Safe destructive commands** — For operations like `rm` or `kill`, the AI prefers the safest variant
- **cd via shell wrapper** — Directory navigation works through a shell function wrapper (`eval "$(xx init zsh)"`), using the same safe pattern as `zoxide` and `nvm`. Without the wrapper, `cd` commands are detected and shown as output
- **Full history** — Every command is logged to `~/.xx-cli/history.json` for audit
- **Pipe input limits** — Piped data is truncated to 4000 characters to prevent prompt injection and keep responses fast
- **Workflow halt-on-failure** — Multi-step workflows stop immediately if any step fails, preventing cascading damage
- **Chat context cap** — Chat history is limited to 20 messages to stay within the model's context window and prevent degraded responses
- **100% local** — Nothing leaves your machine. Ever.

## Architecture

```
xx-cli/
├── main.go                        # Entry point
├── cmd/
│   ├── root.go                    # CLI setup (Cobra), flag definitions
│   ├── run.go                     # Core execution flow, intent-based UX, pipe input, workflow runner, smart retry
│   ├── init.go                    # Shell wrapper generator (zsh/bash/fish)
│   ├── explain.go                 # Explain subcommand
│   ├── chat.go                    # Interactive chat mode
│   ├── recap.go                   # Daily standup summary from history
│   ├── wtf.go                     # Error diagnosis
│   ├── watch.go                   # Polling monitor with change alerts
│   ├── learn.go                   # Teach xx preferred commands
│   ├── diffexplain.go             # Git diff → plain English summary
│   ├── config.go                  # Config subcommands
│   └── history.go                 # History subcommand
├── internal/
│   ├── ai/
│   │   ├── client.go              # Ollama API client, prompt engineering, workflow translation
│   │   └── types.go               # Intent constants, result types, Ollama request/response types
│   ├── config/
│   │   ├── config.go              # Config loading/saving
│   │   └── config_test.go         # Config tests
│   ├── context/
│   │   ├── detect.go              # Project type + git context detection
│   │   └── filesystem.go          # Directory scanner for navigation
│   ├── executor/
│   │   ├── executor.go            # Safe command execution, cd detection
│   │   └── executor_test.go       # Executor tests
│   ├── history/
│   │   └── history.go             # Command history management
│   ├── learn/
│   │   └── learn.go               # Few-shot correction storage
│   └── ui/
│       └── spinner.go             # Terminal spinner for loading states
├── Makefile                       # Build, test, install targets
├── .goreleaser.yaml               # Cross-platform release config
└── .gitignore
```

### Key Design Decisions

- **Ollama (local AI)** — Zero cost, zero latency to external APIs, works offline, full privacy
- **Intent classification** — The AI classifies each request as `query`, `execute`, `display`, or `workflow` to determine UX behavior: whether to confirm, how to present output, and whether to run a multi-step pipeline
- **Smart confirmation** — Only destructive/state-changing commands require user confirmation. Read-only operations run immediately for a frictionless experience
- **Clean output** — Commands are hidden by default for queries and display. The user sees answers, not implementation details
- **Post-execution summarization** — For `query` intents, a second AI call interprets raw command output into a human-friendly answer
- **Context awareness** — Automatically detects project type (Go, Node, Python, Rust, etc.) from config files in the current directory and tailors commands accordingly
- **Loading spinners** — Visual feedback while the AI is thinking, running commands, and summarizing output
- **Command explainer** — `xx explain` breaks down any shell command into plain English, great for learning
- **Cobra CLI framework** — Industry-standard Go CLI library (used by kubectl, Hugo, GitHub CLI)
- **No external dependencies at runtime** — Single binary, just needs Ollama running
- **Shell wrapper for cd** — Uses the same `eval "$(tool init shell)"` pattern as `zoxide`, `nvm`, and `rbenv`. The Go binary emits a `__XX_CD__` marker, and the shell function intercepts it to run `cd` in the parent shell
- **Pipe input analysis** — Detects stdin data and routes to a dedicated `Analyze()` AI call instead of command translation. Truncates at 4000 chars for safety
- **Multi-step workflows** — When a request involves multiple sequential commands, the AI returns a `workflow` intent with individual steps. Each step runs sequentially with progress feedback, and the pipeline halts on first failure
- **Git context awareness** — Automatically detects current branch, uncommitted changes (`git diff --stat`), and recent commit history. This context is fed into every AI prompt so git commands and commit messages are accurate and meaningful
- **Auto-split safety net** — If the AI chains commands with `&&` despite instructions, the client automatically splits them into proper workflow steps. Ensures consistent step-by-step UX regardless of model behavior
- **Version flag** — `xx --version` prints the build version, set at compile time via Go ldflags
- **Smart retry** — When a command fails, the AI analyzes the error output and suggests a corrected command. One confirmation to retry
- **Few-shot learning** — `xx learn` stores user corrections in `~/.xx-cli/learned.json`. These are injected as few-shot examples into the system prompt, so the AI adapts to your specific workflow over time
- **Error diagnosis** — `xx wtf` takes any error message and returns a structured diagnosis: what happened, why, and the exact fix command
- **Diff explanation** — `xx diff-explain` reads your git diff and generates a human-readable summary, useful for PR descriptions and commit messages
- **Watch mode** — `xx watch` translates a query once, then polls the resulting command at intervals, alerting on output changes with a terminal bell

## Development

### Build

```bash
make build       # Build to bin/xx
make install     # Install to $GOPATH/bin
```

### Test

```bash
make test        # Run all tests with race detection
```

### Lint

```bash
make lint        # Requires golangci-lint
```

### Release

This project uses [GoReleaser](https://goreleaser.com/) for cross-platform builds:

```bash
goreleaser release --snapshot --clean
```

This produces binaries for:
- macOS (amd64, arm64)
- Linux (amd64, arm64)
- Windows (amd64, arm64)

## FAQ

### Does xx run in the background?

No. `xx` only runs when you invoke it, does its job, and exits immediately. **Ollama** runs as a background service (it starts automatically after installation), but `xx` itself has zero background footprint.

### Why don't I see the command for questions?

By design. When you ask "is chrome running", you want the answer — not `ps -ef | grep Chrome`. Use `xx -v is chrome running` if you want to see the command too.

### Can I use a different AI provider?

Currently `xx` is built for Ollama (local inference). The architecture is modular — the AI client is isolated in `internal/ai/` — so swapping to OpenAI, Groq, or any OpenAI-compatible API is straightforward.

### What if Ollama isn't running?

You'll see a clear error: `could not reach Ollama — is it running? (start with: ollama serve)`. Start it with:

```bash
ollama serve
```

On macOS, Ollama typically runs automatically as a system service after installation.

### Is my data sent anywhere?

No. Everything runs locally on your machine. Your prompts and command outputs never leave your computer.

### How does `xx go to downloads` actually change my directory?

It uses a shell wrapper function. When you run `eval "$(xx init zsh)"`, it installs a shell function that intercepts the `xx` command. When the Go binary detects a `cd` operation, it emits a special `__XX_CD__:/path` marker. The shell function catches this and runs `cd` in your actual shell session. This is the same pattern used by `zoxide`, `nvm`, and `rbenv`.

### Why does `xx is slack running?` fail?

The `?` character is a glob wildcard in zsh. Either drop it (`xx is slack running`) or quote your prompt (`xx "is slack running?"`).

## Tech Stack

| Component | Technology | Why |
|---|---|---|
| Language | Go | Fast startup, single binary, great CLI ecosystem |
| CLI Framework | [Cobra](https://github.com/spf13/cobra) | Industry standard (kubectl, Hugo, GitHub CLI) |
| AI Backend | [Ollama](https://ollama.com) | Free, local, private, fast |
| Default Model | Llama 3.2 | Good balance of speed and accuracy for command translation |
| Terminal Colors | [fatih/color](https://github.com/fatih/color) | Cross-platform terminal coloring |
| Spinner | [briandowns/spinner](https://github.com/briandowns/spinner) | Smooth loading animations |
| Releases | [GoReleaser](https://goreleaser.com) | Cross-platform binary builds |

## Roadmap

Features planned for future releases:

### Tier 1 — Next Up

| Feature | Description |
|---|---|
| Multiple providers | Support OpenAI, Groq, Anthropic alongside Ollama. `xx config set-provider openai`. The Provider interface is already built — just needs new implementations. |
| Custom rules | `~/.xx-cli/rules.yaml` — teams define rules like "always use pnpm", "never rm -rf without confirmation". AI reads these rules automatically. |
| Shell completion | Tab-complete subcommands and flags in zsh/bash/fish. Cobra supports this natively. |

### Tier 2 — Power User Features

| Feature | Description |
|---|---|
| Plugin system | Community-driven extensibility. `xx plugin add docker` adds Docker-specific intelligence with custom handlers. |
| Team sharing | Export/import aliases and learned corrections. `xx sync` pushes config to a shared repo so your whole team benefits. |
| Web dashboard | `xx dashboard` opens a local web UI showing command history, usage stats, most-used commands, and success/failure rates. |

## License

MIT — see [LICENSE](LICENSE) for details.
