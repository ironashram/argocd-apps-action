package main

import (
	"context"

	"github.com/ironashram/argocd-apps-action/config"
	"github.com/ironashram/argocd-apps-action/internal"
	"github.com/ironashram/argocd-apps-action/argoaction"
)

func main() {
	ctx := context.Background()
	action := internal.NewGithubActionInterface()

	cfg, err := config.NewFromInputs(action)
	if err != nil {
		action.Fatalf("Error parsing inputs: %v", err)
	}

	err = argoaction.StartUpdate(ctx, cfg, action)
	if err != nil {
		action.Fatalf("Error starting action: %v", err)
	}
}
