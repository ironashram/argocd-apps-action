package config

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ironashram/argocd-apps-action/internal"
	"github.com/ironashram/argocd-apps-action/models"
)

func NewFromInputs(action internal.ActionInterface) (*models.Config, error) {
	skipPreReleaseStr := action.GetInput("skip_prerelease")
	targetBranch := action.GetInput("target_branch")
	createPrStr := action.GetInput("create_pr")
	appsFolder := action.GetInput("apps_folder")
	labelsStr := action.GetInput("labels")

	createPr, err := strconv.ParseBool(createPrStr)
	if err != nil {
		return nil, fmt.Errorf("create_pr input is invalid: %w", err)
	}

	skipPreRelease, err := strconv.ParseBool(skipPreReleaseStr)
	if err != nil {
		return nil, fmt.Errorf("skip_prerelease input is invalid: %w", err)
	}

	labels := strings.Split(labelsStr, ",")

	for i, label := range labels {
		labels[i] = strings.TrimSpace(label)
	}

	token := action.Getenv("GITHUB_TOKEN")
	repo := action.Getenv("GITHUB_REPOSITORY")
	workspace := action.Getenv("GITHUB_WORKSPACE")

	parts := strings.Split(repo, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid GITHUB_REPOSITORY: %s", repo)
	}

	owner := parts[0]
	name := parts[1]

	action.Debugf("skip_prerelease: %v", skipPreRelease)
	action.Debugf("target_branch: %s", targetBranch)
	action.Debugf("create_pr: %v", createPr)
	action.Debugf("apps_folder: %s", appsFolder)

	c := models.Config{
		SkipPreRelease: skipPreRelease,
		TargetBranch:   targetBranch,
		CreatePr:       createPr,
		AppsFolder:     appsFolder,
		Token:          token,
		Repo:           repo,
		Workspace:      workspace,
		Owner:          owner,
		Name:           name,
		Labels:         labels,
	}
	return &c, nil
}
