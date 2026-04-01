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
	UploadTime int       `json:"upload_time"` // upload duration in milliseconds
}

type Device struct {
	ID         string
	heartbeats []Heartbeat
	stats      []Stats
}

var devices []Device // In-memory storage for devices, a poorman’s database for this example

func main() {

	initializeDevices("devices.csv")

	r := setupRouter()

	// Start the server on port 6733
	if err := r.Run(":6733"); err != nil {
		log.Fatal("Error starting server: ", err)
	}
}

func setupRouter() *gin.Engine {
	r := gin.Default()
	r.POST("/devices/:device_id/heartbeat", postHeartbeat)
	r.POST("/devices/:device_id/stats", postStats)
	r.GET("/devices/:device_id/stats", getStats)
	return r
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
	c.Status(http.StatusNoContent)
}

func postStats(c *gin.Context) {
	deviceID := c.Param("device_id")
	device := retrieveDevice(deviceID)
	if device == nil {
		c.JSON(http.StatusNotFound, gin.H{"msg": "Device not found"})
		return
	}

	var s Stats
	if err := c.BindJSON(&s); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid request body"})
		return
	}

	device.stats = append(device.stats, s)
	c.Status(http.StatusNoContent)
}

func getStats(c *gin.Context) {
	deviceID := c.Param("device_id")
	device := retrieveDevice(deviceID)
	if device == nil {
		c.JSON(http.StatusNotFound, gin.H{"msg": "Device not found"})
		return
	}

	if len(device.stats) == 0 {
		c.Status(http.StatusNoContent)
		return
	}

	var totalUpload int
	for _, s := range device.stats {
		totalUpload += s.UploadTime
	}
	avgNs := totalUpload / len(device.stats)
	avgDuration := time.Duration(avgNs)

	var uptimePct float64
	if len(device.heartbeats) > 0 {
		first := device.heartbeats[0].SentAt
		last := device.heartbeats[len(device.heartbeats)-1].SentAt
		expected := last.Sub(first).Hours()
		if expected > 0 {
			uptimePct = (float64(len(device.heartbeats)) / expected) * 100
		} else {
			uptimePct = 100
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"avg_upload_time": avgDuration.String(),
		"uptime":          uptimePct,
	})
}
