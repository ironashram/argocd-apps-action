package internal

import (
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/go-git/go-git/v5/storage"
)

type GitOperations interface {
	Push(*git.PushOptions) error
	SetReference(name string, ref *plumbing.Reference) error
	Worktree() (*git.Worktree, error)
	Head() (*plumbing.Reference, error)
	Storer() storage.Storer
	IterReferences() (storer.ReferenceIter, error)
	PlainOpen(string) (*git.Repository, error)
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

func (r *GitRepo) Worktree() (*git.Worktree, error) {
	return r.Repo.Worktree()
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

func (r *GitRepo) PlainOpen(path string) (*git.Repository, error) {
	repo, err := git.PlainOpen(path)
	if err != nil {
		return nil, err
	}
	r.Repo = repo
	return r.Repo, nil
}

type MockGitRepo struct {
	MockPush           func(*git.PushOptions) error
	MockSetReference   func(name string, ref *plumbing.Reference) error
	MockWorktree       func() (*git.Worktree, error)
	MockHead           func() (*plumbing.Reference, error)
	MockStorer         func() storage.Storer
	MockIterReferences func() (storer.ReferenceIter, error)
	MockPlainOpen      func(string) (*git.Repository, error)
}

func (r *MockGitRepo) Push(options *git.PushOptions) error {
	return r.MockPush(options)
}

func (r *MockGitRepo) SetReference(name string, ref *plumbing.Reference) error {
	return r.MockSetReference(name, ref)
}

func (r *MockGitRepo) Worktree() (*git.Worktree, error) {
	return r.MockWorktree()
}

func (r *MockGitRepo) Head() (*plumbing.Reference, error) {
	return r.MockHead()
}

func (r *MockGitRepo) Storer() storage.Storer {
	return r.MockStorer()
}

func (r *MockGitRepo) IterReferences() (storer.ReferenceIter, error) {
	return r.MockIterReferences()
}

func (r *MockGitRepo) PlainOpen(path string) (*git.Repository, error) {
	return r.MockPlainOpen(path)
}
