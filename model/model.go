package model

import (
	"math"
	"sync"
	"time"
)

// Store is the interface that wraps device lookup. A caller can retrieve a
// device by its ID without knowing anything about the underlying storage.
type Store interface {
	GetDevice(id string) (*Device, bool)
}

// Device holds the identity and in-memory aggregate metrics for a single
// monitored device. All mutable fields are protected by mu; use the provided
// methods for concurrent access.
type Device struct {
	ID               string
	mu               sync.RWMutex
	firstHeartbeatAt time.Time
	lastHeartbeatAt  time.Time
	heartbeatCount   int64
	uploadTimeMean   float64
	uploadTimeCount  int64
}

// AddHeartbeat records a heartbeat timestamp, updating the first/last seen
// times and total count.
func (d *Device) AddHeartbeat(sentAt time.Time) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.heartbeatCount++
	if d.heartbeatCount == 1 {
		d.firstHeartbeatAt = sentAt
	}
	d.lastHeartbeatAt = sentAt
}

// AddStats records an upload-time sample (nanoseconds) and maintains an
// incremental mean, avoiding the need to store every individual sample.
func (d *Device) AddStats(sentAt time.Time, uploadTime int64) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.uploadTimeCount++
	d.uploadTimeMean += (float64(uploadTime) - d.uploadTimeMean) / float64(d.uploadTimeCount)
}

// UploadTimeMean returns the rounded mean of all recorded upload-time samples
// (nanoseconds). Rounding is applied here so callers receive a clean int64
// rather than silently truncating a float64 at the call site.
func (d *Device) UploadTimeMean() int64 {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return int64(math.Round(d.uploadTimeMean))
}

// StatsCount returns the total number of upload-time samples recorded for
// this device.
func (d *Device) StatsCount() int64 {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.uploadTimeCount
}

// HeartbeatSummary returns the first heartbeat time, last heartbeat time, and
// total heartbeat count for this device.
func (d *Device) HeartbeatSummary() (time.Time, time.Time, int64) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.firstHeartbeatAt, d.lastHeartbeatAt, d.heartbeatCount
}
