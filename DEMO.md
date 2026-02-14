# xx — Demo Script

A step-by-step demo showcasing all features of `xx`. Run these commands in order for the best flow.

---

## 1. Questions (auto-runs, friendly answer)

```bash
xx is chrome running
xx how much free ram do i have
xx is port 3000 in use
xx what version of node do i have
```

No confirmation needed — `xx` runs the command and gives you a plain-English answer.

## 2. Actions (confirms before executing)

```bash
xx kill slack
xx empty the trash
xx create a folder called projects
```

Shows the command, asks `Execute? [y/N]`, then reports ✓ Done or ✗ Failed.

## 3. Data Display (auto-runs, raw output)

```bash
xx show me disk usage
xx list all running docker containers
xx show me the top 10 largest files here
xx find all .log files larger than 100mb
```

Runs immediately and shows the command output directly.

## 4. Context Awareness

`xx` detects your project type and tailors commands automatically.

```bash
# Inside a Go project (has go.mod)
xx run tests              # → go test ./...
xx build this             # → go build .

# Inside a Node project (has package.json)
xx run tests              # → npm test
xx install deps           # → npm install

# Inside a Python project (has requirements.txt)
xx run tests              # → pytest
xx install deps           # → pip install -r requirements.txt
```

## 5. Explain Any Command

Paste any command you don't understand:

```bash
xx explain "tar -xzf archive.tar.gz"
xx explain "find / -name '*.log' -size +100M -delete"
xx explain "chmod 755 script.sh"
xx explain "awk '{print $1}' file.txt"
```

## 6. Shell Navigation (requires shell wrapper)

With `eval "$(xx init zsh)"` in your `~/.zshrc`, `xx` can navigate directories:

```bash
xx go to my downloads
# → cd /Users/you/Downloads

xx take me to the sodms project
# → cd ~/AAG/SODMS/sodms

xx go home
# → cd ~
```

## 7. Pipe Input (Analyze Data)

Pipe any data into `xx` and ask questions about it:

```bash
cat error.log | xx what went wrong
ps aux | xx which process is using the most memory
git log --oneline -20 | xx summarize recent changes
cat package.json | xx what dependencies does this use
```

## 8. Multi-Step Workflows

`xx` can break complex tasks into step-by-step pipelines:

```bash
xx stage everything commit with a good message and push
# Shows:
#   📋 Workflow (3 steps):
#   1. git add -A
#   2. git commit -m "feat: meaningful message based on your actual changes"
#   3. git push origin main
#   Run all? [y/N]

xx clean build and run tests
# → go clean → go build → go test ./...

xx create a new branch called feature-login and switch to it
```

`xx` reads your git context (branch, diff, recent commits) to generate accurate commands and meaningful commit messages.

## 9. Interactive Chat Mode

Start a conversation — context carries over between messages:

```bash
xx chat
```

Then try:
```
you → how do i set up ssh keys
you → how do i check which ports are open
you → what's the difference between kill and kill -9
you → bye
```

## 10. Smart Retry (automatic)

When a command fails, `xx` diagnoses the error and suggests a fix — no extra typing needed:

```bash
xx install tensorflow
# → pip install tensorflow
# Execute? [y/N] y
# ✗ Failed: ERROR: Could not find a version...
#
# 🔧 Suggested fix:
# → pip3 install tensorflow
# Retry? [y/N] y
# ✓ Done.
```

## 11. WTF — Error Diagnosis

Paste any error and get an instant diagnosis:

```bash
xx wtf "EACCES: permission denied, open /usr/local/lib/node_modules"
# 🔍 Diagnosis
# 1. What happened: Permission denied when accessing node_modules
# 2. Why: The directory is owned by root, not your user
# 3. Fix: sudo chown -R $USER /usr/local/lib/node_modules

# Also works with piped input
npm install 2>&1 | xx wtf
```

## 12. Learn — Teach xx Your Preferences

Correct the AI when it gets a command wrong. It remembers for next time:

```bash
xx learn "run tests" "make test"
# ✓ Learned: "run tests" → make test

xx learn "deploy" "./scripts/deploy.sh"
# ✓ Learned: "deploy" → ./scripts/deploy.sh

# View all corrections
xx learn --list
```

## 13. Diff Explain — PR Descriptions in Seconds

Reads your git diff and explains what changed in plain English:

```bash
xx diff-explain
# 📝 Diff Summary
# Added provider interface for pluggable AI backends...

xx diff-explain --staged    # Only staged changes
```

## 14. Watch — Monitor and Alert

Poll a query and get alerted when the status changes:

```bash
xx watch is my server still running
# 👁 Watching: is my server still running
# Command: curl -s -o /dev/null -w "%{http_code}" localhost:3000
# Interval: 10s (Ctrl+C to stop)
# [14:23:01] Initial: 200
# [14:23:11] No change
# [14:23:21] ⚠ CHANGED: 000 (connection refused)

xx watch --interval 5 is port 3000 in use
```

## 15. Recap — AI-Powered Standup

Summarize your terminal activity into a standup-ready recap:

```bash
xx recap
# 📋 Today's Recap
# • Built and tested provider abstraction for AI backends
# • 3 git pushes to main branch (xx-cli project)
# • Ran gradle clean build in SODMS project
```

## 16. Doctor — System Health Check

```bash
xx doctor
# 🩺 xx doctor
# ✓ xx binary installed
# ✓ Ollama installed — ollama version is 0.15.0
# ✓ Ollama server reachable
# ✓ Model available (llama3.2:latest)
# ✓ Embedding model (nomic-embed-text) — ready
# ✓ Shell wrapper configured — zsh
# ...
# All 9 checks passed. You're good to go.
```

## 17. Index — RAG Knowledge Base

Build a local vector store that makes `xx` smarter about OS-specific commands:

```bash
# Build the index (one-time, re-run to refresh)
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

# See what RAG retrieved for a query
xx -v --dry-run how much RAM do I have
# 📚 RAG context:
# - [builtin] how much total RAM on macOS: use 'sysctl hw.memsize'
# ...
# → sysctl hw.memsize
```

History entries that overlap with curated builtins are automatically skipped during indexing — this prevents auto-learned bad commands from poisoning the knowledge base. Builtin entries also get a 1.2x score boost at search time.

Without RAG, the AI might suggest `free -h` (doesn't exist on macOS). With RAG, it knows to use `sysctl hw.memsize`.

## 18. Auto-Learning (Online Learning)

Every time a command succeeds, `xx` automatically learns from it in the background — no user action needed:

```bash
# First time: xx doesn't know your preference
xx run tests
# → go test ./...
# ✓ Done.
# (background: embeds "run tests → go test ./..." into vector store)

# Teach it your preference
xx learn "run tests" "make test"

# Next time: xx uses your correction AND remembers the success
xx run tests
# → make test
# ✓ Done.
# (background: embeds "run tests → make test" — dedup skips if already known)
```

The learning happens via a detached background process — zero latency impact on the user. If the embedding fails or Ollama is busy, nobody notices. The vector store just gets smarter over time.

```bash
# Check the doc count before and after
xx index
# ✓ Indexed 71 documents total

# Run a novel command
xx what version of python do i have
# → python3 --version
# ✓ Python 3.12.0

# Re-index to see the new doc
xx index
# ✓ Indexed 72 documents total  ← auto-learned!

# Run the same thing again — dedup kicks in
xx what version of python do i have
# (background: cosine similarity > 0.95 with existing doc → skip)
xx index
# ✓ Indexed 72 documents total  ← no bloat
```

## 19. Stats — Usage Dashboard

```bash
xx stats
# 📊 xx stats
# Commands:  47 total  (12 today, 47 this week)
# Success:   89%
# AI time:   1823ms avg
# Intent Breakdown
# query      ████████ 18 (38%)
# display    ██████ 14 (30%)
# Top Commands
# 1. ps aux | grep chrome (8x)
```

## 20. Flags

```bash
xx --dry-run delete all node_modules folders    # See command without running
xx --yolo show me disk usage                    # Skip confirmation
xx -v is chrome running                         # Show the underlying command
```

## 21. History & Config

```bash
xx history                          # See past commands
xx history -n 5                     # Last 5 only
xx config show                      # Current configuration
xx config set-model llama3.1:latest # Switch AI model
```

---

## Suggested Demo Flow

For the best impression, run in this order:

1. `xx is chrome running` — smart query intent (no confirmation, friendly answer)
2. `xx show me disk usage` — display intent (raw output, no confirmation)
3. `xx --dry-run kill slack` — execute intent (command + confirmation)
4. `xx explain "tar -xzf archive.tar.gz"` — command explainer
5. `xx run tests` — context awareness (detects project type)
6. `xx go to my downloads` — shell navigation (cd in your shell)
7. `cat package.json | xx what deps does this use` — pipe input analysis
8. `xx stage everything commit with a good message and push` — multi-step workflow with git-aware commit messages
9. `xx wtf "EACCES: permission denied"` — instant error diagnosis
10. `xx learn "run tests" "make test"` — teach it your preferences
11. `xx diff-explain` — PR description from your git diff
12. `xx watch is port 3000 in use` — live monitoring with alerts
13. `xx recap` — AI-powered standup summary
14. `xx index` — build the RAG knowledge index
15. `xx index --flush` — flush and rebuild from scratch (shows dedup in action)
16. `xx -v --dry-run how much RAM do I have` — see RAG context + correct macOS command
16. `xx doctor` — system health check (9 pass/fail checks)
17. `xx stats` — usage dashboard with metrics
18. `xx chat` → ask a few questions — conversational mode
19. `xx --version` — version info
20. `xx history` — shows everything you just did
