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

type Updater struct {
	GitOps       internal.GitOperations
	GitHubClient internal.GitHubClient
	Config       *models.Config
	Action       internal.ActionInterface
}

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

	u := &Updater{
		GitOps:       gitOps,
		GitHubClient: realClient,
		Config:       cfg,
		Action:       action,
	}

	err = u.CheckForUpdates()
	if err != nil {
		return fmt.Errorf("checking for updates: %w", err)
	}

	return nil
}
