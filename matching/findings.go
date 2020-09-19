package matching

import (
	"crypto/sha1" //nolint:gosec
	"fmt"
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

func (f *Finding) GenerateID() (string, error) {
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
			return "", err
		}
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
