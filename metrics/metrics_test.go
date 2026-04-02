package metrics

import (
	"testing"
	"time"

	"github.com/icleary5/SafelyYouChallenge/model"
)

func TestUptimeWithNoHeartbeatsReturnsZero(t *testing.T) {
	uptime := Uptime(nil)
	if uptime != 0 {
		t.Fatalf("expected 0 uptime, got %f", uptime)
	}
}

func TestUptimeWithElapsedMinutes(t *testing.T) {
	start := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	heartbeats := []model.Heartbeat{
		{SentAt: start},
		{SentAt: start.Add(30 * time.Second)},
		{SentAt: start.Add(60 * time.Second)},
	}

	uptime := Uptime(heartbeats)
	if uptime != 300 {
		t.Fatalf("expected 300 uptime, got %f", uptime)
	}
}

func TestUptimeWithNoElapsedTimeReturnsZero(t *testing.T) {
	now := time.Now().UTC()
	heartbeats := []model.Heartbeat{{SentAt: now}, {SentAt: now}}

	uptime := Uptime(heartbeats)
	if uptime != 0 {
		t.Fatalf("expected 0 uptime, got %f", uptime)
	}
}

func TestAverageUploadDurationWithNoStatsReturnsZero(t *testing.T) {
	d := AverageUploadDuration(nil)
	if d != 0 {
		t.Fatalf("expected 0 duration, got %s", d)
	}
}

func TestAverageUploadDuration(t *testing.T) {
	stats := []model.Stats{
		{UploadTime: 500000000},
		{UploadTime: 700000000},
	}

	d := AverageUploadDuration(stats)
	if d != 600*time.Millisecond {
		t.Fatalf("expected 600ms, got %s", d)
	}
}
