package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

// roundTripFunc allows a plain function to satisfy http.RoundTripper.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func newMockClient(fn roundTripFunc) httpDoer {
	return &http.Client{Transport: fn}
}

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func TestCollect_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/repos/golang/go") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, mustJSON(repoResponse{StargazersCount: 25000, OpenIssuesCount: 300}))
	}))
	defer srv.Close()

	p := New("golang/go", newTestLogger(), WithBaseURL(srv.URL))
	measurements, err := p.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	if len(measurements) != 2 {
		t.Fatalf("got %d measurements, want 2", len(measurements))
	}

	byName := make(map[string]float64)
	for _, m := range measurements {
		byName[m.Name] = m.Value
	}

	if got := byName["gopher_pulse_github_stars"]; got != 25000 {
		t.Errorf("stars = %v, want 25000", got)
	}
	if got := byName["gopher_pulse_github_open_issues"]; got != 300 {
		t.Errorf("open_issues = %v, want 300", got)
	}

	for _, m := range measurements {
		if m.Attributes["provider"] != "github" {
			t.Errorf("provider attribute = %q, want %q", m.Attributes["provider"], "github")
		}
		if m.Attributes["repo"] != "golang/go" {
			t.Errorf("repo attribute = %q, want %q", m.Attributes["repo"], "golang/go")
		}
	}
}

func TestCollect_NotFound(t *testing.T) {
	client := newMockClient(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       io.NopCloser(strings.NewReader(`{"message":"Not Found"}`)),
			Header:     make(http.Header),
		}, nil
	})

	p := New("owner/nonexistent", newTestLogger(), WithHTTPClient(client))
	_, err := p.Collect(context.Background())
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error %q should mention 'not found'", err.Error())
	}
}

func TestCollect_RateLimit(t *testing.T) {
	client := newMockClient(func(r *http.Request) (*http.Response, error) {
		h := make(http.Header)
		h.Set("Retry-After", "3600")
		return &http.Response{
			StatusCode: http.StatusForbidden,
			Body:       io.NopCloser(strings.NewReader(`{"message":"rate limited"}`)),
			Header:     h,
		}, nil
	})

	p := New("golang/go", newTestLogger(), WithHTTPClient(client))
	_, err := p.Collect(context.Background())
	if err == nil {
		t.Fatal("expected error for rate limit, got nil")
	}
	if !strings.Contains(err.Error(), "rate limited") {
		t.Errorf("error %q should mention 'rate limited'", err.Error())
	}
}

func TestCollect_TransportError_Retries(t *testing.T) {
	callCount := 0
	client := newMockClient(func(r *http.Request) (*http.Response, error) {
		callCount++
		return nil, fmt.Errorf("connection refused")
	})

	p := New("golang/go", newTestLogger(), WithHTTPClient(client))

	// Shorten backoff for the test by pointing to a fast-failing server.
	_, err := p.Collect(context.Background())
	if err == nil {
		t.Fatal("expected error after retries, got nil")
	}
	if callCount != maxRetries {
		t.Errorf("expected %d attempts, got %d", maxRetries, callCount)
	}
	if !strings.Contains(err.Error(), "after 3 attempts") {
		t.Errorf("error %q should mention attempt count", err.Error())
	}
}

func TestCollect_ContextCancelledMidRetry(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	callCount := 0
	client := newMockClient(func(r *http.Request) (*http.Response, error) {
		callCount++
		if callCount == 1 {
			cancel() // cancel after the first failure
		}
		return nil, fmt.Errorf("transport error")
	})

	p := New("golang/go", newTestLogger(), WithHTTPClient(client))
	_, err := p.Collect(ctx)
	if err == nil {
		t.Fatal("expected error after context cancellation, got nil")
	}
	// Should not have exhausted all retries.
	if callCount >= maxRetries {
		t.Errorf("expected early exit on cancel, but made %d calls", callCount)
	}
}

func TestCollect_InvalidJSON(t *testing.T) {
	client := newMockClient(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`not json`)),
			Header:     make(http.Header),
		}, nil
	})

	p := New("golang/go", newTestLogger(), WithHTTPClient(client))
	_, err := p.Collect(context.Background())
	if err == nil {
		t.Fatal("expected decode error, got nil")
	}
	if !strings.Contains(err.Error(), "decode response") {
		t.Errorf("error %q should mention 'decode response'", err.Error())
	}
}

func TestCollect_AuthHeader(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, mustJSON(repoResponse{}))
	}))
	defer srv.Close()

	p := New("golang/go", newTestLogger(), WithBaseURL(srv.URL), WithToken("my-token"))
	_, err := p.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if gotAuth != "Bearer my-token" {
		t.Errorf("Authorization header = %q, want %q", gotAuth, "Bearer my-token")
	}
}

func TestBackoff(t *testing.T) {
	tests := []struct {
		attempt int
		minWant time.Duration
		maxWant time.Duration
	}{
		{1, baseBackoff, baseBackoff * 3},
		{2, baseBackoff * 2, baseBackoff * 6},
		{3, baseBackoff * 4, baseBackoff * 12},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("attempt_%d", tt.attempt), func(t *testing.T) {
			d := backoff(tt.attempt)
			if d < tt.minWant || d > tt.maxWant {
				t.Errorf("backoff(%d) = %v, want between %v and %v", tt.attempt, d, tt.minWant, tt.maxWant)
			}
		})
	}
}

func TestName(t *testing.T) {
	p := New("golang/go", newTestLogger())
	if got := p.Name(); got != "github" {
		t.Errorf("Name() = %q, want %q", got, "github")
	}
}
