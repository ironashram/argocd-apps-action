package internal_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ironashram/argocd-apps-action/internal"
	"github.com/stretchr/testify/assert"
)

func newTestProvider(server *httptest.Server, providerHint string) *internal.RestProvider {
	p := internal.NewRestProvider(server.URL, "owner", "repo", "token", providerHint)
	p.Client = server.Client()
	return p
}

func TestResolveRefreshStyle(t *testing.T) {
	assert.Equal(t, internal.RefreshGitHub, internal.ResolveRefreshStyle("github", ""))
	assert.Equal(t, internal.RefreshGitea, internal.ResolveRefreshStyle("forgejo", ""))
	assert.Equal(t, internal.RefreshGitea, internal.ResolveRefreshStyle("gitea", ""))
	assert.Equal(t, internal.RefreshGitea, internal.ResolveRefreshStyle("codeberg", ""))
	assert.Equal(t, internal.RefreshGitHub, internal.ResolveRefreshStyle("", "https://api.github.com"))
	assert.Equal(t, internal.RefreshGitHub, internal.ResolveRefreshStyle("", ""))
	assert.Equal(t, internal.RefreshGitea, internal.ResolveRefreshStyle("", "https://git.example.com/api/v1"))
	assert.Equal(t, internal.RefreshGitea, internal.ResolveRefreshStyle("auto", "https://git.example.com/api/v1"))
}

func TestFindOpenPR(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/repos/owner/repo/pulls", r.URL.Path)
		assert.Equal(t, "open", r.URL.Query().Get("state"))
		assert.Equal(t, "token token", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"number":7,"head":{"ref":"other"}},{"number":9,"head":{"ref":"update-x-1.0.0"}}]`))
	}))
	defer server.Close()

	p := newTestProvider(server, "forgejo")
	pr, err := p.FindOpenPR(context.Background(), "update-x-1.0.0")
	assert.NoError(t, err)
	assert.NotNil(t, pr)
	assert.Equal(t, 9, pr.Number)

	none, err := p.FindOpenPR(context.Background(), "missing")
	assert.NoError(t, err)
	assert.Nil(t, none)
}

func TestCreatePR(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/repos/owner/repo/pulls", r.URL.Path)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"number":42,"head":{"ref":"update-x-1.0.0"}}`))
	}))
	defer server.Close()

	p := newTestProvider(server, "github")
	pr, err := p.CreatePR(context.Background(), internal.NewPR{Title: "t", Head: "update-x-1.0.0", Base: "main", Body: "b"})
	assert.NoError(t, err)
	assert.Equal(t, 42, pr.Number)
}

func TestCreatePR_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"message":"validation failed"}`))
	}))
	defer server.Close()

	p := newTestProvider(server, "github")
	_, err := p.CreatePR(context.Background(), internal.NewPR{Title: "t"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create pull request")
}

func TestRefreshPR_GitHub(t *testing.T) {
	var gotPath, gotMethod string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	p := newTestProvider(server, "github")
	err := p.RefreshPR(context.Background(), 5)
	assert.NoError(t, err)
	assert.Equal(t, http.MethodPut, gotMethod)
	assert.Equal(t, "/repos/owner/repo/pulls/5/update-branch", gotPath)
}

func TestRefreshPR_GitHubUpToDate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
	}))
	defer server.Close()

	p := newTestProvider(server, "github")
	err := p.RefreshPR(context.Background(), 5)
	assert.True(t, errors.Is(err, internal.ErrPRUpToDate))
}

func TestRefreshPR_Forgejo(t *testing.T) {
	var gotPath, gotMethod string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	p := newTestProvider(server, "forgejo")
	err := p.RefreshPR(context.Background(), 8)
	assert.NoError(t, err)
	assert.Equal(t, http.MethodPost, gotMethod)
	assert.Equal(t, "/repos/owner/repo/pulls/8/update", gotPath)
}

func TestRefreshPR_ForgejoConflict(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
	}))
	defer server.Close()

	p := newTestProvider(server, "forgejo")
	err := p.RefreshPR(context.Background(), 8)
	assert.True(t, errors.Is(err, internal.ErrPRUpToDate))
}

func TestAddLabels(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		assert.Equal(t, "/repos/owner/repo/issues/3/labels", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	p := newTestProvider(server, "forgejo")
	err := p.AddLabels(context.Background(), 3, []string{"deps"})
	assert.NoError(t, err)
	assert.True(t, called)

	called = false
	err = p.AddLabels(context.Background(), 3, nil)
	assert.NoError(t, err)
	assert.False(t, called)
}
