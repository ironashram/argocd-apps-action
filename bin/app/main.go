package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"net/http"

	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"

	"github.com/Masterminds/semver"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/google/go-github/v33/github"
	"github.com/sethvargo/go-githubactions"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v2"
)

type Application struct {
	Spec struct {
		Source struct {
			Chart          string `yaml:"chart"`
			RepoURL        string `yaml:"repoURL"`
			TargetRevision string `yaml:"targetRevision"`
		} `yaml:"source"`
	} `yaml:"spec"`
}

type Index struct {
	Entries map[string][]struct {
		Version string `yaml:"version"`
	} `yaml:"entries"`
}

func readAndParseYAML(path string) (Application, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Application{}, err
	}

	var app Application
	err = yaml.Unmarshal(data, &app)
	if err != nil {
		return Application{}, err
	}

	return app, nil
}

func getHTTPResponse(url string) ([]byte, error) {
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

func processFile(path string, repo *git.Repository, githubClient *github.Client, baseBranch string, createPr bool) error {
	app, err := readAndParseYAML(path)
	if err != nil {
		return err
	}

	chart := app.Spec.Source.Chart
	url := app.Spec.Source.RepoURL + "/index.yaml"
	targetRevision := app.Spec.Source.TargetRevision

	if chart == "" || url == "" || targetRevision == "" {
		fmt.Printf("Skipping invalid application manifest %s\n", path)
		return nil
	}

	fmt.Printf("Checking %s from %s, current version is %s\n", chart, url, targetRevision)

	body, err := getHTTPResponse(url)
	if err != nil {
		return err
	}

	var index Index
	err = yaml.Unmarshal(body, &index)
	if err != nil {
		return err
	}

	if _, ok := index.Entries[chart]; !ok || len(index.Entries[chart]) == 0 {
		fmt.Printf("Chart entry %s does not exist or is empty at %s\n", chart, url)
		return nil
	}

	newest, err := getNewestVersion(targetRevision, index.Entries)
	if err != nil {
		fmt.Printf("Error comparing versions: %v\n", err)
		return err
	}

	if newest != nil {
		fmt.Printf("There is a newer %s version: %s\n", chart, newest)

		if createPr {
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

			err = pushChanges(repo, branchName)
			if err != nil {
				return err
			}

			prTitle := "Update " + chart + " to version " + newest.String()
			prBody := "This PR updates " + chart + " to version " + newest.String()
			err = createPullRequest(githubClient, baseBranch, branchName, prTitle, prBody)
			if err != nil {
				return err
			}
		} else {
			fmt.Printf("Create PR is disabled, skipping PR creation for %s\n", chart)
		}
	} else {
		fmt.Printf("No newer version of %s is available\n", chart)
	}

	return nil
}

func checkForUpdates(dir string, repo *git.Repository, githubClient *github.Client, baseBranch string, createPr bool) error {
	dir = filepath.Clean(dir)

	var walkErr error
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if filepath.Ext(path) == ".yaml" {
			err := processFile(path, repo, githubClient, baseBranch, createPr)
			if err != nil {
				return err
			}
		}

		return nil
	})

	return walkErr
}

func getNewestVersion(targetVersion string, entries map[string][]struct {
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

func createNewBranch(repo *git.Repository, branchName string) error {
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

func commitChanges(repo *git.Repository, path string, commitMessage string) error {
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

func pushChanges(repo *git.Repository, branchName string) error {
	err := repo.Push(&git.PushOptions{
		Auth: &githttp.BasicAuth{
			Username: "github-actions[bot]",
			Password: os.Getenv("GITHUB_TOKEN"),
		},
		RefSpecs: []config.RefSpec{config.RefSpec(branchName + ":" + branchName)},
	})
	if err != nil {
		return err
	}

	return nil
}

func createPullRequest(githubClient *github.Client, baseBranch string, newBranch string, title string, body string) error {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")},
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

	_, _, err := client.PullRequests.Create(ctx, "your-username", "your-repo", newPR)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	targetBranch := os.Args[1]
	createPrStr := os.Args[2]
	appsFolder := os.Args[3]

	createPr, err := strconv.ParseBool(createPrStr)
	if err != nil {
		fmt.Println("Error parsing createPr:", err)
		return
	}

	fmt.Println("Target Branch: ", targetBranch)
	fmt.Println("Create PR: ", createPr)
	fmt.Println("Apps Folder: ", appsFolder)
	
	repoName := strings.Split(os.Getenv("GITHUB_REPOSITORY"), "/")[1]
	repoPath := path.Join(os.Getenv("GITHUB_WORKSPACE"), repoName)

	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		githubactions.Fatalf("error: %v", err)
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")},
	)
	ctx = context.WithValue(ctx, oauth2.HTTPClient, &http.Client{})
	tc := oauth2.NewClient(ctx, ts)

	githubClient := github.NewClient(tc)

	err = checkForUpdates(appsFolder, repo, githubClient, targetBranch, createPr)
	if err != nil {
		githubactions.Fatalf("error: %v", err)
	}
}
