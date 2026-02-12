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
│  query   → auto-run, │
│            summarize  │
│  execute → confirm,  │
│            then run   │
│  display → auto-run, │
│            raw output │
└──────────────────────┘
```

**xx** classifies every request into one of three intents:

| Intent | When | Confirmation | Output |
|---|---|---|---|
| `query` | You're asking a question ("is X running?", "how much RAM?") | No — runs automatically | Friendly plain-English summary |
| `execute` | You want an action ("kill Slack", "delete temp files") | Yes — asks `y/N` | ✓ Done / ✗ Failed |
| `display` | You want to see data ("show disk usage", "list files") | No — runs automatically | Raw command output |

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
git clone https://github.com/arin/xx-cli.git
cd xx-cli

# Build and install
make install

# Create the short alias (so you can type 'xx' instead of 'xx-cli')
cp $(go env GOPATH)/bin/xx-cli $(go env GOPATH)/bin/xx
```

### Step 4: Verify everything works

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

### Flags

| Flag | Short | Description |
|---|---|---|
| `--dry-run` | | Show the generated command without executing it |
| `--yolo` | | Skip confirmation even for destructive commands |
| `--verbose` | `-v` | Show the underlying shell command for all intents |

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
- **cd detection** — Since a subprocess can't change your shell's directory, `cd` commands are detected and you're shown the command to run directly
- **Full history** — Every command is logged to `~/.xx-cli/history.json` for audit
- **100% local** — Nothing leaves your machine. Ever.

## Architecture

```
xx-cli/
├── main.go                        # Entry point
├── cmd/
│   ├── root.go                    # CLI setup (Cobra), flag definitions
│   ├── run.go                     # Core execution flow, intent-based UX
│   ├── explain.go                 # Explain subcommand
│   ├── chat.go                    # Interactive chat mode
│   ├── config.go                  # Config subcommands
│   └── history.go                 # History subcommand
├── internal/
│   ├── ai/
│   │   ├── client.go              # Ollama API client, prompt engineering
│   │   └── types.go               # Request/response types
│   ├── config/
│   │   ├── config.go              # Config loading/saving
│   │   └── config_test.go         # Config tests
│   ├── context/
│   │   └── detect.go              # Project type detection (Go, Node, Python, etc.)
│   ├── executor/
│   │   ├── executor.go            # Safe command execution
│   │   └── executor_test.go       # Executor tests
│   ├── history/
│   │   └── history.go             # Command history management
│   └── ui/
│       └── spinner.go             # Terminal spinner for loading states
├── Makefile                       # Build, test, install targets
├── .goreleaser.yaml               # Cross-platform release config
└── .gitignore
```

### Key Design Decisions

- **Ollama (local AI)** — Zero cost, zero latency to external APIs, works offline, full privacy
- **Intent classification** — The AI classifies each request as `query`, `execute`, or `display` to determine UX behavior: whether to confirm, how to present output
- **Smart confirmation** — Only destructive/state-changing commands require user confirmation. Read-only operations run immediately for a frictionless experience
- **Clean output** — Commands are hidden by default for queries and display. The user sees answers, not implementation details
- **Post-execution summarization** — For `query` intents, a second AI call interprets raw command output into a human-friendly answer
- **Context awareness** — Automatically detects project type (Go, Node, Python, Rust, etc.) from config files in the current directory and tailors commands accordingly
- **Loading spinners** — Visual feedback while the AI is thinking, running commands, and summarizing output
- **Command explainer** — `xx explain` breaks down any shell command into plain English, great for learning
- **Cobra CLI framework** — Industry-standard Go CLI library (used by kubectl, Hugo, GitHub CLI)
- **No external dependencies at runtime** — Single binary, just needs Ollama running

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

You'll see a clear error: `is Ollama running? connection refused`. Start it with:

```bash
ollama serve
```

On macOS, Ollama typically runs automatically as a system service after installation.

### Is my data sent anywhere?

No. Everything runs locally on your machine. Your prompts and command outputs never leave your computer.

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

### Tier 1 — Game Changers

| Feature | Description |
|---|---|
| Shell function wrapper | `eval "$(xx init zsh)"` — enables `xx go to sodms` to actually `cd` in your shell. Solves the subprocess directory limitation the same way `zoxide` and `nvm` do. |
| Multi-step workflows | `xx deploy my app` runs a sequence: git add → commit → push → ssh → pull → restart. Shows each step, confirms once, executes sequentially. Like a mini CI pipeline from natural language. |
| Pipe input | `cat error.log \| xx what went wrong` or `xx analyze this file package.json`. Feed file contents or command output into xx for AI-powered debugging and analysis. |

### Tier 2 — Power User Features

| Feature | Description |
|---|---|
| Smart retry | When a command fails, automatically asks the AI to diagnose the error and suggests a corrected command. No more copy-pasting errors into Google. |
| Aliases / shortcuts | `xx alias save deploy "git push origin main"` — save natural language shortcuts. Next time just type `xx deploy`. |
| Shell completion | Tab-complete subcommands and flags in zsh/bash/fish. Cobra supports this natively. |
| `xx watch` | Monitor something continuously: `xx watch is my server still running` — polls every N seconds and alerts you if the status changes. |

### Tier 3 — Portfolio / Wow Factor

| Feature | Description |
|---|---|
| Plugin system | Community-driven extensibility. `xx plugin add docker` adds Docker-specific intelligence with custom handlers. |
| `xx learn` | Correct the AI when it gets a command wrong. Stores corrections as few-shot examples locally so it gets smarter over time for your specific workflow. |
| Team sharing | Export/import aliases and learned corrections. `xx sync` pushes config to a shared repo so your whole team benefits. |
| Web dashboard | `xx dashboard` opens a local web UI showing command history, usage stats, most-used commands, and success/failure rates. |

## License

MIT — see [LICENSE](LICENSE) for details.
