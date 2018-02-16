package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
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
		"https://github.com/golang/go.git",
	}

	// clone or update local repos
	for i := range repos {
		repo := newRepo(repos[i])
		oldHead, err := repo.ensure() // fetch or clone
		if err != nil {
			log.Printf("error updating %s, err=%v", repo.localpath, err)
			continue
		}
		curHead, err := repo.head()
		if err != nil {
			log.Printf("error getting current head of %s, err=%v", repo.localpath, err)
			continue
		}

		if curHead != oldHead { // found changes
			commits, err := repo.log(oldHead, curHead)
			if err != nil {
				log.Printf("error getting log of %s from %s to %s, err=%v", repo.localpath, oldHead, curHead, err)
				continue
			}
			fmt.Println(len(commits))
		}
	}
}

func dirExists(path string) bool {
	s, err := os.Stat(path)
	return err == nil && s.IsDir()
}

type repo struct {
	// original url
	repoUrl string

	// e.g. github.com
	host string

	// e.g. github.com/adamdecaf/cert-manage
	localpath string

	// remote name, defaults to origin
	remote string
}

func newRepo(repoUrl string) *repo {
	repo := parseRepoName(repoUrl)
	repo.repoUrl = repoUrl
	if repo.remote == "" {
		repo.remote = "origin"
	}
	return repo
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

func (r *repo) storagePath() string {
	return path.Join(*flagStorageDir, r.localpath)
}

func (r *repo) head() (string, error) {
	out, err := exec.Command("git", "rev-parse", "HEAD").CombinedOutput()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func (r *repo) ensure() (prevHead string, err error) {
	storagePath := r.storagePath()

	// Just fetch updates if we've got a local clone
	if dirExists(storagePath) {
		head, err := r.head()
		if err != nil {
			return "", err
		}
		cmd := exec.Command("git", "fetch", r.remote)
		cmd.Dir = storagePath
		if err := cmd.Run(); err != nil {
			return "", err
		}
		log.Printf("git fetch on %s", r.localpath)
		return head, nil
	}

	// Clone fresh copy locally
	cmd := exec.Command("git", "clone", "--depth=1", r.repoUrl)
	cmd.Dir = filepath.Dir(storagePath)
	if err := os.MkdirAll(cmd.Dir, 0755); err != nil {
		return "", err
	}
	if err := cmd.Run(); err != nil {
		return "", err
	}
	log.Printf("git clone on %s", r.localpath)
	return "", nil // fresh clone so no previous HEAD
}

type commit struct {
	shortRef string
	message  string
}

// $ git log --oneline 9bf483aa848fdfb2b625b6b84fedce6d46b32d68...6abb583cc70fc0e6aa83115aa81f63700979f398 -- main.go
// 6abb583 cmd/version: on -dev builds add git hash and Go runtime version to output
func (r *repo) log(old, cur string) ([]*commit, error) {
	out, err := exec.Command("git", "log", "--oneline", fmt.Sprintf("%s...%s", old, cur)).CombinedOutput() // TODO(adam): -- $dir
	if err != nil {
		return nil, err
	}
	var commits []*commit
	rdr := bufio.NewScanner(bytes.NewReader(out))
	for rdr.Scan() {
		line := strings.TrimSpace(rdr.Text())
		if err := rdr.Err(); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		if line == "" {
			continue
		}

		// split into shortRef and message
		idx := strings.Index(line, " ")
		if idx > 0 {
			shortRef := line[:idx-1]
			message := line[idx:]
			commits = append(commits, &commit{
				shortRef: shortRef,
				message:  message,
			})
		}
	}
	return commits, nil
}
