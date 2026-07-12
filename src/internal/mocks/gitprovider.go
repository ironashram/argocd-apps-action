package mocks

import (
	"context"

	"github.com/ironashram/argocd-apps-action/internal"
)

type MockGitProvider struct {
	FindOpenPRFunc func(ctx context.Context, headBranch string) (*internal.PR, error)
	CreatePRFunc   func(ctx context.Context, p internal.NewPR) (*internal.PR, error)
	RefreshPRFunc  func(ctx context.Context, number int) error
	AddLabelsFunc  func(ctx context.Context, number int, labels []string) error
}

var _ internal.GitProvider = (*MockGitProvider)(nil)

func (m *MockGitProvider) FindOpenPR(ctx context.Context, headBranch string) (*internal.PR, error) {
	if m.FindOpenPRFunc != nil {
		return m.FindOpenPRFunc(ctx, headBranch)
	}
	return nil, nil
}

func (m *MockGitProvider) CreatePR(ctx context.Context, p internal.NewPR) (*internal.PR, error) {
	if m.CreatePRFunc != nil {
		return m.CreatePRFunc(ctx, p)
	}
	return nil, nil
}

func (m *MockGitProvider) RefreshPR(ctx context.Context, number int) error {
	if m.RefreshPRFunc != nil {
		return m.RefreshPRFunc(ctx, number)
	}
	return nil
}

func (m *MockGitProvider) AddLabels(ctx context.Context, number int, labels []string) error {
	if m.AddLabelsFunc != nil {
		return m.AddLabelsFunc(ctx, number, labels)
	}
	return nil
}
