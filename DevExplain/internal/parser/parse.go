package parser

import (
	"fmt"
	"net/url"
	"path"
	"strings"
)

// ParseGitHubURL extracts owner and repo name from GitHub URL
// Returns (owner, repo, error)
func ParseGitHubURL(repoURL string) (string, string, error) {
	u, err := url.Parse(repoURL)
	if err != nil {
		return "", "", fmt.Errorf("invalid URL: %w", err)
	}

	// Handle both http and ssh formats
	if u.Host != "github.com" && u.Host != "www.github.com" {
		return "", "", fmt.Errorf("not a GitHub URL")
	}

	// Clean path and split components
	cleanPath := path.Clean(u.Path)
	parts := strings.Split(strings.TrimPrefix(cleanPath, "/"), "/")

	if len(parts) < 2 {
		return "", "", fmt.Errorf("url must contain owner/repo")
	}

	owner := parts[0]
	repo := strings.TrimSuffix(parts[1], ".git") // Remove .git suffix if present

	return owner, repo, nil
}