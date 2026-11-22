package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestLoggerRecordsHTTPStatus(t *testing.T) {
	tests := []struct {
		name       string
		write      func(http.ResponseWriter)
		wantStatus int
	}{
		{
			name:       "implicit status without a body",
			write:      func(http.ResponseWriter) {},
			wantStatus: http.StatusOK,
		},
		{
			name: "writing the body commits status 200",
			write: func(w http.ResponseWriter) {
				w.Write([]byte("body"))
				w.WriteHeader(http.StatusBadRequest)
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "informational response followed by a final status",
			write: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusEarlyHints)
				w.WriteHeader(http.StatusNoContent)
			},
			wantStatus: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			routes := http.NewServeMux()
			routes.HandleFunc("GET /account/{id}", func(w http.ResponseWriter, _ *http.Request) {
				tt.write(w)
			})
			var output bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&output, nil))
			request := httptest.NewRequest(http.MethodGet, "/account/1", nil)
			RequestLogger(logger, routes).ServeHTTP(httptest.NewRecorder(), request)

			var entry struct {
				Method     string   `json:"method"`
				Route      string   `json:"route"`
				Status     int      `json:"status"`
				DurationMS *float64 `json:"duration_ms"`
			}
			if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
				t.Fatalf("decode request log: %v", err)
			}
			if entry.Method != http.MethodGet || entry.Route != "GET /account/{id}" {
				t.Errorf("logged method %q and route %q", entry.Method, entry.Route)
			}
			if entry.Status != tt.wantStatus {
				t.Errorf("logged status = %d, want %d", entry.Status, tt.wantStatus)
			}
			if entry.DurationMS == nil || *entry.DurationMS < 0 {
				t.Errorf("want a non-negative duration_ms, got %s", output.Bytes())
			}
		})
	}
}
