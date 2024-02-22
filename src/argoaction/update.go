package argoaction

import (
	"context"
	"path"

	"net/http"

	"github.com/ironashram/argocd-apps-action/internal"
	"github.com/ironashram/argocd-apps-action/models"

	"github.com/go-git/go-git/v5"
	"github.com/google/go-github/v59/github"

	"golang.org/x/oauth2"
)

func StartUpdate(ctx context.Context, cfg *models.Config, action internal.ActionInterface) error {

	repoPath := path.Join(cfg.Workspace)

	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		action.Fatalf("error: %v", err)
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: cfg.Token},
	)
	ctx = context.WithValue(ctx, oauth2.HTTPClient, &http.Client{})
	tc := oauth2.NewClient(ctx, ts)

	githubClient := github.NewClient(tc)
	realClient := &internal.RealGitHubClient{Client: githubClient}

	err = checkForUpdates(repo, realClient, cfg, action)
	if err != nil {
		action.Fatalf("error: %v", err)
	}

	return nil
}
