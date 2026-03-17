// Package health provides an HTTP handler that reports agent liveness and uptime.
package health

import (
	"encoding/json"
	"net/http"
	"time"
)

// response is the JSON body returned by the health endpoint.
type response struct {
	Status string `json:"status"`
	Uptime string `json:"uptime"`
}

// Handler returns an http.HandlerFunc that always responds 200 OK with a JSON
// body containing the agent status and uptime since start.
func Handler(start time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uptime := time.Since(start).Truncate(time.Second)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response{ //nolint:errcheck
			Status: "ok",
			Uptime: uptime.String(),
		})
	}
}
