package matching

import (
	"crypto/sha1" //nolint:gosec
	"fmt"
	"gitrob/common"
	"io"
)

type Finding struct {
	ID                          string
	FilePath                    string
	Action                      string
	FileSignatureDescription    string
	FileSignatureComment        string
	ContentSignatureDescription string
	ContentSignatureComment     string
	RepositoryOwner             string
	RepositoryName              string
	CommitHash                  string
	CommitMessage               string
	CommitAuthor                string
	FileURL                     string
	CommitURL                   string
	RepositoryURL               string
	CloneURL                    string
}

func (f *Finding) setupUrls(isGithubSession bool) {
	if isGithubSession {
		f.RepositoryURL = fmt.Sprintf("https://github.com/%s/%s", f.RepositoryOwner, f.RepositoryName)
		f.FileURL = fmt.Sprintf("%s/blob/%s/%s", f.RepositoryURL, f.CommitHash, f.FilePath)
		f.CommitURL = fmt.Sprintf("%s/commit/%s", f.RepositoryURL, f.CommitHash)
	} else {
		results := common.CleanURLSpaces(f.RepositoryOwner, f.RepositoryName)
		f.RepositoryURL = fmt.Sprintf("https://gitlab.com/%s/%s", results[0], results[1])
		f.FileURL = fmt.Sprintf("%s/blob/%s/%s", f.RepositoryURL, f.CommitHash, f.FilePath)
		f.CommitURL = fmt.Sprintf("%s/commit/%s", f.RepositoryURL, f.CommitHash)
	}
}

func (f *Finding) generateID() error {
	h := sha1.New() //nolint:gosec

	for _, s := range []string{
		f.FilePath,
		f.Action,
		f.RepositoryOwner,
		f.RepositoryName,
		f.CommitHash,
		f.CommitMessage,
		f.CommitAuthor,
	} {
		_, err := io.WriteString(h, s)
		if err != nil {
			return err
		}
	}

	f.ID = fmt.Sprintf("%x", h.Sum(nil))
	return nil
}

func (f *Finding) Initialize(isGithubSession bool) error {
	f.setupUrls(isGithubSession)
	return f.generateID()
}
