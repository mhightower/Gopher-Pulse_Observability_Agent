package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHandler(t *testing.T) {
	tests := []struct {
		name           string
		uptimeApprox   time.Duration
		wantStatus     string
		wantStatusCode int
	}{
		{
			name:           "returns ok with zero uptime",
			uptimeApprox:   0,
			wantStatus:     "ok",
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "returns ok with non-zero uptime",
			uptimeApprox:   2*time.Hour + 3*time.Minute,
			wantStatus:     "ok",
			wantStatusCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now().Add(-tt.uptimeApprox)
			h := Handler(start)

			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			rec := httptest.NewRecorder()

			h(rec, req)

			if rec.Code != tt.wantStatusCode {
				t.Errorf("status code = %d, want %d", rec.Code, tt.wantStatusCode)
			}

			ct := rec.Header().Get("Content-Type")
			if ct != "application/json" {
				t.Errorf("Content-Type = %q, want %q", ct, "application/json")
			}

			var body response
			if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
				t.Fatalf("failed to decode response body: %v", err)
			}

			if body.Status != tt.wantStatus {
				t.Errorf("status = %q, want %q", body.Status, tt.wantStatus)
			}

			if body.Uptime == "" {
				t.Error("uptime field must not be empty")
			}
		})
	}
}

func TestHandlerUptimeProgresses(t *testing.T) {
	// Verify that a start time in the past produces a non-zero uptime string.
	start := time.Now().Add(-5 * time.Minute)
	h := Handler(start)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	h(rec, req)

	var body response
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}

	// A 5-minute uptime truncated to seconds must not be "0s".
	if body.Uptime == "0s" {
		t.Errorf("expected non-zero uptime for 5-minute-old start, got %q", body.Uptime)
	}
}
