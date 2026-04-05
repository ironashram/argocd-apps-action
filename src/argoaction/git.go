package argoaction

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	githttp "github.com/go-git/go-git/v6/plumbing/transport/http"
	"github.com/ironashram/argocd-apps-action/internal"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/config"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"
	"github.com/google/go-github/v77/github"
)

func (u *Updater) createNewBranch(baseBranch, branchName string) error {
	worktree, err := u.GitOps.Worktree()
	if err != nil {
		return err
	}

	err = worktree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(baseBranch),
	})
	if err != nil {
		return err
	}

	headRef, err := u.GitOps.Head()
	if err != nil {
		return err
	}

	newBranchRefName := plumbing.NewBranchReferenceName(branchName)
	newReference := plumbing.NewHashReference(newBranchRefName, headRef.Hash())
	err = u.GitOps.SetReference(newBranchRefName.String(), newReference)
	if err != nil {
		return fmt.Errorf("failed to create new branch: %w", err)
	}

	worktree, err = u.GitOps.Worktree()
	if err != nil {
		return err
	}

	err = worktree.Checkout(&git.CheckoutOptions{
		Branch: newBranchRefName,
	})
	if err != nil {
		return err
	}

	return nil
}

func (u *Updater) commitChanges(path string, commitMessage string) error {
	worktree, err := u.GitOps.Worktree()
	if err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	basePath, err := worktree.Root()
	if err != nil {
		return fmt.Errorf("failed to get worktree root: %w", err)
	}

	relativePath, err := filepath.Rel(basePath, path)
	if err != nil {
		return fmt.Errorf("failed to get relative path: %w", err)
	}

	_, err = worktree.Add(relativePath)
	if err != nil {
		return err
	}

	_, err = worktree.Commit(commitMessage, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "github-actions[bot]",
			Email: "41898282+github-actions[bot]@users.noreply.github.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		return err
	}
	return nil
}

func (u *Updater) pushChanges(branchName string) error {
	err := u.GitOps.Push(&git.PushOptions{
		Auth: &githttp.BasicAuth{
			Username: "github-actions[bot]",
			Password: u.Config.Token,
		},
		RefSpecs: []config.RefSpec{config.RefSpec("refs/heads/" + branchName + ":refs/heads/" + branchName)},
	})
	if err != nil {
		if strings.Contains(err.Error(), "non-fast-forward update") {
			return fmt.Errorf("branch already exists: %s", branchName)
		}
		return fmt.Errorf("failed to push changes: %w", err)
	}
	return nil
}

func (u *Updater) createPullRequest(baseBranch string, newBranch string, title string, body string) (*github.PullRequest, error) {
	newPR := &github.NewPullRequest{
		Title:               github.Ptr(title),
		Head:                github.Ptr(newBranch),
		Base:                github.Ptr(baseBranch),
		Body:                github.Ptr(body),
		MaintainerCanModify: github.Ptr(true),
	}

	if u.GitHubClient == nil {
		return nil, errors.New("githubClient is nil")
	}

	pullRequests := u.GitHubClient.PullRequests()
	if pullRequests == nil {
		return nil, errors.New("PullRequests is nil")
	}

	pr, _, err := pullRequests.Create(context.Background(), u.Config.Owner, u.Config.Name, newPR)
	if err != nil {
		return pr, err
	}

	return pr, nil
}

func (u *Updater) addLabelsToPullRequest(pr *github.PullRequest, labels []string) error {
	if u.GitHubClient == nil {
		return errors.New("githubClient is nil")
	}

	issues := u.GitHubClient.Issues()
	if issues == nil {
		return errors.New("issues is nil")
	}

	_, _, err := issues.AddLabelsToIssue(context.Background(), u.Config.Owner, u.Config.Name, *pr.Number, labels)
	if err != nil {
		return err
	}

	return nil
}

func (u *Updater) handleNewVersion(chart string, newest *semver.Version, path string, osw internal.OSInterface) error {
	filename := filepath.Base(path)
	filename = strings.TrimSuffix(filename, filepath.Ext(filename))
	branchName := "update-" + chart + "-" + filename + "-" + newest.String()
	err := u.createNewBranch(u.Config.TargetBranch, branchName)
	if err != nil {
		return fmt.Errorf("creating new branch: %w", err)
	}

	err = updateTargetRevision(newest, path, u.Action, osw)
	if err != nil {
		return fmt.Errorf("updating target revision: %w", err)
	}

	commitMessage := "chore: bump " + chart + " to version " + newest.String()
	err = u.commitChanges(path, commitMessage)
	if err != nil {
		if strings.Contains(err.Error(), "cannot create empty commit: clean working tree") {
			u.Action.Infof("No changes to commit for %s, branch already up to date", chart)
			return nil
		}
		return fmt.Errorf("committing changes: %w", err)
	}

	err = u.pushChanges(branchName)
	if err != nil {
		if strings.Contains(err.Error(), "branch already exists") {
			u.Action.Infof("Branch %s already exists, skipping", branchName)
			return nil
		}
		return fmt.Errorf("pushing changes: %w", err)
	}

	prTitle := "chore: bump " + chart + " to version " + newest.String()
	prBody := "This PR updates " + chart + " to version " + newest.String()
	pr, err := u.createPullRequest(u.Config.TargetBranch, branchName, prTitle, prBody)
	if err != nil {
		return fmt.Errorf("creating pull request: %w", err)
	}

	labels := u.Config.Labels
	err = u.addLabelsToPullRequest(pr, labels)
	if err != nil {
		return fmt.Errorf("adding labels to pull request: %w", err)
	}

	u.Action.Infof("Pull request created for %s", chart)
	return nil
}
