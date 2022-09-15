package util

import (
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/storage/memory"
)

func GetLatestCommit(url string) (string, error) {
	r, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
		URL: url,
	})

	if err != nil {
		return "", err
	}

	ref, _ := r.Head()
	commit, err := r.CommitObject(ref.Hash())

	if err != nil {
		return "", err
	}

	return commit.Hash.String(), nil
}
