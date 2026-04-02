package model

import (
	"encoding/csv"
	"os"
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
	heartbeats []Heartbeat
	stats      []Stats
}

var devices []Device

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

	for _, record := range records {
		if len(record) < 1 {
			continue
		}
		devices = append(devices, Device{ID: record[0]})
	}

	return nil
}

func ResetDevices() {
	devices = nil
}

func GetDevice(deviceID string) *Device {
	for i := range devices {
		if devices[i].ID == deviceID {
			return &devices[i]
		}
	}
	return nil
}

func (d *Device) AddHeartbeat(hb Heartbeat) {
	d.heartbeats = append(d.heartbeats, hb)
}

func (d *Device) AddStats(s Stats) {
	d.stats = append(d.stats, s)
}

func (d *Device) Heartbeats() []Heartbeat {
	return append([]Heartbeat(nil), d.heartbeats...)
}

func (d *Device) Stats() []Stats {
	return append([]Stats(nil), d.stats...)
}
