package model

import (
	"testing"
	"time"
)

func TestAddHeartbeatUpdatesSummaryAndPreservesSlice(t *testing.T) {
	device := &Device{ID: "test-device"}

	first := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	second := first.Add(30 * time.Second)
	third := first.Add(60 * time.Second)

	device.AddHeartbeat(Heartbeat{SentAt: first})
	device.AddHeartbeat(Heartbeat{SentAt: second})
	device.AddHeartbeat(Heartbeat{SentAt: third})

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

	heartbeats := device.Heartbeats()
	if len(heartbeats) != 3 {
		t.Fatalf("expected 3 heartbeats in slice, got %d", len(heartbeats))
	}
	if !heartbeats[0].SentAt.Equal(first) || !heartbeats[1].SentAt.Equal(second) || !heartbeats[2].SentAt.Equal(third) {
		t.Fatalf("expected heartbeats to be preserved in insertion order")
	}
}
