package internal_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestListOpenPRs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/repos/owner/repo/pulls", r.URL.Path)
		assert.Equal(t, "open", r.URL.Query().Get("state"))
		assert.Equal(t, "token token", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"number":7,"title":"t7","head":{"ref":"other"}},{"number":9,"title":"t9","head":{"ref":"update-x-1.0.0"}}]`))
	}))
	defer server.Close()

	p := newTestProvider(server, "forgejo")
	prs, err := p.ListOpenPRs(context.Background())
	assert.NoError(t, err)
	assert.Len(t, prs, 2)
	assert.Equal(t, 9, prs[1].Number)
	assert.Equal(t, "update-x-1.0.0", prs[1].HeadRef)
	assert.Equal(t, "t9", prs[1].Title)
}

func TestListOpenPRs_SkipsForkHeads(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"number":7,"head":{"ref":"update-x-1.0.0","repo":{"full_name":"someone/fork"}}},` +
			`{"number":9,"head":{"ref":"update-x-2.0.0","repo":{"full_name":"owner/repo"}}}]`))
	}))
	defer server.Close()

	p := newTestProvider(server, "forgejo")
	prs, err := p.ListOpenPRs(context.Background())
	assert.NoError(t, err)
	assert.Len(t, prs, 1)
	assert.Equal(t, 9, prs[0].Number)
}

func TestListOpenPRs_Paginates(t *testing.T) {
	var pages []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pages = append(pages, r.URL.Query().Get("page"))
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("page") == "1" {
			var b strings.Builder
			b.WriteString("[")
			for i := range 50 {
				if i > 0 {
					b.WriteString(",")
				}
				fmt.Fprintf(&b, `{"number":%d,"head":{"ref":"b%d"}}`, i, i)
			}
			b.WriteString("]")
			_, _ = w.Write([]byte(b.String()))
			return
		}
		_, _ = w.Write([]byte(`[{"number":99,"head":{"ref":"last"}}]`))
	}))
	defer server.Close()

	p := newTestProvider(server, "forgejo")
	prs, err := p.ListOpenPRs(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, []string{"1", "2"}, pages)
	assert.Len(t, prs, 51)
}

func TestClosePR(t *testing.T) {
	var commentBody, closedPath, closedState string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)
		switch r.Method {
		case http.MethodPost:
			assert.Equal(t, "/repos/owner/repo/issues/4/comments", r.URL.Path)
			commentBody, _ = payload["body"].(string)
		case http.MethodPatch:
			closedPath = r.URL.Path
			closedState, _ = payload["state"].(string)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	p := newTestProvider(server, "forgejo")
	err := p.ClosePR(context.Background(), 4, "superseded")
	assert.NoError(t, err)
	assert.Equal(t, "superseded", commentBody)
	assert.Equal(t, "/repos/owner/repo/pulls/4", closedPath)
	assert.Equal(t, "closed", closedState)
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

func TestDeleteBranch(t *testing.T) {
	for hint, wantPath := range map[string]string{
		"github":  "/repos/owner/repo/git/refs/heads/update-netbox-8.3.62",
		"forgejo": "/repos/owner/repo/branches/update-netbox-8.3.62",
	} {
		var gotMethod, gotPath string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotMethod, gotPath = r.Method, r.URL.Path
			w.WriteHeader(http.StatusNoContent)
		}))

		p := newTestProvider(server, hint)
		err := p.DeleteBranch(context.Background(), "update-netbox-8.3.62")
		assert.NoError(t, err)
		assert.Equal(t, http.MethodDelete, gotMethod)
		assert.Equal(t, wantPath, gotPath, hint)
		server.Close()
	}
}

func TestDeleteBranch_MissingIsNotAnError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	p := newTestProvider(server, "github")
	assert.NoError(t, p.DeleteBranch(context.Background(), "gone"))
}

func TestDeleteBranch_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"branch is protected"}`))
	}))
	defer server.Close()

	p := newTestProvider(server, "github")
	err := p.DeleteBranch(context.Background(), "protected")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "delete branch")
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
