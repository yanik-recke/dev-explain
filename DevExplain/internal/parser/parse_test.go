package parser

import "testing"

func TestShaParse(t *testing.T) {
	sha := ParseSHA("Explain commit c587fc428d.")

	if sha != "c587fc428d" {
		t.Errorf("sha: %s", sha)
	}
}

func TestShaParse2(t *testing.T) {
	sha := ParseSHA("What happened in commit d6679ecd14?")

	if sha != "d6679ecd14" {
		t.Errorf("sha: %s", sha)
	}
}
