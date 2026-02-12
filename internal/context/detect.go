// Package context detects the current project type and environment
// to provide better command suggestions.
package context

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ProjectInfo holds detected project metadata.
type ProjectInfo struct {
	Type        string   // e.g. "go", "node", "python", "rust", "gradle", "unknown"
	Dir         string   // current working directory
	DirName     string   // basename of cwd
	HasGit      bool     // is this a git repo
	HasGradlew  bool     // has ./gradlew wrapper
	ConfigFiles []string // detected config files
	Git         *GitInfo // git context (nil if not a git repo)
}

// GitInfo holds git repository context for smarter AI prompts.
type GitInfo struct {
	Branch     string // current branch name
	DiffStat   string // short summary of uncommitted changes
	RecentLogs string // last 5 commit messages (oneline)
}

// markers maps file names to project types.
var markers = map[string]string{
	"go.mod":         "go",
	"go.sum":         "go",
	"package.json":   "node",
	"yarn.lock":      "node",
	"pnpm-lock.yaml": "node",
	"requirements.txt": "python",
	"pyproject.toml":   "python",
	"Pipfile":          "python",
	"setup.py":         "python",
	"Cargo.toml":       "rust",
	"Gemfile":          "ruby",
	"build.gradle":            "gradle",
	"build.gradle.kts":        "gradle",
	"settings.gradle":         "gradle",
	"settings.gradle.kts":     "gradle",
	"gradlew":                 "gradle",
	"pom.xml":                 "java",
	"Makefile":         "make",
	"Dockerfile":       "docker",
	"docker-compose.yml":  "docker",
	"docker-compose.yaml": "docker",
	"terraform.tf":        "terraform",
	"main.tf":             "terraform",
}

// Detect analyzes the current directory and returns project info.
func Detect() *ProjectInfo {
	cwd, _ := os.Getwd()

	info := &ProjectInfo{
		Type:    "unknown",
		Dir:     cwd,
		DirName: filepath.Base(cwd),
	}

	entries, err := os.ReadDir(cwd)
	if err != nil {
		return info
	}

	for _, entry := range entries {
		name := entry.Name()

		if name == ".git" {
			info.HasGit = true
			continue
		}

		if name == "gradlew" {
			info.HasGradlew = true
		}

		if projType, ok := markers[name]; ok {
			info.ConfigFiles = append(info.ConfigFiles, name)
			// Prefer more specific types over generic ones.
			if info.Type == "unknown" || isMoreSpecific(projType, info.Type) {
				info.Type = projType
			}
		}
	}

	// Gather git context if this is a git repo.
	if info.HasGit {
		info.Git = detectGit(cwd)
	}

	return info
}

// Summary returns a human-readable context string for the AI prompt.
func (p *ProjectInfo) Summary() string {
	var parts []string

	parts = append(parts, "Current directory: "+p.Dir)

	if p.Type != "unknown" {
		parts = append(parts, "Project type: "+p.Type)
	}

	if p.HasGit {
		parts = append(parts, "Git repository: yes")
	}

	if p.Git != nil {
		if p.Git.Branch != "" {
			parts = append(parts, "Git branch: "+p.Git.Branch)
		}
		if p.Git.DiffStat != "" {
			parts = append(parts, "Uncommitted changes:\n"+p.Git.DiffStat)
		}
		if p.Git.RecentLogs != "" {
			parts = append(parts, "Recent commits:\n"+p.Git.RecentLogs)
		}
	}

	if p.HasGradlew {
		parts = append(parts, "Gradle wrapper: yes (use ./gradlew instead of gradle)")
	}

	if len(p.ConfigFiles) > 0 {
		parts = append(parts, "Config files: "+strings.Join(p.ConfigFiles, ", "))
	}

	return strings.Join(parts, "\n")
}

func isMoreSpecific(newType, oldType string) bool {
	// Language-specific types are more specific than tool types.
	tools := map[string]bool{"make": true, "docker": true, "terraform": true}
	if tools[oldType] && !tools[newType] {
		return true
	}
	return false
}

// detectGit gathers git context from the given directory.
// Each call is best-effort — failures are silently ignored.
func detectGit(dir string) *GitInfo {
	gi := &GitInfo{}
	gi.Branch = gitCmd(dir, "rev-parse", "--abbrev-ref", "HEAD")
	gi.DiffStat = gitCmd(dir, "diff", "--stat", "--no-color")
	gi.RecentLogs = gitCmd(dir, "log", "--oneline", "-5", "--no-decorate")
	if gi.Branch == "" && gi.DiffStat == "" && gi.RecentLogs == "" {
		return nil
	}
	return gi
}

// gitCmd runs a git command in the given directory and returns trimmed stdout.
// Returns empty string on any error.
func gitCmd(dir string, args ...string) string {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
