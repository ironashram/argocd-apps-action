package argoaction

import (
	"context"
	"fmt"
	"time"

	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/ironashram/argocd-apps-action/internal"
	"github.com/ironashram/argocd-apps-action/models"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/google/go-github/v59/github"
)

var createNewBranch = func(gitOps internal.GitOperations, branchName string) error {
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

	worktree, err := gitOps.Worktree()
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

	_, err = worktree.Add(path)
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
		RefSpecs: []config.RefSpec{config.RefSpec(branchName + ":" + branchName)},
	})
	if err != nil {
		return fmt.Errorf("failed to push changes: %w", err)
	}
	return nil
}

var createPullRequest = func(githubClient internal.GitHubClient, baseBranch string, newBranch string, title string, body string, action internal.ActionInterface) error {

	newPR := &github.NewPullRequest{
		Title:               github.String(title),
		Head:                github.String(newBranch),
		Base:                github.String(baseBranch),
		Body:                github.String(body),
		MaintainerCanModify: github.Bool(true),
	}

	_, _, err := githubClient.PullRequests().Create(context.Background(), action.GetInput("owner"), action.GetInput("repo"), newPR)
	if err != nil {
		return err
	}

	return nil
}
