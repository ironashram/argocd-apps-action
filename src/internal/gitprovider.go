package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

var ErrPRUpToDate = errors.New("pull request already up to date")

type PR struct {
	Number  int
	HeadRef string
}

type NewPR struct {
	Title string
	Head  string
	Base  string
	Body  string
}

type GitProvider interface {
	FindOpenPR(ctx context.Context, headBranch string) (*PR, error)
	CreatePR(ctx context.Context, p NewPR) (*PR, error)
	RefreshPR(ctx context.Context, number int) error
	AddLabels(ctx context.Context, number int, labels []string) error
}

type RefreshStyle struct {
	Method     string
	PathSuffix string
}

var (
	RefreshGitHub = RefreshStyle{Method: http.MethodPut, PathSuffix: "/update-branch"}
	RefreshGitea  = RefreshStyle{Method: http.MethodPost, PathSuffix: "/update"}
)

func ResolveRefreshStyle(hint, apiURL string) RefreshStyle {
	switch strings.ToLower(strings.TrimSpace(hint)) {
	case "github":
		return RefreshGitHub
	case "gitea", "forgejo", "codeberg":
		return RefreshGitea
	}
	if apiURL == "" || strings.Contains(apiURL, "api.github.com") {
		return RefreshGitHub
	}
	return RefreshGitea
}

type RestProvider struct {
	Client  *http.Client
	APIURL  string
	Owner   string
	Repo    string
	Token   string
	Refresh RefreshStyle
}

var _ GitProvider = (*RestProvider)(nil)

func NewRestProvider(apiURL, owner, repo, token, providerHint string) *RestProvider {
	return &RestProvider{
		Client:  http.DefaultClient,
		APIURL:  strings.TrimSuffix(apiURL, "/"),
		Owner:   owner,
		Repo:    repo,
		Token:   token,
		Refresh: ResolveRefreshStyle(providerHint, apiURL),
	}
}

func (p *RestProvider) do(ctx context.Context, method, path string, body any) (*http.Response, error) {
	var rdr io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		rdr = bytes.NewReader(buf)
	}
	req, err := http.NewRequestWithContext(ctx, method, p.APIURL+path, rdr)
	if err != nil {
		return nil, err
	}
	if p.Token != "" {
		req.Header.Set("Authorization", "token "+p.Token)
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return p.Client.Do(req)
}

func apiError(action string, resp *http.Response) error {
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	msg := strings.TrimSpace(string(b))
	if msg == "" {
		return fmt.Errorf("%s: unexpected status %d", action, resp.StatusCode)
	}
	return fmt.Errorf("%s: status %d: %s", action, resp.StatusCode, msg)
}

type prPayload struct {
	Number int `json:"number"`
	Head   struct {
		Ref string `json:"ref"`
	} `json:"head"`
}

func (p *RestProvider) FindOpenPR(ctx context.Context, headBranch string) (*PR, error) {
	path := fmt.Sprintf("/repos/%s/%s/pulls?state=open&limit=50&per_page=50&head=%s",
		p.Owner, p.Repo, url.QueryEscape(p.Owner+":"+headBranch))
	resp, err := p.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, apiError("list pull requests", resp)
	}
	var prs []prPayload
	if err := json.NewDecoder(resp.Body).Decode(&prs); err != nil {
		return nil, err
	}
	for _, pr := range prs {
		if pr.Head.Ref == headBranch {
			return &PR{Number: pr.Number, HeadRef: pr.Head.Ref}, nil
		}
	}
	return nil, nil
}

func (p *RestProvider) CreatePR(ctx context.Context, np NewPR) (*PR, error) {
	path := fmt.Sprintf("/repos/%s/%s/pulls", p.Owner, p.Repo)
	body := map[string]any{
		"title": np.Title,
		"head":  np.Head,
		"base":  np.Base,
		"body":  np.Body,
	}
	resp, err := p.do(ctx, http.MethodPost, path, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, apiError("create pull request", resp)
	}
	var pr prPayload
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return nil, err
	}
	return &PR{Number: pr.Number, HeadRef: pr.Head.Ref}, nil
}

func (p *RestProvider) RefreshPR(ctx context.Context, number int) error {
	path := fmt.Sprintf("/repos/%s/%s/pulls/%d%s", p.Owner, p.Repo, number, p.Refresh.PathSuffix)
	resp, err := p.do(ctx, p.Refresh.Method, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		return nil
	case resp.StatusCode == http.StatusUnprocessableEntity || resp.StatusCode == http.StatusConflict:
		return ErrPRUpToDate
	default:
		return apiError("refresh pull request", resp)
	}
}

func (p *RestProvider) AddLabels(ctx context.Context, number int, labels []string) error {
	if len(labels) == 0 {
		return nil
	}
	path := fmt.Sprintf("/repos/%s/%s/issues/%d/labels", p.Owner, p.Repo, number)
	resp, err := p.do(ctx, http.MethodPost, path, map[string]any{"labels": labels})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return apiError("add labels", resp)
	}
	return nil
}
