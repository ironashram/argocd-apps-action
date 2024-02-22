package config

import (
	"fmt"
	"strconv"

	"github.com/ironashram/argocd-apps-action/internal"
	"github.com/ironashram/argocd-apps-action/models"
)

func NewFromInputs(action internal.ActionInterface) (*models.Config, error) {
	targetBranch := action.GetInput("target_branch")
	createPrStr := action.GetInput("create_pr")
	appsFolder := action.GetInput("apps_folder")

	createPr, err := strconv.ParseBool(createPrStr)
	if err != nil {
		fmt.Println("Error parsing createPr:", err)
		return nil, err
	}

	token := action.Getenv("GITHUB_TOKEN")
	repo := action.Getenv("GITHUB_REPOSITORY")
	workspace := action.Getenv("GITHUB_WORKSPACE")

	action.Debugf("target_branch: %s", targetBranch)
	action.Debugf("create_pr: %v", createPr)
	action.Debugf("apps_folder: %s", appsFolder)

	c := models.Config{
		TargetBranch: targetBranch,
		CreatePr:     createPr,
		AppsFolder:   appsFolder,
		Token:        token,
		Repo:         repo,
		Workspace:    workspace,
	}
	return &c, nil
}
