package mocks

import (
	"io"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/storer"
	"github.com/go-git/go-git/v6/storage"
	"github.com/ironashram/argocd-apps-action/internal"
	"github.com/stretchr/testify/mock"
)

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

func (m *MockGitRepo) Worktree() (internal.WorktreeOperations, error) {
	args := m.Called()
	return args.Get(0).(internal.WorktreeOperations), args.Error(1)
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
