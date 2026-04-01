package main

import (
	"encoding/csv"
	"fmt"
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
	if err := r.Run("localhost:6733"); err != nil {
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

	var uptime float64
	// Calculate uptime from heartbeats
	if len(device.heartbeats) != 0 {
		first := device.heartbeats[0].SentAt
		last := device.heartbeats[len(device.heartbeats)-1].SentAt
		elapsed := last.Sub(first).Minutes()
		heartbeatCount := len(device.heartbeats)
		uptime = float64(heartbeatCount) / elapsed * 100
	} else {
		uptime = 0.0
	}

	// Calculate average upload time
	var avgUploadTime float64
	if len(device.stats) != 0 {
		var totalUploadTime int
		for _, stat := range device.stats {
			totalUploadTime += stat.UploadTime
		}
		avgUploadTime = float64(totalUploadTime) / float64(len(device.stats))
	} else {
		avgUploadTime = 0.0
	}

	// It's unclear what unit of measure the upload times are in, so I will simply 
	// convert the floating point average to a string
	var avgUploadTimeStr string
	avgUploadTimeStr = fmt.Sprintf("%.2f", avgUploadTime)

	c.JSON(http.StatusOK, gin.H{
		"avg_upload_time": avgUploadTimeStr,
		"uptime":         uptime,
	})
}
