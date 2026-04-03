package model

import (
	"encoding/csv"
	"encoding/json"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

// Device holds the identity and in-memory aggregate metrics for a single monitored device.
// All fields are protected by mu; use the provided methods for concurrent access.
type Device struct {
	ID               string
	mu               sync.RWMutex
	firstHeartbeatAt time.Time
	lastHeartbeatAt  time.Time
	heartbeatCount   int64
	uploadTimeMean   float64
	uploadTimeCount  int64
}

var (
	devicesMu sync.RWMutex
	devices   []Device

	externalStoreLogger = log.New(io.Discard, "", 0)
)

type heartbeatStoreRecord struct {
	DeviceID string    `json:"device_id"`
	SentAt   time.Time `json:"sent_at"`
}

type statsStoreRecord struct {
	DeviceID   string    `json:"device_id"`
	SentAt     time.Time `json:"sent_at"`
	UploadTime int       `json:"upload_time"`
}

// InitializeDevices reads device IDs from a CSV file and populates the global device list.
// Each row must have at least one column containing the device ID.
// Call this once at startup before serving any requests.
func InitializeDevices(filePath string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	csvReader := csv.NewReader(f)
	records, err := csvReader.ReadAll()
	if err != nil {
		return err
	}

	devicesMu.Lock()
	defer devicesMu.Unlock()

	for _, record := range records {
		if len(record) < 1 {
			continue
		}
		devices = append(devices, Device{ID: record[0]})
	}

	return nil
}

// ResetDevices clears the global device list. Intended for use in tests.
func ResetDevices() {
	devicesMu.Lock()
	defer devicesMu.Unlock()
	devices = nil
}

// GetDevice returns a pointer to the Device with the given ID, or nil if not found.
func GetDevice(deviceID string) *Device {
	devicesMu.RLock()
	defer devicesMu.RUnlock()
	for i := range devices {
		if devices[i].ID == deviceID {
			return &devices[i]
		}
	}
	return nil
}

// streamHeartbeatToExternalStore simulates writing a heartbeat record to an external component.
func streamHeartbeatToExternalStore(deviceID string, sentAt time.Time) {
	record := heartbeatStoreRecord{DeviceID: deviceID, SentAt: sentAt}
	encoded, err := json.Marshal(record)
	if err != nil {
		externalStoreLogger.Printf("failed to encode heartbeat for device %s: %v", deviceID, err)
		return
	}
	externalStoreLogger.Print(string(encoded))
}

// streamStatsToExternalStore simulates writing a stats record to an external component.
func streamStatsToExternalStore(deviceID string, sentAt time.Time, uploadTime int) {
	record := statsStoreRecord{DeviceID: deviceID, SentAt: sentAt, UploadTime: uploadTime}
	encoded, err := json.Marshal(record)
	if err != nil {
		externalStoreLogger.Printf("failed to encode stats for device %s: %v", deviceID, err)
		return
	}
	externalStoreLogger.Print(string(encoded))
}

// AddHeartbeat records a heartbeat timestamp, updating the first/last seen times and count.
// The event is also forwarded to the external store.
func (d *Device) AddHeartbeat(sentAt time.Time) {
	d.mu.Lock()
	defer d.mu.Unlock()

	streamHeartbeatToExternalStore(d.ID, sentAt)

	d.heartbeatCount++
	if d.heartbeatCount == 1 {
		d.firstHeartbeatAt = sentAt
	}
	d.lastHeartbeatAt = sentAt
}

// AddStats records an upload-time sample (nanoseconds) and maintains an incremental mean.
// The event is also forwarded to the external store.
func (d *Device) AddStats(sentAt time.Time, uploadTime int) {
	d.mu.Lock()
	defer d.mu.Unlock()

	streamStatsToExternalStore(d.ID, sentAt, uploadTime)

	d.uploadTimeCount++
	incoming := float64(uploadTime)
	d.uploadTimeMean = d.uploadTimeMean + (incoming-d.uploadTimeMean)/float64(d.uploadTimeCount)
}

// UploadTimeMean returns the current incremental mean of all recorded upload-time samples (nanoseconds).
func (d *Device) UploadTimeMean() float64 {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.uploadTimeMean
}

// StatsCount returns the total number of upload-time samples recorded for this device.
func (d *Device) StatsCount() int64 {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.uploadTimeCount
}

// HeartbeatSummary returns the first heartbeat time, last heartbeat time, and total heartbeat count.
func (d *Device) HeartbeatSummary() (time.Time, time.Time, int64) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.firstHeartbeatAt, d.lastHeartbeatAt, d.heartbeatCount
}
