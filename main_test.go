package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

func resetDevices() {
	devices = nil
	initializeDevices("devices.csv")
}

func TestPostDeviceHeartbeat(t *testing.T) {
	resetDevices()
	router := setupRouter()

	body := `{"sent_at":"2026-03-31T12:00:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/60-6b-44-84-dc-64/heartbeat", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

func TestPostDeviceStats(t *testing.T) {
	resetDevices()
	router := setupRouter()

	body := `{"sent_at":"2026-03-31T12:00:00Z","upload_time":500000000}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/60-6b-44-84-dc-64/stats", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

func TestGetDeviceStats(t *testing.T) {
	resetDevices()
	router := setupRouter()

	// Seed one stats entry so the GET returns 200 rather than 204
	statsBody := `{"sent_at":"2026-03-31T12:00:00Z","upload_time":500000000}`
	postReq := httptest.NewRequest(http.MethodPost, "/api/v1/devices/60-6b-44-84-dc-64/stats", strings.NewReader(statsBody))
	postReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(httptest.NewRecorder(), postReq)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/60-6b-44-84-dc-64/stats", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp struct {
		AvgUploadTime string  `json:"avg_upload_time"`
		Uptime        float64 `json:"uptime"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if resp.AvgUploadTime == "" {
		t.Error("expected avg_upload_time to be present in response")
	}
	// uptime is a float64; zero is a valid value so just confirm the field decoded without error
}
