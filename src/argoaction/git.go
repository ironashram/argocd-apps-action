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

func (u *Updater) findExistingPR(branchName string) *internal.PR {
	for i := range u.openPRs {
		if u.openPRs[i].HeadRef == branchName {
			return &u.openPRs[i]
		}
	}
	return nil
}

// Branches this action owns are "<prefix><chart>-<version>", so a candidate is
// one of ours only when the remainder after the chart name parses as a version.
// Without that check the prefix of "prometheus" also matches a branch opened for
// "prometheus-operator".
func supersededVersion(headRef, prefix, chart string, newest *semver.Version) *semver.Version {
	rest, found := strings.CutPrefix(headRef, prefix+chart+"-")
	if !found {
		return nil
	}
	v, err := semver.NewVersion(rest)
	if err != nil || !v.LessThan(newest) {
		return nil
	}
	return v
}

func (u *Updater) closeSupersededPRs(ctx context.Context, prefix, chart string, newest *semver.Version, keep int) {
	for _, pr := range u.openPRs {
		if pr.Number == keep {
			continue
		}
		old := supersededVersion(pr.HeadRef, prefix, chart, newest)
		if old == nil {
			continue
		}
		comment := fmt.Sprintf("Superseded by #%d, which bumps %s to %s.", keep, chart, newest)
		if err := u.Provider.ClosePR(ctx, pr.Number, comment); err != nil {
			u.Action.Infof("PR #%d could not be closed: %v", pr.Number, err)
			continue
		}
		u.Action.Infof("PR #%d (%s %s) closed, superseded by #%d", pr.Number, chart, old, keep)

		if !u.Config.DeleteBranch {
			continue
		}
		if err := u.Provider.DeleteBranch(ctx, pr.HeadRef); err != nil {
			u.Action.Infof("Branch %s could not be deleted: %v", pr.HeadRef, err)
			continue
		}
		u.Action.Infof("Branch %s deleted", pr.HeadRef)
	}
}

func (u *Updater) handleChartGroup(ctx context.Context, key models.ChartRef, newest *semver.Version, files []models.AppFile, osw internal.OSInterface) error {
	chart := key.Chart
	prefix := "update-"
	summary := "chore: bump " + chart + " to version " + newest.String()
	if u.Config.Scope != "" {
		prefix += u.Config.Scope + "-"
		summary += " (" + u.Config.Scope + ")"
	}
	branchName := prefix + chart + "-" + newest.String()

	scan := u.scanChartPins(ctx, key, newest.String(), files)
	summary += scan.titleSuffix()
	body := buildPRBody(chart, newest, files, u.Config.Workspace) + scan.bodySection(chart, newest.String())

	if existing := u.findExistingPR(branchName); existing != nil {
		err := u.Provider.RefreshPR(ctx, existing.Number)
		switch {
		case err == nil:
			u.Action.Infof("PR #%d refreshed against %s", existing.Number, u.Config.TargetBranch)
		case errors.Is(err, internal.ErrPRUpToDate):
			u.Action.Infof("PR #%d already up to date with %s", existing.Number, u.Config.TargetBranch)
		default:
			u.Action.Infof("PR #%d refresh failed: %v", existing.Number, err)
		}
		if err := u.Provider.UpdatePR(ctx, existing.Number, summary, body); err != nil {
			u.Action.Infof("PR #%d could not be rewritten: %v", existing.Number, err)
		}
		u.closeSupersededPRs(ctx, prefix, chart, newest, existing.Number)
		return nil
	}

	err := u.createNewBranch(u.Config.TargetBranch, branchName)
	if err != nil {
		return fmt.Errorf("creating new branch: %w", err)
	}

	paths := make([]string, 0, len(files))
	for _, f := range files {
		if err := u.updateVersion(f, newest, osw); err != nil {
			return fmt.Errorf("updating version for %s: %w", f.Path, err)
		}
		paths = append(paths, f.Path)
	}

	err = u.commitChanges(paths, summary)
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

	prBody := buildPRBody(chart, newest, files, u.Config.Workspace)
	pr, err := u.createPullRequest(ctx, u.Config.TargetBranch, branchName, summary, prBody)
	if err != nil {
		return fmt.Errorf("creating pull request: %w", err)
	}

	labels := u.Config.Labels
	err = u.addLabelsToPullRequest(ctx, pr, labels)
	if err != nil {
		return fmt.Errorf("adding labels to pull request: %w", err)
	}

	u.Action.Infof("Pull request created for %s (%d file(s))", chart, len(files))
	u.closeSupersededPRs(ctx, prefix, chart, newest, pr.Number)
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
