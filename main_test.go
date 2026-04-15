package main

import (
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/icleary5/SafelyYouChallenge/model"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

// newTestStore returns a store pre-loaded with the single device used across
// all integration tests. No filesystem access is required.
func newTestStore() *model.MemoryStore {
	return model.NewMemoryStore([]string{"60-6b-44-84-dc-64"})
}

// assertStatus fails the test if the recorded response code does not match want.
func assertStatus(t *testing.T, w *httptest.ResponseRecorder, want int) {
	t.Helper()
	if w.Code != want {
		t.Errorf("expected status %d, got %d", want, w.Code)
	}
}

func TestPostDeviceHeartbeat(t *testing.T) {
	router := setupRouter(newTestStore())

	body := `{"sent_at":"2026-03-31T12:00:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/60-6b-44-84-dc-64/heartbeat", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusNoContent)
}

func TestPostDeviceStats(t *testing.T) {
	router := setupRouter(newTestStore())

	body := `{"sent_at":"2026-03-31T12:00:00Z","upload_time":500000000}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/60-6b-44-84-dc-64/stats", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusNoContent)
}

func TestPostHeartbeatNoBodyReturnsNoContent(t *testing.T) {
	router := setupRouter(newTestStore())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/60-6b-44-84-dc-64/heartbeat", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusNoContent)
}

func TestPostStatsNoBodyReturnsNoContent(t *testing.T) {
	router := setupRouter(newTestStore())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/60-6b-44-84-dc-64/stats", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusNoContent)
}

func TestGetDeviceStats(t *testing.T) {
	router := setupRouter(newTestStore())

	heartbeats := []string{
		`{"sent_at":"2026-03-31T12:00:00Z"}`,
		`{"sent_at":"2026-03-31T12:00:30Z"}`,
		`{"sent_at":"2026-03-31T12:01:00Z"}`,
	}
	for _, heartbeatBody := range heartbeats {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/60-6b-44-84-dc-64/heartbeat", strings.NewReader(heartbeatBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusNoContent {
			t.Fatalf("expected heartbeat POST 204, got %d", w.Code)
		}
	}

	statsBody := `{"sent_at":"2026-03-31T12:00:00Z","upload_time":500000000}`
	postReq := httptest.NewRequest(http.MethodPost, "/api/v1/devices/60-6b-44-84-dc-64/stats", strings.NewReader(statsBody))
	postReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(httptest.NewRecorder(), postReq)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/60-6b-44-84-dc-64/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusOK)

	var resp struct {
		AvgUploadTime string  `json:"avg_upload_time"`
		Uptime        float64 `json:"uptime"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if resp.AvgUploadTime != "500ms" {
		t.Errorf("expected avg_upload_time 500ms, got %q", resp.AvgUploadTime)
	}
	if math.Abs(resp.Uptime-300.0) > 1e-9 {
		t.Errorf("expected uptime 300.0, got %f", resp.Uptime)
	}
}

func TestGetDeviceStatsNoDataReturnsNoContent(t *testing.T) {
	router := setupRouter(newTestStore())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/60-6b-44-84-dc-64/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusNoContent)
}

// TestPostHeartbeatErrors exercises all invalid-input paths for the heartbeat
// endpoint in a table-driven style.
func TestPostHeartbeatErrors(t *testing.T) {
	cases := []struct {
		name        string
		body        string
		contentType string
		wantStatus  int
	}{
		{
			name:        "missing sent_at",
			body:        `{}`,
			contentType: "application/json",
			wantStatus:  http.StatusInternalServerError,
		},
		{
			name:        "malformed JSON",
			body:        `{"sent_at":`,
			contentType: "application/json",
			wantStatus:  http.StatusInternalServerError,
		},
		{
			name:        "wrong content type",
			body:        `{"sent_at":"2026-03-31T12:00:00Z"}`,
			contentType: "text/plain",
			wantStatus:  http.StatusInternalServerError,
		},
		{
			name:        "unknown device",
			body:        `{"sent_at":"2026-03-31T12:00:00Z"}`,
			contentType: "application/json",
			wantStatus:  http.StatusNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			router := setupRouter(newTestStore())

			path := "/api/v1/devices/60-6b-44-84-dc-64/heartbeat"
			if tc.wantStatus == http.StatusNotFound {
				path = "/api/v1/devices/unknown/heartbeat"
			}

			req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", tc.contentType)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assertStatus(t, w, tc.wantStatus)

			var resp map[string]string
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("failed to decode response body: %v", err)
			}
			if resp["msg"] == "" {
				t.Error("expected msg field in response body")
			}
		})
	}
}

// TestPostStatsErrors exercises all invalid-input paths for the stats endpoint
// in a table-driven style, symmetric with TestPostHeartbeatErrors.
func TestPostStatsErrors(t *testing.T) {
	cases := []struct {
		name        string
		body        string
		contentType string
		wantStatus  int
	}{
		{
			name:        "missing fields",
			body:        `{}`,
			contentType: "application/json",
			wantStatus:  http.StatusInternalServerError,
		},
		{
			name:        "malformed JSON",
			body:        `{"sent_at":`,
			contentType: "application/json",
			wantStatus:  http.StatusInternalServerError,
		},
		{
			name:        "wrong content type",
			body:        `{"sent_at":"2026-03-31T12:00:00Z","upload_time":500000000}`,
			contentType: "text/plain",
			wantStatus:  http.StatusInternalServerError,
		},
		{
			name:        "unknown device",
			body:        `{"sent_at":"2026-03-31T12:00:00Z","upload_time":500000000}`,
			contentType: "application/json",
			wantStatus:  http.StatusNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			router := setupRouter(newTestStore())

			path := "/api/v1/devices/60-6b-44-84-dc-64/stats"
			if tc.wantStatus == http.StatusNotFound {
				path = "/api/v1/devices/unknown/stats"
			}

			req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", tc.contentType)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assertStatus(t, w, tc.wantStatus)

			var resp map[string]string
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("failed to decode response body: %v", err)
			}
			if resp["msg"] == "" {
				t.Error("expected msg field in response body")
			}
		})
	}
}
