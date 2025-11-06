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
	"github.com/ironashram/argocd-apps-action/models"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/config"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"
	"github.com/google/go-github/v74/github"
)

var createNewBranch = func(gitOps internal.GitOperations, baseBranch, branchName string) error {
	worktree, err := gitOps.Worktree()
	if err != nil {
		return err
	}

	err = worktree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(baseBranch),
	})
	if err != nil {
		return err
	}

	headRef, err := gitOps.Head()
	if err != nil {
		return err
	}

	newBranchRefName := plumbing.NewBranchReferenceName(branchName)
	newReference := plumbing.NewHashReference(newBranchRefName, headRef.Hash())
	err = gitOps.SetReference(newBranchRefName.String(), newReference)
	if err != nil {
		return fmt.Errorf("failed to create new branch: %w", err)
	}

	worktree, err = gitOps.Worktree()
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

var commitChanges = func(gitOps internal.GitOperations, path string, commitMessage string) error {
	worktree, err := gitOps.Worktree()
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

var pushChanges = func(gitOps internal.GitOperations, branchName string, cfg *models.Config) error {
	err := gitOps.Push(&git.PushOptions{
		Auth: &githttp.BasicAuth{
			Username: "github-actions[bot]",
			Password: cfg.Token,
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

var createPullRequest = func(githubClient internal.GitHubClient, baseBranch string, newBranch string, title string, body string, action internal.ActionInterface, cfg *models.Config) (*github.PullRequest, error) {

	newPR := &github.NewPullRequest{
		Title:               github.String(title),
		Head:                github.String(newBranch),
		Base:                github.String(baseBranch),
		Body:                github.String(body),
		MaintainerCanModify: github.Bool(true),
	}

	if githubClient == nil {
		return nil, errors.New("githubClient is nil")
	}

	pullRequests := githubClient.PullRequests()
	if pullRequests == nil {
		return nil, errors.New("PullRequests is nil")
	}

	if action == nil {
		return nil, errors.New("action is nil")
	}

	pr, _, err := pullRequests.Create(context.Background(), cfg.Owner, cfg.Name, newPR)
	if err != nil {
		return pr, err
	}

	return pr, nil
}

var addLabelsToPullRequest = func(githubClient internal.GitHubClient, pr *github.PullRequest, labels []string, cfg *models.Config) error {
	if githubClient == nil {
		return errors.New("githubClient is nil")
	}

	issues := githubClient.Issues()
	if issues == nil {
		return errors.New("issues is nil")
	}

	_, _, err := issues.AddLabelsToIssue(context.Background(), cfg.Owner, cfg.Name, *pr.Number, labels)
	if err != nil {
		return err
	}

	return nil
}

var handleNewVersion = func(chart string, newest *semver.Version, path string, gitOps internal.GitOperations, cfg *models.Config, action internal.ActionInterface, osw internal.OSInterface, githubClient internal.GitHubClient) error {
	filename := filepath.Base(path)
	filename = strings.TrimSuffix(filename, filepath.Ext(filename))
	branchName := "update-" + chart + "-" + filename + "-" + newest.String()
	err := createNewBranch(gitOps, cfg.TargetBranch, branchName)
	if err != nil {
		action.Fatalf("Error creating new branch: %v", err)
		return err
	}

	err = updateTargetRevision(newest, path, action, osw)
	if err != nil {
		action.Fatalf("Error updating target revision: %v", err)
		return err
	}

	commitMessage := "chore: bump " + chart + " to version " + newest.String()
	err = commitChanges(gitOps, path, commitMessage)
	if err != nil {
		if strings.Contains(err.Error(), "cannot create empty commit: clean working tree") {
			action.Infof("No changes to commit for %s, branch already up to date", chart)
			return nil
		}
		action.Fatalf("Error committing changes: %v", err)
		return err
	}

	err = pushChanges(gitOps, branchName, cfg)
	if err != nil {
		if strings.Contains(err.Error(), "branch already exists") {
			action.Infof("Branch %s already exists, skipping", branchName)
			return nil
		}
		action.Fatalf("Error pushing changes: %v", err)
		return err
	}

	prTitle := "chore: bump " + chart + " to version " + newest.String()
	prBody := "This PR updates " + chart + " to version " + newest.String()
	pr, err := createPullRequest(githubClient, cfg.TargetBranch, branchName, prTitle, prBody, action, cfg)
	if err != nil {
		action.Fatalf("Error creating pull request: %v", err)
		return err
	}

	labels := cfg.Labels
	err = addLabelsToPullRequest(githubClient, pr, labels, cfg)
	if err != nil {
		action.Fatalf("Error adding labels to pull request: %v", err)
	}

	action.Infof("Pull request created for %s", chart)
	return nil
}
