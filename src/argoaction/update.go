package argoaction

import (
	"context"
	"path"

	"net/http"

	"github.com/go-git/go-git/v5"
	"github.com/ironashram/argocd-apps-action/internal"
	"github.com/ironashram/argocd-apps-action/models"

	"github.com/google/go-github/v66/github"

	"golang.org/x/oauth2"
)

func StartUpdate(ctx context.Context, cfg *models.Config, action internal.ActionInterface) error {

	repoPath := path.Join(cfg.Workspace)

	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		action.Fatalf("error: %v", err)
	}

	gitOps := &internal.GitRepo{Repo: repo}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: cfg.Token},
	)
	ctx = context.WithValue(ctx, oauth2.HTTPClient, &http.Client{})
	tc := oauth2.NewClient(ctx, ts)

	githubClient := github.NewClient(tc)
	realClient := &internal.RealGitHubClient{Client: githubClient}

	err = checkForUpdates(gitOps, realClient, cfg, action)
	if err != nil {
		action.Fatalf("error: %v", err)
	}

	return nil
}
