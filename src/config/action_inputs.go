package config

import (
	"fmt"
	"path/filepath"
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
	cleanedAppsFolder := filepath.Clean(appsFolder)
	if filepath.IsAbs(cleanedAppsFolder) || cleanedAppsFolder == ".." || strings.HasPrefix(cleanedAppsFolder, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("apps_folder must be a relative path within the workspace: %q", appsFolder)
	}
	appsFolder = cleanedAppsFolder
	labelsStr := action.GetInput("labels")
	fileExtStr := action.GetInput("file_extensions")
	allowRegexFallbackStr := action.GetInput("allow_regex_fallback")

	createPr, err := strconv.ParseBool(createPrStr)
	if err != nil {
		return nil, fmt.Errorf("create_pr input is invalid: %w", err)
	}

	skipPreRelease, err := strconv.ParseBool(skipPreReleaseStr)
	if err != nil {
		return nil, fmt.Errorf("skip_prerelease input is invalid: %w", err)
	}

	allowRegexFallback := false
	if strings.TrimSpace(allowRegexFallbackStr) != "" {
		allowRegexFallback, err = strconv.ParseBool(allowRegexFallbackStr)
		if err != nil {
			return nil, fmt.Errorf("allow_regex_fallback input is invalid: %w", err)
		}
	}

	labels := strings.Split(labelsStr, ",")
	for i, label := range labels {
		labels[i] = strings.TrimSpace(label)
	}

	if strings.TrimSpace(fileExtStr) == "" {
		return nil, fmt.Errorf("file_extensions input is empty")
	}
	var fileExtensions []string
	for _, ext := range strings.Split(fileExtStr, ",") {
		ext = strings.TrimSpace(ext)
		if ext == "" {
			return nil, fmt.Errorf("file_extensions input is invalid: %q", fileExtStr)
		}
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		fileExtensions = append(fileExtensions, ext)
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
	action.Debugf("file_extensions: %v", fileExtensions)
	action.Debugf("allow_regex_fallback: %v", allowRegexFallback)

	c := models.Config{
		SkipPreRelease:     skipPreRelease,
		TargetBranch:       targetBranch,
		CreatePr:           createPr,
		AppsFolder:         appsFolder,
		Token:              token,
		Repo:               repo,
		Workspace:          workspace,
		Owner:              owner,
		Name:               name,
		Labels:             labels,
		FileExtensions:     fileExtensions,
		AllowRegexFallback: allowRegexFallback,
	}
	return &c, nil
}
