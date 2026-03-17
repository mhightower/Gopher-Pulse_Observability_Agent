// Package github provides a metric provider that collects repository statistics
// from the GitHub REST API.
package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/mhightower/gopher-pulse/internal/provider"
)

const (
	defaultBaseURL = "https://api.github.com"
	defaultTimeout = 10 * time.Second
	maxRetries     = 3
	baseBackoff    = 200 * time.Millisecond
)

// httpDoer is the interface satisfied by *http.Client, allowing transport injection in tests.
type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Provider fetches stars and open issue counts for a single GitHub repository.
type Provider struct {
	repo    string
	token   string
	baseURL string
	client  httpDoer
	logger  *slog.Logger
}

// Option configures a Provider.
type Option func(*Provider)

// WithToken sets a GitHub personal access token for authenticated requests.
func WithToken(token string) Option {
	return func(p *Provider) { p.token = token }
}

// WithBaseURL overrides the GitHub API base URL (useful in tests).
func WithBaseURL(url string) Option {
	return func(p *Provider) { p.baseURL = url }
}

// WithHTTPClient replaces the default HTTP client.
func WithHTTPClient(c httpDoer) Option {
	return func(p *Provider) { p.client = c }
}

// New constructs a GitHub Provider for the given repo (e.g. "golang/go").
func New(repo string, logger *slog.Logger, opts ...Option) *Provider {
	p := &Provider{
		repo:    repo,
		baseURL: defaultBaseURL,
		client:  &http.Client{Timeout: defaultTimeout},
		logger:  logger,
	}
	for _, o := range opts {
		o(p)
	}
	return p
}

// Name returns the stable provider identifier.
func (p *Provider) Name() string { return "github" }

// repoResponse is the subset of the GitHub repo API response we care about.
type repoResponse struct {
	StargazersCount int `json:"stargazers_count"`
	OpenIssuesCount int `json:"open_issues_count"`
}

// Collect fetches repository metrics from the GitHub API with up to maxRetries
// attempts using exponential backoff with jitter.
func (p *Provider) Collect(ctx context.Context) ([]provider.Measurement, error) {
	var (
		data *repoResponse
		err  error
	)

	for attempt := 1; attempt <= maxRetries; attempt++ {
		data, err = p.fetch(ctx)
		if err == nil {
			break
		}

		// Do not retry if the context is done.
		if ctx.Err() != nil {
			return nil, fmt.Errorf("fetch github repo %s: %w", p.repo, ctx.Err())
		}

		p.logger.Warn("github fetch failed, retrying",
			slog.String("provider", p.Name()),
			slog.String("repo", p.repo),
			slog.Int("attempt", attempt),
			slog.Int("max_retries", maxRetries),
			slog.String("error", err.Error()),
		)

		if attempt < maxRetries {
			sleep := backoff(attempt)
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("fetch github repo %s: %w", p.repo, ctx.Err())
			case <-time.After(sleep):
			}
		}
	}

	if err != nil {
		return nil, fmt.Errorf("fetch github repo %s after %d attempts: %w", p.repo, maxRetries, err)
	}

	attrs := map[string]string{
		"provider": p.Name(),
		"repo":     p.repo,
	}

	return []provider.Measurement{
		{
			Name:       "gopher_pulse_github_stars",
			Value:      float64(data.StargazersCount),
			Unit:       "1",
			Attributes: attrs,
		},
		{
			Name:       "gopher_pulse_github_open_issues",
			Value:      float64(data.OpenIssuesCount),
			Unit:       "1",
			Attributes: attrs,
		},
	}, nil
}

// fetch performs a single HTTP request to the GitHub repo endpoint.
func (p *Provider) fetch(ctx context.Context) (*repoResponse, error) {
	url := fmt.Sprintf("%s/repos/%s", p.baseURL, p.repo)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if p.token != "" {
		req.Header.Set("Authorization", "Bearer "+p.token)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	switch resp.StatusCode {
	case http.StatusOK:
		// handled below
	case http.StatusTooManyRequests, http.StatusForbidden:
		retryAfter := resp.Header.Get("Retry-After")
		return nil, fmt.Errorf("rate limited (HTTP %d), retry-after: %s", resp.StatusCode, retryAfter)
	case http.StatusNotFound:
		return nil, fmt.Errorf("repository %q not found (HTTP 404)", p.repo)
	default:
		return nil, fmt.Errorf("unexpected HTTP %d: %s", resp.StatusCode, truncate(body, 120))
	}

	var result repoResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

// backoff returns an exponential backoff duration with jitter for the given attempt (1-based).
func backoff(attempt int) time.Duration {
	exp := baseBackoff * time.Duration(1<<uint(attempt-1))
	jitter := time.Duration(rand.Int63n(int64(exp) / 2)) //nolint:gosec // weak RNG is fine for backoff jitter; no security requirement here
	return exp + jitter
}

// truncate shortens a byte slice to at most n characters for safe log output.
func truncate(b []byte, n int) string {
	s := strconv.Quote(string(b))
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}
