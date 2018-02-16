package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

const Version = "0.1.0-dev"

var (
	flagStorageDir = flag.String("storage", ".storage/", "Local filesystem storage path, used for caching")
)

func main() {
	flag.Parse()

	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Println(Version)
		return
	}

	// TODO(adam): flag or config file, something..
	repos := []string{
		"https://github.com/adamdecaf/cert-manage.git",
	}

	// clone or update local repos
	for i := range repos {
		repo := parseRepoName(repos[i])
		storagePath := path.Join(*flagStorageDir, repo.localpath)

		if dirExists(storagePath) {
			cmd := exec.Command("git", "fetch", "origin")
			cmd.Dir = storagePath
			if err := cmd.Run(); err != nil {
				log.Fatal(err) // TODO(adam)
			}
			log.Printf("git fetch on %s", repo.localpath)
		} else {
			cmd := exec.Command("git", "clone", "--depth=1", repos[i])
			cmd.Dir = filepath.Dir(storagePath) // TODO(adam): probably need to go one level higher
			if err := os.MkdirAll(cmd.Dir, 0755); err != nil {
				log.Fatal(err)
			}
			if err := cmd.Run(); err != nil {
				log.Fatal(err) // TODO(adam)
			}
			log.Printf("git clone on %s", repo.localpath)
		}

	}

	// check if dir exists, clone --depth otherwise
	// if dir exists, git fetch $remote

	// This works instead, `main.go` can be any path
	// $ git log --oneline 9bf483aa848fdfb2b625b6b84fedce6d46b32d68...6abb583cc70fc0e6aa83115aa81f63700979f398 -- main.go
	// 6abb583 cmd/version: on -dev builds add git hash and Go runtime version to output
}

func dirExists(path string) bool {
	s, err := os.Stat(path)
	return err == nil && s.IsDir()
}

type repo struct {
	// e.g. github.com
	host string

	// e.g. github.com/adamdecaf/cert-manage
	localpath string
}

func parseRepoName(repoUrl string) *repo {
	// assume they're URI's, which is probably good enough
	// TODO(adam): I assume git@github.com:adamdecaf/repo.git urls fail?
	u, err := url.Parse(repoUrl)
	if err != nil {
		return nil
	}
	return &repo{
		host:      u.Host,
		localpath: path.Join(u.Host, strings.TrimSuffix(u.Path, ".git")),
	}
}
