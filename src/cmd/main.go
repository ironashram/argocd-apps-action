package main

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/ironashram/argocd-apps-action/argoaction"
	"github.com/ironashram/argocd-apps-action/config"
	"github.com/ironashram/argocd-apps-action/internal"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

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
