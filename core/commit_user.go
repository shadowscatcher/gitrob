package core

import (
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"sync"
	"time"
)

type UserSignature struct {
	Role     string    `json:"role"`
	Username string    `json:"username"`
	Email    string    `json:"email"`
	URL      string    `json:"url"`
	When     time.Time `json:"when"`
}

type UniqueSignatures struct {
	signatures map[string]UserSignature
	lock       sync.Locker
}

func NewUniqueSignatures(locker sync.Locker) *UniqueSignatures {
	if locker == nil {
		locker = &sync.Mutex{}
	}

	return &UniqueSignatures{
		signatures: make(map[string]UserSignature),
		lock:       locker,
	}
}

func (u *UniqueSignatures) AddCommit(commit *object.Commit, url string) {
	u.lock.Lock()
	defer u.lock.Unlock()

	u.addSignature(commit.Author, url, "author")
	if !credsEqual(commit) {
		u.addSignature(commit.Committer, url, "committer")
	}
}

func (u *UniqueSignatures) UniqueSignatures() []UserSignature {
	u.lock.Lock()
	defer u.lock.Unlock()

	sigs := make([]UserSignature, len(u.signatures))
	i := 0
	for _, signature := range u.signatures {
		sigs[i] = signature
		i++
	}
	return sigs
}

func (u *UniqueSignatures) exists(signatureID string) bool {
	_, ok := u.signatures[signatureID]
	return ok
}

func (u *UniqueSignatures) addSignature(sig object.Signature, url, role string) {
	if sig.Email == "" && sig.Name == "" {
		return
	}

	id := sig.String()
	if !u.exists(id) {
		u.signatures[id] = UserSignature{
			Role:     role,
			Username: sig.Name,
			Email:    sig.Email,
			URL:      url,
			When:     sig.When,
		}
	}
}

func credsEqual(commit *object.Commit) bool {
	return commit.Author.Name == commit.Committer.Name && commit.Author.Email == commit.Committer.Email
}
