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

## 10. Flags

```bash
xx --dry-run delete all node_modules folders    # See command without running
xx --yolo show me disk usage                    # Skip confirmation
xx -v is chrome running                         # Show the underlying command
```

## 11. History & Config

```bash
xx history                          # See past commands
xx history -n 5                     # Last 5 only
xx config show                      # Current configuration
xx config set-model llama3.1:latest # Switch AI model
```

---

## Suggested Demo Flow

For the best impression, run in this order:

1. `xx is chrome running` — shows the smart query intent (no confirmation, friendly answer)
2. `xx show me disk usage` — shows display intent (raw output, no confirmation)
3. `xx --dry-run kill slack` — shows execute intent (command + confirmation)
4. `xx explain "tar -xzf archive.tar.gz"` — shows the explain feature
5. `xx run tests` — shows context awareness (detects project type)
6. `xx go to my downloads` — shows shell navigation (cd in your shell)
7. `cat package.json | xx what deps does this use` — shows pipe input analysis
8. `xx stage everything commit with a good message and push` — shows multi-step workflow with git-aware commit messages
9. `xx chat` → ask a few questions — shows the conversational mode
10. `xx --version` — shows version info
11. `xx history` — shows everything you just did
