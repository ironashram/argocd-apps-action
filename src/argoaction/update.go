package argoaction

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-git/go-git/v6"
	"github.com/ironashram/argocd-apps-action/internal"
	"github.com/ironashram/argocd-apps-action/models"

	"github.com/google/go-github/v77/github"

	"golang.org/x/oauth2"
)

func StartUpdate(ctx context.Context, cfg *models.Config, action internal.ActionInterface) error {
	repo, err := git.PlainOpen(cfg.Workspace)
	if err != nil {
		return fmt.Errorf("opening repository: %w", err)
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
		return fmt.Errorf("checking for updates: %w", err)
	}

	return nil
}
