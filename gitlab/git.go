package gitlab

import (
	"fmt"
	"gitrob/common"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"
	"gopkg.in/src-d/go-git.v4/storage/memory"
	"io/ioutil"
)

type Cloner struct {
	username string
	token    string
}

func NewCloner(username, token string) *Cloner {
	return &Cloner{
		username: username,
		token:    token,
	}
}

func (c *Cloner) CloneRepository(cloneConfig common.CloneConfiguration) (*git.Repository, string, error) {
	cloneOptions := &git.CloneOptions{
		URL:           cloneConfig.URL,
		Depth:         cloneConfig.Depth,
		ReferenceName: plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", cloneConfig.Branch)),
		SingleBranch:  true,
		Tags:          git.NoTags,
		Auth: &http.BasicAuth{
			Username: c.username,
			Password: c.token,
		},
	}

	var repository *git.Repository
	var err error
	var dir string
	if !cloneConfig.InMemClone {
		dir, err = ioutil.TempDir("", "gitrob")
		if err != nil {
			return nil, "", err
		}
		repository, err = git.PlainClone(dir, false, cloneOptions)
	} else {
		repository, err = git.Clone(memory.NewStorage(), nil, cloneOptions)
	}
	if err != nil {
		return nil, dir, err
	}
	return repository, dir, nil
}
