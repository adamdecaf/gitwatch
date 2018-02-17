package main

import (
	"testing"
)

func TestGitWatch__parseRepoName(t *testing.T) {
	repo := parseRepoName("https://github.com/adamdecaf/gitwatch.git")
	if repo == nil {
		t.Fatal("expected result")
	}
	if repo.host != "github.com" {
		t.Errorf("got %q", repo.host)
	}
	if repo.localpath != "github.com/adamdecaf/gitwatch" {
		t.Errorf("got %q", repo.localpath)
	}

	// storagePath
	storagePath := repo.storagePath()
	if storagePath != ".storage/github.com/adamdecaf/gitwatch" {
		t.Errorf("got %q", storagePath)
	}
}

func TestGitWatch__head(t *testing.T) {
	repo := parseRepoName("https://github.com/adamdecaf/gitwatch.git")
	if repo == nil {
		t.Fatal("expected result")
	}

	if ref, _ := repo.head(); ref == "" {
		t.Error("expected commit")
	}
}
