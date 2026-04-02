package model

import (
	"testing"
	"time"
)

func TestAddHeartbeatUpdatesSummary(t *testing.T) {
	device := &Device{ID: "test-device"}

	first := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	second := first.Add(30 * time.Second)
	third := first.Add(60 * time.Second)

	device.AddHeartbeat(first)
	device.AddHeartbeat(second)
	device.AddHeartbeat(third)

	firstAt, lastAt, count := device.HeartbeatSummary()
	if count != 3 {
		t.Fatalf("expected heartbeat count 3, got %d", count)
	}
	if !firstAt.Equal(first) {
		t.Fatalf("expected first heartbeat %s, got %s", first, firstAt)
	}
	if !lastAt.Equal(third) {
		t.Fatalf("expected last heartbeat %s, got %s", third, lastAt)
	}
}
