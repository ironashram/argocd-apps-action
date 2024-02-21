package updater

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"time"

	"net/http"

	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/ironashram/argocd-apps-action/internal"

	"github.com/Masterminds/semver"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/google/go-github/v59/github"

	"golang.org/x/oauth2"
	"gopkg.in/yaml.v2"
)

var readAndParseYAML = func(path string) (*internal.Application, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var app internal.Application
	err = yaml.Unmarshal(data, &app)
	if err != nil {
		return nil, err
	}

	return &app, nil
}

var getHTTPResponse = func(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP request failed with status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

var processFile = func(path string, repo *git.Repository, githubClient *github.Client, cfg *internal.Config, action internal.ActionInterface) error {
	app, err := readAndParseYAML(path)
	if err != nil {
		return err
	}

	chart := app.Spec.Source.Chart
	url := app.Spec.Source.RepoURL + "/index.yaml"
	targetRevision := app.Spec.Source.TargetRevision

	if chart == "" || url == "" || targetRevision == "" {
		action.Debugf("Skipping invalid application manifest %s\n", path)
		return nil
	}

	action.Debugf("Checking %s from %s, current version is %s\n", chart, url, targetRevision)

	body, err := getHTTPResponse(url)
	if err != nil {
		return err
	}

	var index internal.Index
	err = yaml.Unmarshal(body, &index)
	if err != nil {
		return err
	}

	if _, ok := index.Entries[chart]; !ok || len(index.Entries[chart]) == 0 {
		action.Debugf("Chart entry %s does not exist or is empty at %s\n", chart, url)
		return nil
	}

	newest, err := getNewestVersion(targetRevision, index.Entries)
	if err != nil {
		action.Debugf("Error comparing versions: %v\n", err)
		return err
	}

	if newest != nil {
		action.Debugf("There is a newer %s version: %s\n", chart, newest)

		if cfg.CreatePr {
			branchName := "update-" + chart
			err = createNewBranch(repo, branchName)
			if err != nil {
				return err
			}

			app.Spec.Source.TargetRevision = newest.String()
			newData, err := yaml.Marshal(app)
			if err != nil {
				return err
			}

			err = os.WriteFile(path, newData, 0644)
			if err != nil {
				return err
			}

			commitMessage := "Update " + chart + " to version " + newest.String()
			err = commitChanges(repo, path, commitMessage)
			if err != nil {
				return err
			}

			err = pushChanges(repo, branchName, cfg)
			if err != nil {
				return err
			}

			prTitle := "Update " + chart + " to version " + newest.String()
			prBody := "This PR updates " + chart + " to version " + newest.String()
			err = createPullRequest(githubClient, cfg.TargetBranch, branchName, prTitle, prBody, action)
			if err != nil {
				return err
			}
		} else {
			action.Debugf("Create PR is disabled, skipping PR creation for %s\n", chart)
		}
	} else {
		action.Debugf("No newer version of %s is available\n", chart)
	}
	return nil
}

var createNewBranch = func(repo *git.Repository, branchName string) error {
	headRef, err := repo.Head()
	if err != nil {
		return err
	}

	newBranchRefName := plumbing.NewBranchReferenceName(branchName)
	err = repo.Storer.SetReference(plumbing.NewHashReference(newBranchRefName, headRef.Hash()))
	if err != nil {
		return err
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return err
	}

	err = worktree.Checkout(&git.CheckoutOptions{
		Branch: newBranchRefName,
	})
	if err != nil {
		return err
	}

	return nil
}

var commitChanges = func(repo *git.Repository, path string, commitMessage string) error {
	worktree, err := repo.Worktree()
	if err != nil {
		return err
	}

	_, err = worktree.Add(path)
	if err != nil {
		return err
	}

	_, err = worktree.Commit(commitMessage, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "github-actions[bot]",
			Email: "41898282+github-actions[bot]@users.noreply.github.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		return err
	}
	return nil
}

var pushChanges = func(repo *git.Repository, branchName string, cfg *internal.Config) error {
	err := repo.Push(&git.PushOptions{
		Auth: &githttp.BasicAuth{
			Username: "github-actions[bot]",
			Password: cfg.Token,
		},
		RefSpecs: []config.RefSpec{config.RefSpec(branchName + ":" + branchName)},
	})
	if err != nil {
		return err
	}
	return nil
}

var createPullRequest = func(githubClient *github.Client, baseBranch string, newBranch string, title string, body string, action internal.ActionInterface) error {

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: action.GetInput("token")},
	)
	ctx = context.WithValue(ctx, oauth2.HTTPClient, &http.Client{})
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	newPR := &github.NewPullRequest{
		Title:               github.String(title),
		Head:                github.String(newBranch),
		Base:                github.String(baseBranch),
		Body:                github.String(body),
		MaintainerCanModify: github.Bool(true),
	}

	_, _, err := client.PullRequests.Create(ctx, action.GetInput("owner"), action.GetInput("repo"), newPR)
	if err != nil {
		return err
	}

	return nil
}

var checkForUpdates = func(repo *git.Repository, githubClient *github.Client, cfg *internal.Config, action internal.ActionInterface) error {
	dir := path.Join(cfg.Workspace, cfg.AppsFolder)

	var walkErr error
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if filepath.Ext(path) == ".yaml" {
			err := processFile(path, repo, githubClient, cfg, action)
			if err != nil {
				return err
			}
		}

		return nil
	})

	return walkErr
}

var getNewestVersion = func(targetVersion string, entries map[string][]struct {
	Version string `yaml:"version"`
}) (*semver.Version, error) {
	target, err := semver.NewVersion(targetVersion)
	if err != nil {
		return nil, err
	}

	var newest *semver.Version
	for _, entry := range entries {
		for _, version := range entry {
			upstream, err := semver.NewVersion(version.Version)
			if err != nil {
				return nil, err
			}

			if target.LessThan(upstream) {
				if newest == nil || newest.LessThan(upstream) {
					newest = upstream
				}
			}
		}
	}

	return newest, nil
}

func StartUpdate(ctx context.Context, cfg *internal.Config, action internal.ActionInterface) error {

	repoPath := path.Join(cfg.Workspace)

	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		action.Fatalf("error: %v", err)
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: cfg.Token},
	)
	ctx = context.WithValue(ctx, oauth2.HTTPClient, &http.Client{})
	tc := oauth2.NewClient(ctx, ts)

	githubClient := github.NewClient(tc)

	err = checkForUpdates(repo, githubClient, cfg, action)
	if err != nil {
		action.Fatalf("error: %v", err)
	}

	return nil
}
