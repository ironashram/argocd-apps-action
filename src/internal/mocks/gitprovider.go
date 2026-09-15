package mocks

import (
	"context"

	"github.com/ironashram/argocd-apps-action/internal"
)

type MockGitProvider struct {
	ListOpenPRsFunc  func(ctx context.Context) ([]internal.PR, error)
	CreatePRFunc     func(ctx context.Context, p internal.NewPR) (*internal.PR, error)
	UpdatePRFunc     func(ctx context.Context, number int, title, body string) error
	RefreshPRFunc    func(ctx context.Context, number int) error
	ClosePRFunc      func(ctx context.Context, number int, comment string) error
	DeleteBranchFunc func(ctx context.Context, branch string) error
	AddLabelsFunc    func(ctx context.Context, number int, labels []string) error
}

var _ internal.GitProvider = (*MockGitProvider)(nil)

func (m *MockGitProvider) ListOpenPRs(ctx context.Context) ([]internal.PR, error) {
	if m.ListOpenPRsFunc != nil {
		return m.ListOpenPRsFunc(ctx)
	}
	return nil, nil
}

func (m *MockGitProvider) UpdatePR(ctx context.Context, number int, title, body string) error {
	if m.UpdatePRFunc != nil {
		return m.UpdatePRFunc(ctx, number, title, body)
	}
	return nil
}

func (m *MockGitProvider) ClosePR(ctx context.Context, number int, comment string) error {
	if m.ClosePRFunc != nil {
		return m.ClosePRFunc(ctx, number, comment)
	}
	return nil
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

func (m *MockGitProvider) DeleteBranch(ctx context.Context, branch string) error {
	if m.DeleteBranchFunc != nil {
		return m.DeleteBranchFunc(ctx, branch)
	}
	return nil
}

func (m *MockGitProvider) AddLabels(ctx context.Context, number int, labels []string) error {
	if m.AddLabelsFunc != nil {
		return m.AddLabelsFunc(ctx, number, labels)
	}
	return nil
}
