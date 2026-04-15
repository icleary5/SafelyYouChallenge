package model

import (
	"os"
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
	if d.UploadTimeMean() != 1000 {
		t.Errorf("expected mean 1000, got %d", d.UploadTimeMean())
	}
}

func TestAddStats_IncrementalMean(t *testing.T) {
	ts := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)

	cases := []struct {
		name    string
		samples []int64
		want    int64
	}{
		{"two equal samples", []int64{200, 200}, 200},
		{"two samples", []int64{100, 300}, 200},
		{"three samples", []int64{100, 200, 300}, 200},
		{"single nanosecond", []int64{1}, 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := &Device{ID: "test-device"}
			for _, s := range tc.samples {
				d.AddStats(ts, s)
			}
			if got := d.UploadTimeMean(); got != tc.want {
				t.Errorf("expected mean %d, got %d", tc.want, got)
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

func TestNewMemoryStoreFromCSV_MalformedCSV(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "bad-*.csv")
	if err != nil {
		t.Fatal(err)
	}
	// A field starting with a quote that is never closed is invalid per RFC 4180.
	if _, err := f.WriteString(`"unclosed`); err != nil {
		t.Fatal(err)
	}
	f.Close()

	_, err = NewMemoryStoreFromCSV(f.Name())
	if err == nil {
		t.Error("expected an error for malformed CSV, got nil")
	}
}

func TestNewMemoryStore_Empty(t *testing.T) {
	store := NewMemoryStore([]string{})

	d, ok := store.GetDevice("any")
	if ok {
		t.Error("expected ok=false for empty store")
	}
	if d != nil {
		t.Error("expected nil Device for empty store")
	}
}

func TestMemoryStore_GetDevice_Concurrent(t *testing.T) {
	const n = 100
	store := NewMemoryStore([]string{"device-a"})

	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			d, ok := store.GetDevice("device-a")
			if !ok || d == nil {
				t.Errorf("expected ok=true and non-nil Device in concurrent read")
			}
		}()
	}
	wg.Wait()
}

func TestAddStats_Concurrent(t *testing.T) {
	const n = 100
	d := &Device{ID: "concurrent-device"}
	ts := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)

	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			d.AddStats(ts, 1000)
		}()
	}
	wg.Wait()

	if got := d.StatsCount(); got != n {
		t.Errorf("expected StatsCount %d, got %d", n, got)
	}
	if got := d.UploadTimeMean(); got != 1000 {
		t.Errorf("expected mean 1000, got %d", got)
	}
}
