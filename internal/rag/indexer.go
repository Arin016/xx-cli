package rag

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/arin/xx-cli/internal/history"
	"github.com/arin/xx-cli/internal/learn"
)

// Indexer builds the vector store by embedding documents from all knowledge sources.
type Indexer struct {
	embedder *EmbedClient
	store    *Store
}

// NewIndexer creates an indexer with the given embedding client.
func NewIndexer(embedder *EmbedClient) *Indexer {
	return &Indexer{
		embedder: embedder,
		store:    NewStore(),
	}
}

// IndexAll embeds all knowledge sources and saves the vector store.
// It calls the progress callback with status messages so the CLI can show progress.
func (idx *Indexer) IndexAll(ctx context.Context, progress func(msg string)) error {
	// 1. Index built-in OS command knowledge.
	progress("Indexing OS command knowledge...")
	osDocs := osCommandDocs()
	if err := idx.embedDocs(ctx, osDocs, progress); err != nil {
		return fmt.Errorf("failed to index OS docs: %w", err)
	}
	progress(fmt.Sprintf("  ✓ %d OS command entries", len(osDocs)))

	// 2. Index learned corrections.
	progress("Indexing learned corrections...")
	learnDocs, err := learnedDocs()
	if err != nil {
		progress(fmt.Sprintf("  ⚠ skipping learned corrections: %v", err))
	} else if len(learnDocs) > 0 {
		if err := idx.embedDocs(ctx, learnDocs, progress); err != nil {
			return fmt.Errorf("failed to index learned docs: %w", err)
		}
		progress(fmt.Sprintf("  ✓ %d learned corrections", len(learnDocs)))
	} else {
		progress("  ✓ no learned corrections yet")
	}

	// 3. Index command history (successful commands only).
	progress("Indexing command history...")
	histDocs, err := historyDocs()
	if err != nil {
		progress(fmt.Sprintf("  ⚠ skipping history: %v", err))
	} else if len(histDocs) > 0 {
		if err := idx.embedDocs(ctx, histDocs, progress); err != nil {
			return fmt.Errorf("failed to index history docs: %w", err)
		}
		progress(fmt.Sprintf("  ✓ %d history entries", len(histDocs)))
	} else {
		progress("  ✓ no command history yet")
	}

	// Save to disk.
	progress("Saving vector store...")
	if err := idx.store.Save(); err != nil {
		return fmt.Errorf("failed to save vector store: %w", err)
	}
	progress(fmt.Sprintf("✓ Indexed %d documents total", idx.store.Len()))

	return nil
}

// embedDocs embeds a batch of documents and adds them to the store.
func (idx *Indexer) embedDocs(ctx context.Context, docs []Document, progress func(string)) error {
	for i := range docs {
		vec, err := idx.embedder.Embed(ctx, docs[i].Text)
		if err != nil {
			return err
		}
		docs[i].Vector = vec
		idx.store.Add(docs[i])

		// Show progress every 50 docs (embedding can be slow).
		if (i+1)%50 == 0 {
			progress(fmt.Sprintf("  embedded %d/%d...", i+1, len(docs)))
		}
	}
	return nil
}

// osCommandDocs returns built-in knowledge about OS-specific commands.
// This is our curated "tldr" — concise, high-signal entries that teach
// the AI which commands to use on this OS.
//
// Each entry is a short description that embeds well: the embedding model
// will place "check memory usage: vm_stat" near queries like "how much RAM".
func osCommandDocs() []Document {
	// We only include docs for the current OS.
	if runtime.GOOS != "darwin" {
		return linuxCommandDocs()
	}
	return macosCommandDocs()
}

func macosCommandDocs() []Document {
	entries := []struct {
		text     string
		category string
	}{
		// Memory / RAM
		{"check memory usage on macOS: use 'vm_stat' to see virtual memory page statistics", "memory"},
		{"how much total RAM on macOS: use 'sysctl hw.memsize' to get total physical memory in bytes", "memory"},
		{"quick memory overview on macOS: use 'top -l 1 -s 0 | head -n 10' for a snapshot of memory and CPU", "memory"},
		{"free RAM percentage on macOS: use 'memory_pressure' to check system memory pressure", "memory"},
		{"NEVER use 'free' or 'free -h' on macOS — it does not exist. Use vm_stat or sysctl hw.memsize instead", "memory"},

		// CPU
		{"check CPU info on macOS: use 'sysctl -n machdep.cpu.brand_string' for CPU model name", "cpu"},
		{"CPU core count on macOS: use 'sysctl -n hw.ncpu' for total cores", "cpu"},
		{"CPU usage on macOS: use 'top -l 1 -s 0 | head -n 10' for a quick CPU snapshot", "cpu"},
		{"NEVER use '/proc/cpuinfo' on macOS — it does not exist. Use sysctl instead", "cpu"},

		// Disk
		{"disk usage on macOS: use 'df -h' to show filesystem usage in human-readable format", "disk"},
		{"largest files on macOS: use 'du -sh * | sort -rh | head -10' to find biggest items in current directory", "disk"},
		{"disk space on macOS: use 'diskutil list' to show all disks and partitions", "disk"},

		// Network
		{"check if port is in use on macOS: use 'lsof -i :PORT' to see what's listening on a port", "network"},
		{"public IP address: use 'curl -s ifconfig.me' to get your public IP", "network"},
		{"local IP address on macOS: use 'ipconfig getifaddr en0' for WiFi IP", "network"},
		{"network connections on macOS: use 'netstat -an | grep LISTEN' to see listening ports", "network"},
		{"DNS lookup: use 'dig example.com' or 'nslookup example.com'", "network"},

		// Process management
		{"check if process is running on macOS: use 'pgrep -x PROCESS_NAME' or 'ps aux | grep PROCESS_NAME'", "process"},
		{"kill a process on macOS: use 'pkill PROCESS_NAME' or 'kill PID'", "process"},
		{"list all running processes: use 'ps aux' for detailed process list", "process"},
		{"find process using a port: use 'lsof -i :PORT' then 'kill PID'", "process"},

		// Package management
		{"install software on macOS: use 'brew install PACKAGE' (Homebrew)", "packages"},
		{"NEVER use 'apt', 'apt-get', or 'yum' on macOS — use 'brew' instead", "packages"},
		{"update packages on macOS: use 'brew update && brew upgrade'", "packages"},
		{"search for a package on macOS: use 'brew search KEYWORD'", "packages"},

		// Files and directories
		{"find files by name: use 'find . -name \"PATTERN\"' or 'find . -iname \"PATTERN\"' for case-insensitive", "files"},
		{"search file contents: use 'grep -r \"PATTERN\" .' to search recursively", "files"},
		{"file permissions: use 'chmod 755 FILE' to set permissions, 'ls -la' to view them", "files"},
		{"compress files on macOS: use 'tar -czf archive.tar.gz FILES' to create a gzip archive", "files"},
		{"extract archive: use 'tar -xzf archive.tar.gz' to extract a gzip archive", "files"},

		// Clipboard
		{"copy to clipboard on macOS: use 'pbcopy' (e.g. 'echo hello | pbcopy')", "clipboard"},
		{"paste from clipboard on macOS: use 'pbpaste'", "clipboard"},
		{"NEVER use 'xclip' or 'xsel' on macOS — use 'pbcopy'/'pbpaste' instead", "clipboard"},

		// System info
		{"macOS version: use 'sw_vers' to show macOS version info", "system"},
		{"system uptime: use 'uptime' to see how long the system has been running", "system"},
		{"open a file or URL on macOS: use 'open FILE' or 'open https://example.com'", "system"},
		{"NEVER use 'xdg-open' on macOS — use 'open' instead", "system"},

		// Git
		{"current git branch: use 'git branch --show-current'", "git"},
		{"git status: use 'git status' to see uncommitted changes", "git"},
		{"git log: use 'git log --oneline -10' for recent commits", "git"},
		{"stage and commit: use 'git add -A && git commit -m \"message\"'", "git"},
		{"undo last commit: use 'git reset --soft HEAD~1' to keep changes staged", "git"},

		// Docker
		{"list docker containers: use 'docker ps' for running, 'docker ps -a' for all", "docker"},
		{"stop docker container: use 'docker stop CONTAINER_ID'", "docker"},
		{"docker logs: use 'docker logs CONTAINER_ID' to view container output", "docker"},
	}

	docs := make([]Document, len(entries))
	for i, e := range entries {
		docs[i] = Document{
			Text:     e.text,
			Source:   "builtin",
			Category: e.category,
		}
	}
	return docs
}

func linuxCommandDocs() []Document {
	entries := []struct {
		text     string
		category string
	}{
		{"check memory usage on Linux: use 'free -h' for human-readable memory info", "memory"},
		{"CPU info on Linux: use 'cat /proc/cpuinfo' or 'lscpu'", "cpu"},
		{"disk usage on Linux: use 'df -h' for filesystem usage", "disk"},
		{"install software on Linux: use 'apt install PACKAGE' (Debian/Ubuntu) or 'yum install PACKAGE' (RHEL/CentOS)", "packages"},
		{"open file on Linux: use 'xdg-open FILE'", "system"},
		{"clipboard on Linux: use 'xclip -selection clipboard' or 'xsel --clipboard'", "clipboard"},
	}

	docs := make([]Document, len(entries))
	for i, e := range entries {
		docs[i] = Document{
			Text:     e.text,
			Source:   "builtin",
			Category: e.category,
		}
	}
	return docs
}

// learnedDocs converts user corrections from learned.json into documents.
// When a user teaches xx "run tests" → "make test", we embed that mapping
// so future queries like "execute my test suite" find it via semantic search.
func learnedDocs() ([]Document, error) {
	corrections, err := learn.LoadAll()
	if err != nil {
		return nil, err
	}

	docs := make([]Document, len(corrections))
	for i, c := range corrections {
		docs[i] = Document{
			Text:     fmt.Sprintf("user correction: when asked '%s', the correct command is '%s'", c.Prompt, c.Command),
			Source:   "learned",
			Category: "learned",
		}
	}
	return docs, nil
}

// historyDocs converts successful command history into documents.
// Past successes are great retrieval targets — if "check disk space" → "df -h"
// worked before, it should be suggested again for similar queries.
func historyDocs() ([]Document, error) {
	entries, err := history.Load(200) // Last 200 successful commands.
	if err != nil {
		return nil, err
	}

	var docs []Document
	seen := make(map[string]bool) // Deduplicate by prompt+command.
	for _, e := range entries {
		if !e.Success || e.Prompt == "" || e.Command == "" {
			continue
		}
		key := e.Prompt + "|" + e.Command
		if seen[key] {
			continue
		}
		seen[key] = true

		docs = append(docs, Document{
			Text:     fmt.Sprintf("'%s' was successfully executed as: %s", e.Prompt, e.Command),
			Source:   "history",
			Category: categorizeCommand(e.Command),
		})
	}
	return docs, nil
}

// categorizeCommand assigns a category to a command based on simple keyword matching.
// This enables the pre-filtering optimization in Search().
func categorizeCommand(cmd string) string {
	lower := strings.ToLower(cmd)
	switch {
	case strings.Contains(lower, "git "):
		return "git"
	case strings.Contains(lower, "docker"):
		return "docker"
	case strings.Contains(lower, "brew "):
		return "packages"
	case strings.Contains(lower, "apt ") || strings.Contains(lower, "yum "):
		return "packages"
	case strings.Contains(lower, "vm_stat") || strings.Contains(lower, "free") || strings.Contains(lower, "memsize"):
		return "memory"
	case strings.Contains(lower, "lsof") || strings.Contains(lower, "netstat") || strings.Contains(lower, "curl"):
		return "network"
	case strings.Contains(lower, "ps ") || strings.Contains(lower, "kill") || strings.Contains(lower, "pgrep"):
		return "process"
	case strings.Contains(lower, "df ") || strings.Contains(lower, "du ") || strings.Contains(lower, "diskutil"):
		return "disk"
	case strings.Contains(lower, "find ") || strings.Contains(lower, "grep ") || strings.Contains(lower, "chmod"):
		return "files"
	default:
		return "general"
	}
}
