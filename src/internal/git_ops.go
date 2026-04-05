package internal

import (
	"path/filepath"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/storer"
	"github.com/go-git/go-git/v6/storage"
)

type GitOperations interface {
	Push(*git.PushOptions) error
	SetReference(name string, ref *plumbing.Reference) error
	Worktree() (WorktreeOperations, error)
	Head() (*plumbing.Reference, error)
	Storer() storage.Storer
	IterReferences() (storer.ReferenceIter, error)
}

type GitRepo struct {
	Repo *git.Repository
}

func (r *GitRepo) Push(options *git.PushOptions) error {
	return r.Repo.Push(options)
}

func (r *GitRepo) SetReference(name string, ref *plumbing.Reference) error {
	return r.Repo.Storer.SetReference(ref)
}

func (r *GitRepo) Worktree() (WorktreeOperations, error) {
	worktree, err := r.Repo.Worktree()
	if err != nil {
		return nil, err
	}
	root, err := filepath.Abs(worktree.Filesystem.Root())
	if err != nil {
		return nil, err
	}
	return &WorktreeWithRoot{worktree, root}, nil
}

func (r *GitRepo) Head() (*plumbing.Reference, error) {
	return r.Repo.Head()
}

func (r *GitRepo) Storer() storage.Storer {
	return r.Repo.Storer
}

func (r *GitRepo) IterReferences() (storer.ReferenceIter, error) {
	return r.Repo.Storer.IterReferences()
}

type WorktreeOperations interface {
	Checkout(opts *git.CheckoutOptions) error
	Add(path string) (plumbing.Hash, error)
	Commit(message string, opts *git.CommitOptions) (plumbing.Hash, error)
	Root() (string, error)
}

type WorktreeWithRoot struct {
	*git.Worktree
	root string
}

func (w *WorktreeWithRoot) Root() (string, error) {
	return w.root, nil
}
