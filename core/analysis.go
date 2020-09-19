package core

import (
	"fmt"
	"gitrob/common"
	"gitrob/github"
	"gitrob/gitlab"
	"gitrob/matching"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"os"
	"strings"
	"sync"
)

func PrintSessionStats(sess *Session) {
	sess.Out.Infof("\nFindings....: %d\n", sess.Stats.Findings)
	sess.Out.Infof("Files.......: %d\n", sess.Stats.Files)
	sess.Out.Infof("Commits.....: %d\n", sess.Stats.Commits)
	sess.Out.Infof("Repositories: %d\n", sess.Stats.Repositories)
	sess.Out.Infof("Targets.....: %d\n\n", sess.Stats.Targets)
}

func GatherTargets(sess *Session) {
	sess.Stats.Status = StatusGathering
	sess.Out.Importantf("Gathering targets...\n")

	for _, loginOption := range sess.Options.Logins {
		target, err := sess.Client.GetUserOrOrganization(loginOption)
		if err != nil || target == nil {
			sess.Out.Errorf(" Errorf retrieving information on %s: %s\n", loginOption, err)
			continue
		}
		sess.Out.Debugf("%s (ID: %d) type: %s\n", *target.Login, *target.ID, *target.Type)
		sess.AddTarget(target)
		if !*sess.Options.NoExpandOrgs && *target.Type == common.TargetTypeOrganization {
			sess.Out.Debugf("Gathering members of %s (ID: %d)...\n", *target.Login, *target.ID)
			members, err := sess.Client.GetOrganizationMembers(target)
			if err != nil {
				sess.Out.Errorf(" Errorf retrieving members of %s: %s\n", *target.Login, err)
				continue
			}
			for _, member := range members {
				sess.Out.Debugf("Adding organization member %s (ID: %d) to targets\n", *member.Login, *member.ID)
				sess.AddTarget(member)
			}
		}
	}
}

func GatherRepositories(sess *Session) {
	var ch = make(chan *common.Owner, len(sess.Targets))
	var wg sync.WaitGroup
	var threadNum int
	if len(sess.Targets) == 1 {
		threadNum = 1
	} else if len(sess.Targets) <= *sess.Options.Threads {
		threadNum = len(sess.Targets) - 1
	} else {
		threadNum = *sess.Options.Threads
	}
	wg.Add(threadNum)
	sess.Out.Debugf("Threads for repository gathering: %d\n", threadNum)
	for i := 0; i < threadNum; i++ {
		go func() {
			for {
				target, ok := <-ch
				if !ok {
					wg.Done()
					return
				}
				repos, err := sess.Client.GetRepositoriesFromOwner(target)
				if err != nil {
					sess.Out.Errorf(" Failed to retrieve repositories from %s: %s\n", *target.Login, err)
				}
				if len(repos) == 0 {
					continue
				}
				for _, repo := range repos {
					sess.Out.Debugf(" Retrieved repository: %s\n", *repo.CloneURL)
					sess.AddRepository(repo)
				}
				sess.Stats.IncrementTargets()
				sess.Out.Infof(" Retrieved %d %s from %s\n", len(repos), common.Pluralize(len(repos), "repository", "repositories"), *target.Login)
			}
		}()
	}

	for _, target := range sess.Targets {
		ch <- target
	}
	close(ch)
	wg.Wait()
}

func deletePath(path, cloneURL string, threadID int, sess *Session) {
	if path != "" {
		err := os.RemoveAll(path)
		if err != nil {
			sess.Out.Errorf("[THREAD #%d][%s] Unable to delete path %s\n", threadID, cloneURL, path)
		} else {
			sess.Out.Debugf("[THREAD #%d][%s] Deleted clone path %s\n", threadID, cloneURL, path)
		}
	}
}

func createFinding(repo common.Repository, commit *object.Commit, change *object.Change,
	fileSignature matching.FileSignature, contentSignature matching.ContentSignature,
	repositoryURL, commitURL string) (*matching.Finding, error) {
	f := &matching.Finding{
		FilePath:                    common.GetChangePath(change),
		Action:                      common.GetChangeAction(change),
		FileSignatureDescription:    fileSignature.GetDescription(),
		FileSignatureComment:        fileSignature.GetComment(),
		ContentSignatureDescription: contentSignature.GetDescription(),
		ContentSignatureComment:     contentSignature.GetComment(),
		RepositoryOwner:             *repo.Owner,
		RepositoryName:              *repo.Name,
		CommitHash:                  commit.Hash.String(),
		CommitMessage:               strings.TrimSpace(commit.Message),
		CommitAuthor:                commit.Author.String(),
		CloneURL:                    *repo.CloneURL,
		RepositoryURL:               repositoryURL,
		CommitURL:                   commitURL,
	}

	f.FileURL = fmt.Sprintf("%s/blob/%s/%s", f.RepositoryURL, f.CommitHash, f.FilePath)
	id, err := f.GenerateID()
	if err != nil {
		return nil, err
	}
	f.ID = id
	return f, err
}

func matchContent(sess *Session,
	matchTarget matching.MatchTarget,
	repo common.Repository,
	change *object.Change,
	commit *object.Commit,
	repositoryURL, commitURL string,
	fileSignature matching.FileSignature,
	threadID int) {
	content, err := common.GetChangeContent(change)
	if err != nil {
		sess.Out.Errorf("Errorf retrieving content in commit %s, change %s:  %s", commit.String(), change.String(), err)
	}
	matchTarget.Content = content
	sess.Out.Debugf("[THREAD #%d][%s] Matching content in %s...\n", threadID, *repo.CloneURL, commit.Hash)
	for _, contentSignature := range sess.Signatures.ContentSignatures {
		matched, err := contentSignature.Match(matchTarget)
		if err != nil {
			sess.Out.Errorf("Errorf while performing content match with '%s': %s\n", contentSignature.Description, err)
		}
		if !matched {
			continue
		}

		finding, err := createFinding(repo, commit, change, fileSignature, contentSignature, repositoryURL, commitURL)
		if err != nil {
			sess.Out.Errorf("Errorf while performing content match with '%s': %s\n", contentSignature.Description, err)
		} else {
			sess.AddFinding(finding)
		}
	}
}

func findSecrets(sess *Session, repo *common.Repository, commit *object.Commit, changes object.Changes, threadID int,
	repositoryURL, commitURL string) {
	for _, change := range changes {
		path := common.GetChangePath(change)
		matchTarget := matching.NewMatchTarget(path)
		if matchTarget.IsSkippable() {
			sess.Out.Debugf("[THREAD #%d][%s] Skipping %s\n", threadID, *repo.CloneURL, matchTarget.Path)
			continue
		}
		sess.Out.Debugf("[THREAD #%d][%s] Inspecting file: %s...\n", threadID, *repo.CloneURL, matchTarget.Path)

		if *sess.Options.Mode != matching.ModeContentMatch {
			for _, fileSignature := range sess.Signatures.FileSignatures {
				matched, err := fileSignature.Match(matchTarget)
				if err != nil {
					sess.Out.Errorf(fmt.Sprintf("Errorf while performing file match: %s\n", err))
				}
				if !matched {
					continue
				}

				if *sess.Options.Mode == matching.ModeFileMatch {
					finding, err := createFinding(*repo, commit, change, fileSignature,
						matching.ContentSignature{Description: "NA"}, repositoryURL, commitURL)
					if err != nil {
						sess.Out.Errorf(fmt.Sprintf("Errorf while performing file match: %s\n", err))
					} else {
						sess.AddFinding(finding)
					}
				}

				if *sess.Options.Mode == matching.ModeMixed {
					matchContent(sess, matchTarget, *repo, change, commit, repositoryURL, commitURL, fileSignature,
						threadID)
				}

				break
			}
			sess.Stats.IncrementFiles()
		} else {
			matchContent(sess, matchTarget, *repo, change, commit, repositoryURL, commitURL,
				matching.FileSignature{Description: "NA"}, threadID)
			sess.Stats.IncrementFiles()
		}
	}
}

func cloneRepository(sess *Session, repo *common.Repository, threadID int) (*git.Repository, string, error) {
	sess.Out.Debugf("[THREAD #%d][%s] Cloning repository...\n", threadID, *repo.CloneURL)

	cloneConfig := common.CloneConfiguration{
		URL:        repo.CloneURL,
		Branch:     repo.DefaultBranch,
		Depth:      sess.Options.CommitDepth,
		Token:      &sess.GitLab.AccessToken,
		InMemClone: sess.Options.InMemClone,
	}

	var clone *git.Repository
	var path string
	var err error

	if sess.IsGithubSession {
		clone, path, err = github.CloneRepository(&cloneConfig)
	} else {
		userName := "oauth2"
		cloneConfig.Username = &userName
		clone, path, err = gitlab.CloneRepository(&cloneConfig)
	}
	if err != nil {
		if err.Error() != "remote repository is empty" {
			sess.Out.Errorf("Errorf cloning repository %s: %s\n", *repo.CloneURL, err)
		}
		sess.Stats.IncrementRepositories()
		sess.Stats.UpdateProgress(sess.Stats.Repositories, len(sess.Repositories))
		return nil, "", err
	}
	sess.Out.Debugf("[THREAD #%d][%s] Cloned repository to: %s\n", threadID, *repo.CloneURL, path)
	return clone, path, err
}

func getRepositoryHistory(sess *Session, clone *git.Repository, repo *common.Repository, path string, threadID int) (
	[]*object.Commit, error) {
	history, err := common.GetRepositoryHistory(clone)
	if err != nil {
		sess.Out.Errorf("[THREAD #%d][%s] Errorf getting commit history: %s\n", threadID, *repo.CloneURL, err)
		deletePath(path, *repo.CloneURL, threadID, sess)
		sess.Stats.IncrementRepositories()
		sess.Stats.UpdateProgress(sess.Stats.Repositories, len(sess.Repositories))
		return nil, err
	}
	sess.Out.Debugf("[THREAD #%d][%s] Number of commits: %d\n", threadID, *repo.CloneURL, len(history))
	return history, err
}

func AnalyzeRepositories(sess *Session) {
	sess.Stats.Status = StatusAnalyzing
	var ch = make(chan *common.Repository, len(sess.Repositories))
	var wg sync.WaitGroup
	var threadNum int

	if len(sess.Repositories) <= 1 {
		threadNum = 1
	} else if len(sess.Repositories) <= *sess.Options.Threads {
		threadNum = len(sess.Repositories) - 1
	} else {
		threadNum = *sess.Options.Threads
	}

	wg.Add(threadNum)

	sess.Out.Debugf("Threads for repository analysis: %d\n", threadNum)

	sess.Out.Importantf("Analyzing %d %s...\n", len(sess.Repositories),
		common.Pluralize(len(sess.Repositories), "repository", "repositories"))

	for i := 0; i < threadNum; i++ {
		go analyze(i, sess, ch, &wg)
	}
	for _, repo := range sess.Repositories {
		ch <- repo
	}

	close(ch)
	wg.Wait()
}

func analyze(threadID int, sess *Session, ch chan *common.Repository, wg *sync.WaitGroup) {
	for {
		sess.Out.Debugf("[THREAD #%d] Requesting new repository to analyze...\n", threadID)
		repo, ok := <-ch
		if !ok {
			sess.Out.Debugf("[THREAD #%d] No more tasks, marking WaitGroup as done\n", threadID)
			wg.Done()
			return
		}

		clone, path, err := cloneRepository(sess, repo, threadID)
		if err != nil {
			continue
		}

		history, err := getRepositoryHistory(sess, clone, repo, path, threadID)
		if err != nil {
			continue
		}

		repositoryURL := getRepositoryURL(*repo.Owner, *repo.Name, IsGithub)

		for _, commit := range history {
			commitURL := getCommitURL(repositoryURL, commit.Hash.String())
			sess.AddCommitUsers(commit, commitURL)
			changes, _ := common.GetChanges(commit, clone)
			sess.Out.Debugf("[THREAD #%d][%s] Analyzing commit: %s\n", threadID, *repo.CloneURL, commit.Hash)
			sess.Out.Debugf("[THREAD #%d][%s] %s changes in %d\n", threadID, *repo.CloneURL, commit.Hash, len(changes))

			findSecrets(sess, repo, commit, changes, threadID, repositoryURL, commitURL)

			sess.Stats.IncrementCommits()
			sess.Out.Debugf("[THREAD #%d][%s] Done analyzing changes in %s\n", threadID, *repo.CloneURL, commit.Hash)
		}

		sess.Out.Debugf("[THREAD #%d][%s] Done analyzing commits\n", threadID, *repo.CloneURL)
		deletePath(path, *repo.CloneURL, threadID, sess)
		sess.Out.Debugf("[THREAD #%d][%s] Deleted %s\n", threadID, *repo.CloneURL, path)
		sess.Stats.IncrementRepositories()
		sess.Stats.UpdateProgress(sess.Stats.Repositories, len(sess.Repositories))
	}
}

func getRepositoryURL(repositoryOwner, repositoryName string, isGithub bool) string {
	if isGithub {
		return fmt.Sprintf("https://github.com/%s/%s", repositoryOwner, repositoryName)
	}
	results := common.CleanURLSpaces(repositoryOwner, repositoryName)
	return fmt.Sprintf("https://gitlab.com/%s/%s", results[0], results[1])
}

func getCommitURL(repositoryURL, commitHash string) string {
	return fmt.Sprintf("%s/commit/%s", repositoryURL, commitHash)
}
