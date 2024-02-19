package main

import (
	"context"

	actionconfig "github.com/ironashram/argocd-apps-action/config"
	actionupdater "github.com/ironashram/argocd-apps-action/updater"
	githubactions "github.com/sethvargo/go-githubactions"
)

func main() {
	ctx := context.Background()
	action := githubactions.New()

	cfg, err := actionconfig.NewFromInputs(action)
	if err != nil {
		action.Fatalf("Error parsing inputs: %v", err)
	}

	err = actionupdater.StartUpdate(ctx, cfg, action)
	if err != nil {
		action.Fatalf("Error starting action: %v", err)
	}
}
