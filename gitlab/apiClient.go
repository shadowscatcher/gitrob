package gitlab

import (
	"fmt"
	"github.com/xanzy/go-gitlab"
	"gitrob/common"
	"strconv"
	"strings"
)

type Client struct {
	apiClient *gitlab.Client
	logger    *common.Logger
}

type projectsGetter func() ([]*gitlab.Project, *gitlab.Response, error)

func NewClient(token string, logger *common.Logger) (*Client, error) {
	c := &Client{}
	var err error

	c.apiClient, err = gitlab.NewClient(token)
	if err != nil {
		return nil, err
	}

	c.apiClient.UserAgent = common.UserAgent
	c.logger = logger
	return c, nil
}

func (c Client) GetUserOrOrganization(login string) (*common.Owner, error) {
	emptyString := gitlab.String("")
	org, orgErr := c.getOrganization(login)
	if orgErr != nil {
		user, userErr := c.getUser(login)
		if userErr != nil {
			return nil, userErr
		}
		id := int64(user.ID)
		return &common.Owner{
			Login:     gitlab.String(user.Username),
			ID:        &id,
			Type:      gitlab.String(common.TargetTypeUser),
			Name:      gitlab.String(user.Name),
			AvatarURL: gitlab.String(user.AvatarURL),
			URL:       gitlab.String(user.WebsiteURL),
			Company:   gitlab.String(user.Organization),
			Blog:      emptyString,
			Location:  emptyString,
			Email:     gitlab.String(user.PublicEmail),
			Bio:       gitlab.String(user.Bio),
		}, nil
	}

	id := int64(org.ID)
	return &common.Owner{
		Login:     gitlab.String(org.Name),
		ID:        &id,
		Type:      gitlab.String(common.TargetTypeOrganization),
		Name:      gitlab.String(org.Name),
		AvatarURL: gitlab.String(org.AvatarURL),
		URL:       gitlab.String(org.WebURL),
		Company:   gitlab.String(org.FullName),
		Blog:      emptyString,
		Location:  emptyString,
		Email:     emptyString,
		Bio:       gitlab.String(org.Description),
	}, nil
}

func (c Client) GetOrganizationMembers(target *common.Owner) ([]*common.Owner, error) {
	var allMembers []*common.Owner
	opt := &gitlab.ListGroupMembersOptions{}
	sID := strconv.FormatInt(*target.ID, 10) // safely downcast an int64 to an int
	for {
		members, resp, err := c.apiClient.Groups.ListAllGroupMembers(sID, opt)
		if err != nil {
			return nil, err
		}
		for _, member := range members {
			id := int64(member.ID)
			allMembers = append(allMembers,
				&common.Owner{
					Login: gitlab.String(member.Username),
					ID:    &id,
					Type:  gitlab.String(common.TargetTypeUser)})
		}
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return allMembers, nil
}

func (c Client) GetRepositoriesFromOwner(target *common.Owner) ([]*common.Repository, error) {
	var allProjects []*common.Repository
	id := int(*target.ID)
	if *target.Type == common.TargetTypeUser {
		userProjects, err := c.getUserProjects(id)
		if err != nil {
			return nil, err
		}
		allProjects = append(allProjects, userProjects...)
	} else {
		groupProjects, err := c.getGroupProjects(target)
		if err != nil {
			return nil, err
		}
		allProjects = append(allProjects, groupProjects...)
	}
	return allProjects, nil
}

func (c Client) getUser(login string) (*gitlab.User, error) {
	users, _, err := c.apiClient.Users.ListUsers(&gitlab.ListUsersOptions{Username: gitlab.String(login)})
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, fmt.Errorf("no GitLab %s or %s %s was found.  If you are targeting a GitLab group, be sure to"+
			" use an ID in place of a name",
			strings.ToLower(common.TargetTypeUser),
			strings.ToLower(common.TargetTypeOrganization),
			login)
	}
	return users[0], err
}

func (c Client) getOrganization(login string) (*gitlab.Group, error) {
	id, err := strconv.Atoi(login)
	if err != nil {
		return nil, err
	}
	org, _, err := c.apiClient.Groups.GetGroup(id)
	if err != nil {
		return nil, err
	}
	return org, err
}

func (c Client) getUserProjects(id int) ([]*common.Repository, error) {
	listUserProjectsOps := &gitlab.ListProjectsOptions{}
	getter := func() ([]*gitlab.Project, *gitlab.Response, error) {
		return c.apiClient.Projects.ListUserProjects(id, listUserProjectsOps)
	}
	increasePage := func(page int) { listUserProjectsOps.Page = page }
	return c.getProjects(getter, increasePage)
}

func (c Client) getGroupProjects(target *common.Owner) ([]*common.Repository, error) {
	listGroupProjectsOps := &gitlab.ListGroupProjectsOptions{}
	id := strconv.FormatInt(*target.ID, 10)
	getter := func() ([]*gitlab.Project, *gitlab.Response, error) {
		return c.apiClient.Groups.ListGroupProjects(id, listGroupProjectsOps)
	}
	increasePage := func(page int) { listGroupProjectsOps.Page = page }
	return c.getProjects(getter, increasePage)
}

func (c Client) getProjects(getter projectsGetter, increasePage func(int)) ([]*common.Repository, error) {
	var allGroupProjects []*common.Repository
	for {
		projects, response, err := getter()
		if err != nil {
			return nil, err
		}
		for _, project := range projects {
			// don't capture forks
			if project.ForkedFromProject == nil {
				id := int64(project.ID)
				p := common.Repository{
					Owner:         gitlab.String(project.Namespace.FullPath),
					ID:            &id,
					Name:          gitlab.String(project.Name),
					FullName:      gitlab.String(project.NameWithNamespace),
					CloneURL:      gitlab.String(project.HTTPURLToRepo),
					URL:           gitlab.String(project.WebURL),
					DefaultBranch: gitlab.String(project.DefaultBranch),
					Description:   gitlab.String(project.Description),
					Homepage:      gitlab.String(project.WebURL),
				}
				allGroupProjects = append(allGroupProjects, &p)
			}
		}
		if response.NextPage == 0 {
			break
		}
		increasePage(response.NextPage)
	}

	return allGroupProjects, nil
}
