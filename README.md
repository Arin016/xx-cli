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
│  RAG Pipeline        │
│  Embed query → search│
│  vector store → top 5│
│  relevant docs       │
└──────────┬───────────┘
           │ context injected
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

### Step 5: Build the Knowledge Index (recommended)

The RAG (Retrieval-Augmented Generation) pipeline gives `xx` a local knowledge base of OS-specific commands, your learned corrections, and command history. This makes command suggestions significantly more accurate — e.g., using `vm_stat` instead of `free` on macOS.

First, pull the embedding model (~274MB, one-time):

```bash
ollama pull nomic-embed-text
```

Then build the index:

```bash
xx index
# 🔍 Building knowledge index...
#   ✓ 49 OS command entries
#   ✓ 1 learned corrections
#   ✓ 28 history entries (12 skipped as duplicates)
# ✓ Indexed 78 documents total
# Done in 1.1s
```

Re-run `xx index` anytime to refresh (e.g., after teaching `xx` new corrections with `xx learn`, or after building up more command history). Use `--flush` to wipe the existing index and rebuild from scratch — this is the fix for a poisoned index where bad auto-learned commands are dominating good results.

> Without the index, `xx` still works — it just won't have the extra knowledge boost. The RAG pipeline fails silently if no index exists.

### Step 6: Verify everything works

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

> **Tip:** The shell wrapper (`eval "$(xx init zsh)"`) includes `noglob`, so special characters like `?`, `*`, `[]` work out of the box. Without the wrapper, zsh treats `?` as a wildcard — use quotes in that case: `xx "is slack running?"`

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

### Index — Local RAG Knowledge Base

`xx` includes a from-scratch RAG (Retrieval-Augmented Generation) pipeline that embeds OS command knowledge, your learned corrections, and command history into a local vector store. At query time, the most relevant documents are retrieved via cosine similarity and injected into the AI's prompt — so it picks the right command for your OS.

```bash
# Build the index (run once after install, re-run to refresh)
xx index
# 🔍 Building knowledge index...
#   ✓ 49 OS command entries
#   ✓ 1 learned corrections
#   ✓ 28 history entries (12 skipped as duplicates)
# ✓ Indexed 78 documents total
# Done in 1.1s

# Flush and rebuild from scratch (fixes poisoned indexes)
xx index --flush
# 🗑  Flushed existing index
# 🔍 Building knowledge index...
#   ✓ 49 OS command entries
#   ✓ 1 learned corrections
#   ✓ 11 history entries (29 skipped as duplicates)
# ✓ Indexed 61 documents total
```

Use `--verbose` to see what RAG retrieved for any query:

```bash
$ xx -v --dry-run how much RAM do I have

  📚 RAG context:
  - [builtin] how much total RAM on macOS: use 'sysctl hw.memsize'
  - [history] 'how much RAM do i have' was successfully executed as: sysctl hw.memsize
  - [builtin] CPU core count on macOS: use 'sysctl -n hw.ncpu'
  ...

  → sysctl hw.memsize
  Get the total physical memory in bytes
```

Without RAG, the AI might suggest `free -h` (which doesn't exist on macOS). With RAG, it knows to use `sysctl hw.memsize`.

The vector store is a compact binary file (~220KB for 78 docs) stored at `~/.xx-cli/vectors.bin`. No external database dependencies — everything is built from scratch using Ollama's `nomic-embed-text` model for embeddings and cosine similarity for search.

The vector store also grows automatically through auto-learning: every time a command succeeds, `xx` spawns a detached background process that embeds the prompt+command pair and appends it to the store — but only if no near-duplicate already exists (cosine similarity > 0.95). This means the system gets smarter with every use, without you ever running `xx index` again. The background process has zero latency impact on the user.

### Doctor — System Health Check

Run a comprehensive health check on your setup:

```bash
$ xx doctor

  🩺 xx doctor

  ✓ xx binary installed — /Users/you/go/bin/xx
  ✓ $GOPATH/bin in PATH — /Users/you/go/bin
  ✓ Ollama installed — ollama version is 0.15.0
  ✓ Ollama server reachable — localhost:11434
  ✓ Model available (llama3.2:latest) — ready
  ✓ Embedding model (nomic-embed-text) — ready
  ✓ Shell wrapper configured — zsh
  ✓ Config directory — /Users/you/.xx-cli
  ✓ System info — darwin/arm64

  All 9 checks passed. You're good to go.
```

### Stats — Usage Dashboard

See your usage metrics, AI performance, and command patterns:

```bash
$ xx stats

  📊 xx stats

  Commands:  47 total  (12 today, 47 this week)
  Success:   89%
  AI time:   1823ms avg
  Exec time: 156ms avg

  Intent Breakdown
  query      ████████ 18 (38%)
  display    ██████ 14 (30%)
  execute    ████ 9 (19%)
  workflow   ██ 6 (13%)

  Top Commands
  1. ps aux | grep chrome (8x)
  2. df -h (5x)
  3. go test ./... (4x)
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

# Build/refresh the RAG knowledge index
xx index
xx index --flush         # Wipe and rebuild from scratch

# System health check
xx doctor

# Usage statistics
xx stats

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
- **noglob alias** — The shell wrapper includes `alias xx='noglob xx'` so special characters (`?`, `*`, `[]`, `#`) are passed through to `xx` instead of being interpreted by the shell as glob patterns
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
│   ├── doctor.go                  # System health check (9 checks)
│   ├── stats.go                   # Usage statistics dashboard
│   ├── index.go                   # Build RAG knowledge index (--flush support)
│   ├── autolearn.go               # Hidden _learn subcommand for background auto-learning
│   ├── config.go                  # Config subcommands
│   └── history.go                 # History subcommand
├── internal/
│   ├── ai/
│   │   ├── client.go              # AI client, prompt engineering, RAG integration, streaming methods
│   │   ├── provider.go            # Provider interface (pluggable backends)
│   │   ├── ollama.go              # Ollama provider (HTTP + NDJSON streaming)
│   │   ├── stream.go              # StreamingProvider interface, StreamDelta type
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
│   │   ├── history.go             # Command history management
│   │   └── history_test.go        # History tests
│   ├── learn/
│   │   └── learn.go               # Few-shot correction storage
│   ├── rag/
│   │   ├── embeddings.go          # Embedding client (Ollama nomic-embed-text API) with LRU cache
│   │   ├── store.go               # Binary vector store v2: cosine search, adaptive scoring, O(1) append, dedup, flush
│   │   ├── indexer.go             # Indexes OS docs, learned corrections, command history (with dedup against builtins)
│   │   ├── rag.go                 # Top-level Retrieve() + LearnFromSuccess() + RecordFeedback()
│   │   ├── rag_test.go            # 42 tests: cosine similarity, store ops, append, dedup, indexer, formatting
│   │   └── rag_bench_test.go      # Benchmarks: search at 100/1K/10K docs, save/load, append, cosine similarity
│   ├── stats/
│   │   └── stats.go               # Command metrics, aggregation, dashboard data
│   └── ui/
│       ├── spinner.go             # Terminal spinner for loading states
│       └── stream.go              # Streaming token renderer
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
- **Streaming responses** — All free-text AI output streams token-by-token via Ollama's NDJSON streaming API. Uses `StreamingProvider` interface with automatic fallback to `Complete()` for non-streaming providers. Replaces the spinner → wall-of-text pattern with real-time incremental output
- **Structured observability** — Every command is instrumented with AI latency, execution latency, intent, and success/failure. `xx stats` renders a terminal dashboard with aggregated metrics, intent breakdown, and top commands
- **System health check** — `xx doctor` runs 9 checks (binary, PATH, Ollama install, server connectivity, model availability, embedding model, shell wrapper, config dir, system info) with pass/fail/warn output. Same pattern as `brew doctor` and `flutter doctor`
- **Local RAG pipeline** — Built from scratch with no external vector DB dependencies. Uses Ollama's `nomic-embed-text` model (768-dimensional vectors) for embeddings, a custom binary vector store with cosine similarity search, and category pre-filtering for hybrid retrieval. Indexes 3 knowledge sources: curated OS command docs (49 macOS / 6 Linux entries), user-taught corrections from `xx learn`, and successful command history. History entries are deduped against builtins at index time — if a history entry is semantically similar to a curated builtin (cosine > 0.7), it's dropped to prevent auto-learned garbage from competing with curated knowledge. At query time, the top-5 most relevant documents (above 0.3 similarity threshold) are injected into the system prompt with source-based boosting (builtin 1.2x, learned 1.1x). The vector store is a compact binary file (~220KB) — no JSON overhead, no external dependencies. Use `xx -v` to see what RAG retrieved for any query. Use `xx index --flush` to wipe a poisoned index and rebuild from scratch
- **Auto-learning (online learning)** — After every successful command, a detached background process embeds the prompt+command pair and appends it to the vector store via O(1) binary append. Semantic deduplication (cosine similarity > 0.95) prevents bloat. The background process is fully decoupled from the user's session — zero latency impact, and if it fails, nobody notices. This is the write-behind pattern: persist knowledge asynchronously after the user-facing operation completes
- **Adaptive relevance scoring** — Each document in the vector store tracks a success count and failure count. After every command execution, a background process updates the score of the most relevant retrieved document. During search, the final score is `cosine * (1 + ln(1+successes) - 0.5*ln(1+failures))`. This is a lightweight bandit-style signal: reliable commands get boosted, unreliable ones get penalized. New documents start at neutral (1.0 multiplier). Log dampening prevents runaway scores. Same principle as Reddit's ranking algorithm
- **Embedding cache (LRU)** — The embedding client maintains an in-memory LRU cache of 100 entries (~300KB). Repeated queries skip the Ollama API call entirely (0ms vs ~200ms). The cache uses exact string matching with LRU eviction — oldest entries are dropped when the cache is full. This is the same pattern used by DNS resolvers and CDN edge caches
- **Binary format versioning** — The vector store uses a version header (v1 = legacy, v2 = with scoring fields). On load, the reader detects the version and handles both formats transparently. v1 files get scoring fields initialized to zero (neutral). This ensures backward compatibility when upgrading

## Benchmarks

Measured on Apple M4 Pro. Run with `go test ./internal/rag/ -bench=. -benchmem`.

| Operation | Time | Allocations | Notes |
|---|---|---|---|
| Cosine similarity (768-dim) | 562 ns | 0 allocs | Single vector comparison |
| Search (100 docs) | 65 µs | 11 allocs | Typical store size |
| Search (1,000 docs) | 668 µs | 15 allocs | Heavy user after months |
| Search (10,000 docs) | 6.9 ms | 22 allocs | Stress test — still fast |
| Save + Load (100 docs) | 1.8 ms | 1,816 allocs | Full round-trip to disk |
| Save + Load (1,000 docs) | 17 ms | 18,016 allocs | ~3MB on disk |
| Append (single doc, 768-dim) | 69 µs | 18 allocs | O(1) — no full rewrite |
| HasNearDuplicate (100 docs) | 56 µs | 0 allocs | Dedup check |
| Adaptive score calculation | 6.2 ns | 0 allocs | Pure math, no allocation |

Key takeaways:
- Search is sub-millisecond for typical store sizes (< 200 docs). Even at 10K docs, it's under 7ms — well within acceptable latency for a CLI tool
- Append is 250x faster than Save+Load at 100 docs, confirming the O(1) design pays off for online learning
- Cosine similarity is zero-allocation — the hot loop does no heap work
- Adaptive scoring adds ~6ns overhead per document — negligible

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

If you have the [shell wrapper](#step-4-enable-shell-wrapper-recommended) installed, it won't — the wrapper includes `noglob` which disables glob expansion for `xx`, so `?`, `*`, `[]`, and other special characters are passed through as-is. Just make sure to reload your shell after updating (`source ~/.zshrc`).

If you're running `xx` without the shell wrapper, `?` is a glob wildcard in zsh. Either drop it (`xx is slack running`) or quote your prompt (`xx "is slack running?"`).

## Tech Stack

| Component | Technology | Why |
|---|---|---|
| Language | Go | Fast startup, single binary, great CLI ecosystem |
| CLI Framework | [Cobra](https://github.com/spf13/cobra) | Industry standard (kubectl, Hugo, GitHub CLI) |
| AI Backend | [Ollama](https://ollama.com) | Free, local, private, fast |
| Default Model | Llama 3.2 | Good balance of speed and accuracy for command translation |
| Embedding Model | nomic-embed-text | 768-dim vectors, runs locally via Ollama, powers the RAG pipeline |
| Terminal Colors | [fatih/color](https://github.com/fatih/color) | Cross-platform terminal coloring |
| Spinner | [briandowns/spinner](https://github.com/briandowns/spinner) | Smooth loading animations |
| Releases | [GoReleaser](https://goreleaser.com) | Cross-platform binary builds |

## Roadmap

### Planned

| Feature | Description |
|---|---|
| Multiple providers | Support OpenAI, Groq, Anthropic alongside Ollama. `xx config set-provider openai`. The Provider interface is already built — just needs new implementations. |
| Custom rules | `~/.xx-cli/rules.yaml` — teams define rules like "always use pnpm", "never rm -rf without confirmation". AI reads these rules automatically. |
| Shell completion | Tab-complete subcommands and flags in zsh/bash/fish. Cobra supports this natively. |
| Plugin system | Community-driven extensibility. `xx plugin add docker` adds Docker-specific intelligence with custom handlers. |
| Team sharing | Export/import aliases and learned corrections. `xx sync` pushes config to a shared repo so your whole team benefits. |
| Web dashboard | `xx dashboard` opens a local web UI showing command history, usage stats, most-used commands, and success/failure rates. |

