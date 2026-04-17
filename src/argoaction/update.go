package argoaction

import (
	"context"
	"fmt"

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
	gitOps, err := internal.OpenRepo(cfg.Workspace)
	if err != nil {
		return fmt.Errorf("opening repository: %w", err)
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: cfg.Token},
	)
	tc := oauth2.NewClient(ctx, ts)

	githubClient := github.NewClient(tc)
	realClient := &internal.RealGitHubClient{Client: githubClient}

	u := &Updater{
		GitOps:       gitOps,
		GitHubClient: realClient,
		Config:       cfg,
		Action:       action,
	}

	err = u.CheckForUpdates(ctx)
	if err != nil {
		return fmt.Errorf("checking for updates: %w", err)
	}

	return nil
}
