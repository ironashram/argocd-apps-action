package internal

import (
	"io"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/go-git/go-git/v5/storage"
	"github.com/stretchr/testify/mock"
)

type GitOperations interface {
	Push(*git.PushOptions) error
	SetReference(name string, ref *plumbing.Reference) error
	Worktree() (WorktreeOperations, error)
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

func (r *GitRepo) PlainOpen(path string) (*git.Repository, error) {
	repo, err := git.PlainOpen(path)
	if err != nil {
		return nil, err
	}
	r.Repo = repo
	return r.Repo, nil
}

type MockGitRepo struct {
	mock.Mock
}

func (m *MockGitRepo) Push(options *git.PushOptions) error {
	args := m.Called(options)
	return args.Error(0)
}

func (m *MockGitRepo) SetReference(name string, ref *plumbing.Reference) error {
	args := m.Called(name, ref)
	return args.Error(0)
}

func (m *MockGitRepo) Worktree() (WorktreeOperations, error) {
	args := m.Called()
	return args.Get(0).(WorktreeOperations), args.Error(1)
}

func (m *MockGitRepo) Head() (*plumbing.Reference, error) {
	args := m.Called()
	return args.Get(0).(*plumbing.Reference), args.Error(1)
}

func (m *MockGitRepo) Storer() storage.Storer {
	args := m.Called()
	return args.Get(0).(storage.Storer)
}

func (m *MockGitRepo) IterReferences() (storer.ReferenceIter, error) {
	args := m.Called()
	return args.Get(0).(storer.ReferenceIter), args.Error(1)
}

func (m *MockGitRepo) PlainOpen(path string) (*git.Repository, error) {
	args := m.Called(path)
	return args.Get(0).(*git.Repository), args.Error(1)
}

type MockReference struct {
	RefName plumbing.ReferenceName
}

func (m *MockReference) Name() plumbing.ReferenceName {
	return m.RefName
}

func (m *MockReference) Hash() plumbing.Hash {
	return plumbing.Hash{}
}

func (m *MockReference) Type() plumbing.ReferenceType {
	return plumbing.HashReference
}

func (m *MockReference) Strings() (string, string) {
	return m.RefName.String(), ""
}

type MockReferenceIter struct{}

func (m *MockReferenceIter) Next() (*plumbing.Reference, error) {
	return nil, io.EOF
}

func (m *MockReferenceIter) ForEach(func(*plumbing.Reference) error) error {
	return io.EOF
}

func (m *MockReferenceIter) Close() {

}

type WorktreeOperations interface {
	Checkout(opts *git.CheckoutOptions) error
	Add(path string) (plumbing.Hash, error)
	Commit(message string, opts *git.CommitOptions) (plumbing.Hash, error)
	Root() (string, error)
}

type MockWorktree struct {
	mock.Mock
}

func (m *MockWorktree) Checkout(opts *git.CheckoutOptions) error {
	args := m.Called(opts)
	return args.Error(0)
}

func (m *MockWorktree) Add(path string) (plumbing.Hash, error) {
	args := m.Called(path)
	return args.Get(0).(plumbing.Hash), args.Error(1)
}

func (m *MockWorktree) Commit(message string, opts *git.CommitOptions) (plumbing.Hash, error) {
	args := m.Called(message, opts)
	return args.Get(0).(plumbing.Hash), args.Error(1)
}

func (m *MockWorktree) Root() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

type WorktreeWithRoot struct {
	*git.Worktree
	root string
}

func (w *WorktreeWithRoot) Root() (string, error) {
	return w.root, nil
}
