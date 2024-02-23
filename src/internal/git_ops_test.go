package internal

import (
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/go-git/go-git/v5/storage"
	"github.com/go-git/go-git/v5/storage/memory"
)

func TestGitRepo_Push(t *testing.T) {
	mockRepo := &MockGitRepo{
		MockPush: func(options *git.PushOptions) error {
			return nil
		},
	}
	options := &git.PushOptions{}
	err := mockRepo.Push(options)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestGitRepo_SetReference(t *testing.T) {
	mockRepo := &MockGitRepo{
		MockSetReference: func(name string, ref *plumbing.Reference) error {
			return nil
		},
	}
	name := "ref"
	ref := &plumbing.Reference{}
	err := mockRepo.SetReference(name, ref)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestGitRepo_Worktree(t *testing.T) {
	mockRepo := &MockGitRepo{
		MockWorktree: func() (*git.Worktree, error) {
			return nil, nil
		},
	}
	_, err := mockRepo.Worktree()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestGitRepo_Head(t *testing.T) {
	mockRepo := &MockGitRepo{
		MockHead: func() (*plumbing.Reference, error) {
			return nil, nil
		},
	}
	_, err := mockRepo.Head()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestGitRepo_Storer(t *testing.T) {
	mockRepo := &MockGitRepo{
		MockStorer: func() storage.Storer {
			return memory.NewStorage()
		},
	}
	storer := mockRepo.Storer()
	if storer == nil {
		t.Errorf("Expected non-nil storer")
	}
}

func TestGitRepo_IterReferences(t *testing.T) {
	mockRepo := &MockGitRepo{
		MockIterReferences: func() (storer.ReferenceIter, error) {
			return nil, nil
		},
	}
	_, err := mockRepo.IterReferences()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestGitRepo_PlainOpen(t *testing.T) {
	mockRepo := &MockGitRepo{
		MockPlainOpen: func(path string) (*git.Repository, error) {
			return nil, nil
		},
	}
	path := "path/to/repo"
	_, err := mockRepo.PlainOpen(path)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}
