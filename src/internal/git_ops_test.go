package internal

import (
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/stretchr/testify/mock"
)

func TestGitRepo_Push(t *testing.T) {
	mockRepo := &MockGitRepo{}
	options := &git.PushOptions{}
	mockRepo.On("Push", options).Return(nil)

	err := mockRepo.Push(options)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	mockRepo.AssertExpectations(t)
}

func TestGitRepo_SetReference(t *testing.T) {
	mockRepo := &MockGitRepo{}
	name := "ref"
	ref := &plumbing.Reference{}
	mockRepo.On("SetReference", name, ref).Return(nil)

	err := mockRepo.SetReference(name, ref)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	mockRepo.AssertExpectations(t)
}

func TestGitRepo_Worktree(t *testing.T) {
	mockRepo := new(MockGitRepo)
	mockWorktree := new(MockWorktree)
	mockRepo.On("Worktree").Return(mockWorktree, nil)

	_, err := mockRepo.Worktree()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	mockRepo.AssertExpectations(t)
	mockWorktree.AssertExpectations(t)
}

func TestGitRepo_Head(t *testing.T) {
	mockRepo := &MockGitRepo{}
	ref := plumbing.NewHashReference(plumbing.ReferenceName("HEAD"), plumbing.NewHash("aefdd0f"))
	mockRepo.On("Head").Return(ref, nil)

	_, err := mockRepo.Head()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	mockRepo.AssertExpectations(t)
}

func TestGitRepo_Storer(t *testing.T) {
	mockRepo := &MockGitRepo{}
	mockRepo.On("Storer").Return(memory.NewStorage())

	storer := mockRepo.Storer()
	if storer == nil {
		t.Errorf("Expected non-nil storer")
	}

	mockRepo.AssertExpectations(t)
}

func TestGitRepo_IterReferences(t *testing.T) {
	mockRepo := &MockGitRepo{}
	iter := &MockReferenceIter{}
	mockRepo.On("IterReferences").Return(iter, nil)

	_, err := mockRepo.IterReferences()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	mockRepo.AssertExpectations(t)
}

func TestGitRepo_PlainOpen(t *testing.T) {
	mockRepo := &MockGitRepo{}
	repo, _ := git.Init(memory.NewStorage(), nil)
	mockRepo.On("PlainOpen", mock.AnythingOfType("string")).Return(repo, nil)

	_, err := mockRepo.PlainOpen("dummyPath")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	mockRepo.AssertExpectations(t)
}
