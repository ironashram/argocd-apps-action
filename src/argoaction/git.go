package argoaction

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	githttp "github.com/go-git/go-git/v6/plumbing/transport/http"
	"github.com/ironashram/argocd-apps-action/internal"
	"github.com/ironashram/argocd-apps-action/models"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/config"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/client"
	"github.com/go-git/go-git/v6/plumbing/object"
)

func (u *Updater) createNewBranch(baseBranch, branchName string) error {
	worktree, err := u.GitOps.Worktree()
	if err != nil {
		return err
	}

	err = worktree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(baseBranch),
	})
	if err != nil {
		return err
	}

	headRef, err := u.GitOps.Head()
	if err != nil {
		return err
	}

	newBranchRefName := plumbing.NewBranchReferenceName(branchName)
	newReference := plumbing.NewHashReference(newBranchRefName, headRef.Hash())
	err = u.GitOps.SetReference(newBranchRefName.String(), newReference)
	if err != nil {
		return fmt.Errorf("failed to create new branch: %w", err)
	}

	err = worktree.Checkout(&git.CheckoutOptions{
		Branch: newBranchRefName,
	})
	if err != nil {
		return err
	}

	return nil
}

func (u *Updater) commitChanges(paths []string, commitMessage string) error {
	worktree, err := u.GitOps.Worktree()
	if err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	basePath, err := worktree.Root()
	if err != nil {
		return fmt.Errorf("failed to get worktree root: %w", err)
	}

	for _, p := range paths {
		relativePath, err := filepath.Rel(basePath, p)
		if err != nil {
			return fmt.Errorf("failed to get relative path for %s: %w", p, err)
		}
		if _, err := worktree.Add(relativePath); err != nil {
			return fmt.Errorf("failed to stage %s: %w", relativePath, err)
		}
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

func (u *Updater) pushChanges(branchName string) error {
	err := u.GitOps.Push(&git.PushOptions{
		ClientOptions: []client.Option{
			client.WithHTTPAuth(&githttp.BasicAuth{
				Username: "github-actions[bot]",
				Password: u.Config.Token,
			}),
		},
		RefSpecs: []config.RefSpec{config.RefSpec("refs/heads/" + branchName + ":refs/heads/" + branchName)},
	})
	if err != nil {
		if strings.Contains(err.Error(), "non-fast-forward update") {
			return fmt.Errorf("branch already exists: %s", branchName)
		}
		return fmt.Errorf("failed to push changes: %w", err)
	}
	return nil
}

func (u *Updater) createPullRequest(ctx context.Context, baseBranch string, newBranch string, title string, body string) (*internal.PR, error) {
	if u.Provider == nil {
		return nil, errors.New("git provider is nil")
	}

	return u.Provider.CreatePR(ctx, internal.NewPR{
		Title: title,
		Head:  newBranch,
		Base:  baseBranch,
		Body:  body,
	})
}

func (u *Updater) addLabelsToPullRequest(ctx context.Context, pr *internal.PR, labels []string) error {
	if u.Provider == nil {
		return errors.New("git provider is nil")
	}

	return u.Provider.AddLabels(ctx, pr.Number, labels)
}

func (u *Updater) findExistingPR(ctx context.Context, branchName string) (*internal.PR, error) {
	if u.Provider == nil {
		return nil, errors.New("git provider is nil")
	}
	return u.Provider.FindOpenPR(ctx, branchName)
}

func (u *Updater) handleChartGroup(ctx context.Context, chart string, newest *semver.Version, files []models.AppFile, osw internal.OSInterface) error {
	branchName := "update-" + chart + "-" + newest.String()

	existing, err := u.findExistingPR(ctx, branchName)
	if err != nil {
		u.Action.Debugf("Error checking for existing PR: %v", err)
	} else if existing != nil {
		err := u.Provider.RefreshPR(ctx, existing.Number)
		switch {
		case err == nil:
			u.Action.Infof("PR #%d refreshed against %s", existing.Number, u.Config.TargetBranch)
		case errors.Is(err, internal.ErrPRUpToDate):
			u.Action.Infof("PR #%d already up to date with %s", existing.Number, u.Config.TargetBranch)
		default:
			u.Action.Infof("PR #%d refresh failed: %v", existing.Number, err)
		}
		return nil
	}

	err = u.createNewBranch(u.Config.TargetBranch, branchName)
	if err != nil {
		return fmt.Errorf("creating new branch: %w", err)
	}

	paths := make([]string, 0, len(files))
	for _, f := range files {
		if err := updateTargetRevision(newest, f.Path, u.Action, osw); err != nil {
			return fmt.Errorf("updating target revision for %s: %w", f.Path, err)
		}
		paths = append(paths, f.Path)
	}

	commitMessage := "chore: bump " + chart + " to version " + newest.String()
	err = u.commitChanges(paths, commitMessage)
	if err != nil {
		if strings.Contains(err.Error(), "cannot create empty commit: clean working tree") {
			u.Action.Infof("No changes to commit for %s, branch already up to date", chart)
			return nil
		}
		return fmt.Errorf("committing changes: %w", err)
	}

	err = u.pushChanges(branchName)
	if err != nil {
		if strings.Contains(err.Error(), "branch already exists") {
			u.Action.Infof("Branch %s already exists, skipping", branchName)
			return nil
		}
		return fmt.Errorf("pushing changes: %w", err)
	}

	prTitle := "chore: bump " + chart + " to version " + newest.String()
	prBody := buildPRBody(chart, newest, files, u.Config.Workspace)
	pr, err := u.createPullRequest(ctx, u.Config.TargetBranch, branchName, prTitle, prBody)
	if err != nil {
		return fmt.Errorf("creating pull request: %w", err)
	}

	labels := u.Config.Labels
	err = u.addLabelsToPullRequest(ctx, pr, labels)
	if err != nil {
		return fmt.Errorf("adding labels to pull request: %w", err)
	}

	u.Action.Infof("Pull request created for %s (%d file(s))", chart, len(files))
	return nil
}

func buildPRBody(chart string, newest *semver.Version, files []models.AppFile, workspace string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "This PR updates %s to version %s.\n\n", chart, newest)
	fmt.Fprintln(&b, "Files updated:")
	for _, f := range files {
		display := f.Path
		if rel, err := filepath.Rel(workspace, f.Path); err == nil {
			display = rel
		}
		fmt.Fprintf(&b, "- %s (%s → %s)\n", display, f.CurrentVersion, newest)
	}
	return b.String()
}
