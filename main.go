// gitwatch periodically refreshes git repositories to note
// changes and report those on an http web page or
// prometheus metrics endpoint
package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const Version = "0.1.1-dev"

var (
	flagConfigFile   = flag.String("config", "", "config file, newline delimited of git repos")
	flagPollInterval = flag.Duration("interval", time.Hour, "how often to refresh git repos")
	flagStorageDir   = flag.String("storage", ".storage/", "Local filesystem storage path, used for caching")

	defaultRepository = newRepo("https://github.com/adamdecaf/gitwatch.git")

	mu    = sync.RWMutex{} // guards repos, lastUpdatedAt
	repos []*repo
)

func main() {
	flag.Parse()

	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Println(Version)
		return
	}

	mu.Lock()
	repos = collectRepos(*flagConfigFile)
	mu.Unlock()
	go func(repos []*repo, pollInterval time.Duration) {
		for {
			watchRepos()
			time.Sleep(pollInterval)
		}
	}(repos, *flagPollInterval)

	log.Fatal(http.ListenAndServe(":6060", handler{}))
}

type handler struct{}

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	mu.RLock()
	defer mu.RUnlock()

	// all output is text/plain
	w.Header().Set("Content-Type", "text/plain")

	// prometheus metrics
	if strings.HasPrefix(r.URL.Path, "/metrics") {
		for i := range repos {
			var commits []*commit
			for _, v := range repos[i].recentCommits {
				commits = append(commits, v...)
			}
			fmt.Fprintf(w, "gitwatch_recent_updated{repo=\"%s\"} %d\n", repos[i].localpath, len(commits))
		}
		return
	}

	// more human oriented format
	for i := range repos {
		var commits []*commit
		for _, v := range repos[i].recentCommits {
			commits = append(commits, v...)
		}
		fmt.Fprintf(w, "%s: %d new commits\n", repos[i].localpath, len(commits))
	}
}

func collectRepos(where string) []*repo {
	f, err := os.Open(where)
	if err != nil {
		if os.IsNotExist(err) {
			return []*repo{
				defaultRepository,
			}
		}
		log.Printf("error reading %s, err=%v", where, err)
		return nil
	}

	// split file based on newlines
	var repos []*repo
	r := bufio.NewScanner(f)
	for r.Scan() {
		line := strings.TrimSpace(r.Text())
		if err := r.Err(); err != nil {
			break // return on io.EOF or any error
		}
		repos = append(repos, newRepo(line))
	}
	return repos
}

func watchRepos() {
	// clone or update local repos
	for i := range repos {
		repo := repos[i]
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

		if oldHead == "" {
			continue
		}

		if oldHead != curHead { // found changes
			commits, err := repo.log(oldHead, curHead)
			if err != nil {
				log.Printf("error getting log of %s from %s to %s, err=%v", repo.localpath, oldHead, curHead, err)
				continue
			}
			mu.Lock()
			prev := time.Now().Add(-24 * time.Hour).Format("2006-01-02")
			curr := time.Now().Format("2006-01-02")
			v, exists := repos[i].recentCommits[curr]
			if exists {
				repos[i].recentCommits[curr] = append(v, commits...)
			} else {
				repos[i].recentCommits[curr] = commits
			}
			// remove commits not under today or yesterday
			for k, _ := range repos[i].recentCommits {
				if k != prev && k != curr {
					delete(repos[i].recentCommits, k)
				}
			}
			mu.Unlock()
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

	// timestamp -> commits
	recentCommits map[string][]*commit
}

func newRepo(repoUrl string) *repo {
	repo := parseRepoName(repoUrl)
	repo.repoUrl = repoUrl
	if repo.remote == "" {
		repo.remote = "origin"
	}
	repo.recentCommits = make(map[string][]*commit, 0)
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
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = r.storagePath()
	out, err := cmd.CombinedOutput()
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
