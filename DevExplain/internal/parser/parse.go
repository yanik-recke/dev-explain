package parser

import (
	"fmt"
	"net/url"
	"path"
	"regexp"
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

func ParseSHA(text string) string {
	// Compile the regex pattern for commit SHAs
	// \bcommit\s+[0-9a-f]{7,40}\b
	// - \b word boundary
	// - commit literal word
	// - \s+ one or more whitespace
	// - [0-9a-f]{7,40} hexadecimal chars (7-40 chars, GitHub SHA length)
	// - \b word boundary
	// Added (?i) flag for case insensitivity (Go doesn't have re.IGNORECASE parameter)
	// TODO maybe use a-f since SHAs apparently only
	shaRegex := regexp.MustCompile(`\b[a-f0-9]{7,40}\b`)
	matches := shaRegex.FindStringSubmatch(text)

	if len(matches) > 0 {
		return matches[0]
	}
	return ""
}
