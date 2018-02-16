package main

import (
	"fmt"
	"strings"
	"os"

	"gopkg.in/src-d/go-billy.v4/memfs"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/storage/memory"
)

const Version = "0.1.0-dev"

func main() {
	fmt.Printf("gitwatch %s\n", Version)

	// TODO(adam): Use a real fs, for caching
	fs := memfs.New()
	storer := memory.NewStorage()

	repo, err := git.Clone(storer, fs, &git.CloneOptions{
		URL: "https://github.com/adamdecaf/cert-manage.git",
		// RemoteName:   "master", // origin?
		SingleBranch:      true,
		// Depth:             100,
		RecurseSubmodules: git.NoRecurseSubmodules,
		Tags:              git.NoTags,
	})
	if err != nil {
		panic(err)
	}

	// This works instead, `main.go` can be any path
	// $ git log --oneline 9bf483aa848fdfb2b625b6b84fedce6d46b32d68...6abb583cc70fc0e6aa83115aa81f63700979f398 -- main.go
	// 6abb583 cmd/version: on -dev builds add git hash and Go runtime version to output

	// just two commits from cert-manage
	// cmd/version: on -dev builds add git hash and Go runtime version to output
	src := "6abb583cc70fc0e6aa83115aa81f63700979f398" // newer
	// Merge pull request #153 from adamdecaf/ui-remove-sha1
	dst := "3007985f34ecebaf5a5dda30911586acab0364dc" // older

	_ = config.RefSpec(fmt.Sprintf("%s:%s", dst, src))

	// TODO(adam): This sees the refs as up to date and won't grab older commits
	err = repo.Fetch(&git.FetchOptions{
		RemoteName: "origin",
		// RefSpecs: []config.RefSpec{
		// 	config.RefSpec("refs/heads/master:refs/remotes/master"),
		// 	// format: <src>:<dst>
		// config.RefSpec(fmt.Sprintf("%s:%s", dst, src)),
		// },
		Progress: os.Stdout,
		Depth: 100, // doesn't work, but .Depth on clone does...
		// Tags: git.NoTags,
		Force: true,
	})
	// err = repo.Fetch(&git.FetchOptions{
	// 	Depth: 100,
	// 	Force: true,
	// })
	fmt.Printf("fetch = %v\n", err)
	if err != nil && err != git.NoErrAlreadyUpToDate {
		panic(err)
	}

	_ = plumbing.NewHash(src)
	// ref := plumbing.NewHashReference(plumbing.HEAD, plumbing.NewHash(src))
	// commit, err := repo.CommitObject(ref.Hash())
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Println(commit)

	// Walk along commits, print each commit sumamry
	commits, err := repo.Log(&git.LogOptions{
		From: plumbing.NewHash("d83526b151a5cb3abd271e874a507ec169f201f3"), // docs: mention 'add' sub-command
		// From: plumbing.NewHash(src),
	})
	if err != nil {
		panic(err)
	}
	defer commits.Close()

	for {
		commit, err := commits.Next()
		if commit == nil {
			break
		}
		if err != nil {
			panic(err)
		}
		if commit.Hash.String() == dst {
			break
		}

		// print first line
		idx := strings.Index(commit.Message, "\n")
		if idx > 0 {
			fmt.Println(commit.Message[:idx])
			continue
		}
		fmt.Println(commit.Message)
	}

}
