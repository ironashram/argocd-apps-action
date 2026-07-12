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

	apiURL := action.Getenv("GITHUB_API_URL")
	if strings.TrimSpace(apiURL) == "" {
		apiURL = "https://api.github.com"
	}
	provider := strings.TrimSpace(action.GetInput("provider"))
	if provider == "" {
		provider = "auto"
	}
	preset := strings.TrimSpace(action.GetInput("preset"))
	if preset == "" {
		preset = "argocd"
	}
	sourcesFile := strings.TrimSpace(action.GetInput("sources_file"))

	var repoCreds []models.RepoCredential
	for _, line := range strings.Split(action.GetInput("repo_credentials"), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 3)
		if len(parts) != 3 || strings.TrimSpace(parts[0]) == "" {
			return nil, fmt.Errorf("repo_credentials line is invalid, expected url-prefix|username|password: %q", line)
		}
		repoCreds = append(repoCreds, models.RepoCredential{
			URLPrefix: strings.TrimSpace(parts[0]),
			Username:  strings.TrimSpace(parts[1]),
			Password:  strings.TrimSpace(parts[2]),
		})
	}

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
	action.Debugf("api_url: %s", apiURL)
	action.Debugf("provider: %s", provider)
	action.Debugf("preset: %s", preset)
	action.Debugf("sources_file: %s", sourcesFile)
	action.Debugf("repo_credentials: %d configured", len(repoCreds))

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
		ApiURL:             apiURL,
		Provider:           provider,
		Preset:             preset,
		SourcesFile:        sourcesFile,
		RepoCreds:          repoCreds,
	}
	return &c, nil
}
