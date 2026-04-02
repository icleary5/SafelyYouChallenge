package model

import (
	"encoding/csv"
	"os"
	"sync"
	"time"
)

type Heartbeat struct {
	SentAt time.Time `json:"sent_at"`
}

type Stats struct {
	SentAt     time.Time `json:"sent_at"`
	UploadTime int       `json:"upload_time"` // upload duration in nanoseconds
}

type Device struct {
	ID         string
	mu         sync.RWMutex
	heartbeats []Heartbeat
	stats      []Stats
	uploadTimeMean  float64
	uploadTimeCount int64
}

var (
	devicesMu sync.RWMutex
	devices   []Device
)

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

func ResetDevices() {
	devicesMu.Lock()
	defer devicesMu.Unlock()
	devices = nil
}

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

func (d *Device) AddHeartbeat(hb Heartbeat) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.heartbeats = append(d.heartbeats, hb)
}

func (d *Device) AddStats(s Stats) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.stats = append(d.stats, s)

	d.uploadTimeCount++
	incoming := float64(s.UploadTime)
	d.uploadTimeMean = d.uploadTimeMean + (incoming-d.uploadTimeMean)/float64(d.uploadTimeCount)
}

func (d *Device) Heartbeats() []Heartbeat {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return append([]Heartbeat(nil), d.heartbeats...)
}

func (d *Device) Stats() []Stats {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return append([]Stats(nil), d.stats...)
}

func (d *Device) UploadTimeMean() float64 {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.uploadTimeMean
}

func (d *Device) StatsCount() int64 {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.uploadTimeCount
}

// HeartbeatsAndStats returns a consistent snapshot of both slices under a single lock,
// preventing a torn read between the two calls in getStats.
func (d *Device) HeartbeatsAndStats() ([]Heartbeat, []Stats) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return append([]Heartbeat(nil), d.heartbeats...), append([]Stats(nil), d.stats...)
}
