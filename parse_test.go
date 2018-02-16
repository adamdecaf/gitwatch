package main

import (
	"testing"
)

func TestGitWatch__parseRepoName(t *testing.T) {
	repo := parseRepoName("https://github.com/adamdecaf/cert-manage.git")
	if repo == nil {
		t.Fatal("expected result")
	}
	if repo.host != "github.com" {
		t.Errorf("got %q", repo.host)
	}
	if repo.localpath != "github.com/adamdecaf/cert-manage" {
		t.Errorf("got %q", repo.localpath)
	}
}
