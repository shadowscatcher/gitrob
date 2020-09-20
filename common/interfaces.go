package common

import "gopkg.in/src-d/go-git.v4"

type VcsHostingClient interface {
	GetUserOrOrganization(login string) (*Owner, error)
	GetRepositoriesFromOwner(target *Owner) ([]*Repository, error)
	GetOrganizationMembers(target *Owner) ([]*Owner, error)
}

type Cloner interface {
	CloneRepository(cloneConfig CloneConfiguration) (*git.Repository, string, error)
}
