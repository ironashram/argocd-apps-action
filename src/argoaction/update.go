package argoaction

import (
	"context"
	"fmt"

	"github.com/ironashram/argocd-apps-action/internal"
	"github.com/ironashram/argocd-apps-action/models"
)

type Updater struct {
	GitOps   internal.GitOperations
	Provider internal.GitProvider
	Config   *models.Config
	Action   internal.ActionInterface
}

func StartUpdate(ctx context.Context, cfg *models.Config, action internal.ActionInterface) error {
	gitOps, err := internal.OpenRepo(cfg.Workspace)
	if err != nil {
		return fmt.Errorf("opening repository: %w", err)
	}

	provider := internal.NewRestProvider(cfg.ApiURL, cfg.Owner, cfg.Name, cfg.Token, cfg.Provider)

	u := &Updater{
		GitOps:   gitOps,
		Provider: provider,
		Config:   cfg,
		Action:   action,
	}

	err = u.CheckForUpdates(ctx)
	if err != nil {
		return fmt.Errorf("checking for updates: %w", err)
	}

	return nil
}
