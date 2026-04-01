package main

import (
	"encoding/csv"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

type Heartbeat struct {
	SentAt time.Time `json:"sent_at"`
}

type Stats struct {
	SentAt     time.Time `json:"sent_at"`
	UploadTime int `json:"upload_time"` // upload duration in milliseconds
}

type Device struct {
	ID         string
	heartbeats []Heartbeat
	stats      []Stats
}

var devices []Device // In-memory storage for devices, a poorman’s database for this example

func main() {

	initializeDevices("devices.csv")

	r := gin.Default()

	r.POST("/devices/:device_id/heartbeat", postHeartbeat)

	// Start the server on port 6733
	if err := r.Run(":6733"); err != nil {
		log.Fatal("Error starting server: ", err)
	}
}

func initializeDevices(filePath string) {

	f, err := os.Open(filePath)
	if err != nil {
		log.Fatal("Error opening devices.csv: ", err)
	}
	defer f.Close()

	csvReader := csv.NewReader(f)
	records, err := csvReader.ReadAll()
	if err != nil {
		log.Fatal("Error reading devices.csv: ", err)
	}

	for _, record := range records {
		if len(record) < 1 {
			continue // Skip invalid records
		}
		devices = append(devices, Device{ID: record[0]})
	}
}

func retrieveDevice(deviceID string) *Device {
	var device *Device
	for i := range devices {
		if devices[i].ID == deviceID {
			device = &devices[i]
			break
		}
	}
	return device
}

// Handler for the /heartbeat endpoint
func postHeartbeat(c *gin.Context) {

	deviceID := c.Param("device_id")
	device := retrieveDevice(deviceID)
	if device == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Device not found"})
		return
	}

	var heartbeat Heartbeat
	if err := c.BindJSON(&heartbeat); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	heartbeats := append(device.heartbeats, heartbeat)
	device.heartbeats = heartbeats
}
