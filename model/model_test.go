package model

import (
	"sync"
	"testing"
	"time"
)

// --- Device tests ---

func TestAddHeartbeat_UpdatesSummary(t *testing.T) {
	d := &Device{ID: "test-device"}

	first := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	second := first.Add(30 * time.Second)
	third := first.Add(60 * time.Second)

	d.AddHeartbeat(first)
	d.AddHeartbeat(second)
	d.AddHeartbeat(third)

	firstAt, lastAt, count := d.HeartbeatSummary()
	if count != 3 {
		t.Fatalf("expected count 3, got %d", count)
	}
	if !firstAt.Equal(first) {
		t.Errorf("expected first %s, got %s", first, firstAt)
	}
	if !lastAt.Equal(third) {
		t.Errorf("expected last %s, got %s", third, lastAt)
	}
}

func TestAddHeartbeat_SingleEntry_FirstEqualsLast(t *testing.T) {
	d := &Device{ID: "test-device"}
	ts := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)

	d.AddHeartbeat(ts)

	firstAt, lastAt, count := d.HeartbeatSummary()
	if count != 1 {
		t.Fatalf("expected count 1, got %d", count)
	}
	if !firstAt.Equal(ts) || !lastAt.Equal(ts) {
		t.Errorf("expected first and last to both equal %s; got first=%s last=%s", ts, firstAt, lastAt)
	}
}

func TestAddHeartbeat_Concurrent(t *testing.T) {
	const n = 100
	d := &Device{ID: "concurrent-device"}
	ts := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)

	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			d.AddHeartbeat(ts)
		}()
	}
	wg.Wait()

	_, _, count := d.HeartbeatSummary()
	if count != n {
		t.Errorf("expected count %d, got %d", n, count)
	}
}

func TestAddStats_SingleSample(t *testing.T) {
	d := &Device{ID: "test-device"}
	ts := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)

	d.AddStats(ts, 1000)

	if d.StatsCount() != 1 {
		t.Errorf("expected StatsCount 1, got %d", d.StatsCount())
	}
	if d.UploadTimeMean() != 1000.0 {
		t.Errorf("expected mean 1000.0, got %f", d.UploadTimeMean())
	}
}

func TestAddStats_IncrementalMean(t *testing.T) {
	ts := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)

	cases := []struct {
		name    string
		samples []int64
		want    float64
	}{
		{"two equal samples", []int64{200, 200}, 200.0},
		{"two samples", []int64{100, 300}, 200.0},
		{"three samples", []int64{100, 200, 300}, 200.0},
		{"single nanosecond", []int64{1}, 1.0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := &Device{ID: "test-device"}
			for _, s := range tc.samples {
				d.AddStats(ts, s)
			}
			if got := d.UploadTimeMean(); got != tc.want {
				t.Errorf("expected mean %f, got %f", tc.want, got)
			}
			if got := d.StatsCount(); got != int64(len(tc.samples)) {
				t.Errorf("expected count %d, got %d", len(tc.samples), got)
			}
		})
	}
}

// --- MemoryStore tests ---

func TestMemoryStore_GetDevice_Found(t *testing.T) {
	store := NewMemoryStore([]string{"device-a", "device-b"})

	d, ok := store.GetDevice("device-a")
	if !ok {
		t.Fatal("expected ok=true for known device")
	}
	if d == nil {
		t.Fatal("expected non-nil Device")
	}
	if d.ID != "device-a" {
		t.Errorf("expected ID device-a, got %s", d.ID)
	}
}

func TestMemoryStore_GetDevice_NotFound(t *testing.T) {
	store := NewMemoryStore([]string{"device-a"})

	d, ok := store.GetDevice("unknown")
	if ok {
		t.Error("expected ok=false for unknown device")
	}
	if d != nil {
		t.Error("expected nil Device for unknown device")
	}
}

func TestNewMemoryStoreFromCSV_LoadsDevices(t *testing.T) {
	store, err := NewMemoryStoreFromCSV("../devices.csv")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The CSV contains the device used in integration tests.
	d, ok := store.GetDevice("60-6b-44-84-dc-64")
	if !ok {
		t.Error("expected device 60-6b-44-84-dc-64 to be loaded from CSV")
	}
	if d == nil {
		t.Error("expected non-nil Device loaded from CSV")
	}
}

func TestNewMemoryStoreFromCSV_MissingFile(t *testing.T) {
	_, err := NewMemoryStoreFromCSV("nonexistent.csv")
	if err == nil {
		t.Error("expected an error for missing file, got nil")
	}
}
