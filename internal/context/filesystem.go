package context

import (
	"os"
	"path/filepath"
	"strings"
)

const maxDepth = 4

// skipDirs are directories we never want to index.
var skipDirs = map[string]bool{
	"node_modules": true, ".git": true, ".cache": true,
	"vendor": true, "dist": true, "build": true,
	".Trash": true, "Library": true, ".local": true,
	".npm": true, ".cargo": true, ".rustup": true,
	".gradle": true, ".m2": true, ".vscode": true,
	".idea": true, "__pycache__": true, ".tox": true,
}

// ScanDirs returns a list of project-like directories under the user's home.
// It's intentionally shallow (maxDepth) and skips junk directories.
func ScanDirs() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	var dirs []string
	walkDir(home, 0, &dirs)
	return dirs
}

func walkDir(path string, depth int, dirs *[]string) {
	if depth > maxDepth {
		return
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()

		// Skip hidden dirs (except at depth 0) and junk dirs.
		if strings.HasPrefix(name, ".") && depth > 0 {
			continue
		}
		if skipDirs[name] {
			continue
		}

		full := filepath.Join(path, name)

		// Only include dirs that look like projects or meaningful folders.
		if isProjectDir(full) || depth < 2 {
			home, _ := os.UserHomeDir()
			rel, _ := filepath.Rel(home, full)
			*dirs = append(*dirs, "~/"+rel)
		}

		walkDir(full, depth+1, dirs)
	}
}

func isProjectDir(path string) bool {
	projectMarkers := []string{
		"go.mod", "package.json", "Cargo.toml", "build.gradle",
		"gradlew", "pom.xml", "requirements.txt", "pyproject.toml",
		"Makefile", "Dockerfile", "main.tf", "Gemfile",
		"build.gradle.kts", "settings.gradle", "settings.gradle.kts",
	}

	for _, marker := range projectMarkers {
		if _, err := os.Stat(filepath.Join(path, marker)); err == nil {
			return true
		}
	}
	return false
}
