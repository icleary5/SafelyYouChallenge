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
	UploadTime int       `json:"upload_time"` // upload duration in nanoseconds
}

type HeartbeatRequest struct {
	SentAt *time.Time `json:"sent_at"`
}

type StatsRequest struct {
	SentAt     *time.Time `json:"sent_at"`
	UploadTime *int       `json:"upload_time"`
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
	if err := r.Run("127.0.0.1:6733"); err != nil {
		log.Fatal("Error starting server: ", err)
	}
}

func setupRouter() *gin.Engine {
	r := gin.Default()
	r.POST("api/v1/devices/:device_id/heartbeat", postHeartbeat)
	r.POST("api/v1/devices/:device_id/stats", postStats)
	r.GET("api/v1/devices/:device_id/stats", getStats)
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
		c.JSON(http.StatusNotFound, gin.H{"msg": "Device not found"})
		return
	}

	var req HeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Server Error"})
		return
	}

	if req.SentAt == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Server Error"})
		return
	}

	heartbeat := Heartbeat{SentAt: *req.SentAt}

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

	var req StatsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Server Error"})
		return
	}

	if req.SentAt == nil || req.UploadTime == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Server Error"})
		return
	}

	s := Stats{SentAt: *req.SentAt, UploadTime: *req.UploadTime}

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

	if len(device.heartbeats) == 0 && len(device.stats) == 0 {
		c.Status(http.StatusNoContent)
		return
	}

	var uptime float64
	// Calculate uptime from heartbeats
	if len(device.heartbeats) != 0 {
		first := device.heartbeats[0].SentAt
		last := device.heartbeats[len(device.heartbeats)-1].SentAt
		elapsed := last.Sub(first).Minutes()
		heartbeatCount := len(device.heartbeats)
		if elapsed > 0 {
		uptime = float64(heartbeatCount) / elapsed * 100
	} else {
		uptime = 0.0
	}
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

	var avgUploadDuration time.Duration
	if avgUploadTime != 0 {
		avgUploadDuration = time.Duration(avgUploadTime) * time.Nanosecond
	} else {
		avgUploadDuration = 0
	}

	c.JSON(http.StatusOK, gin.H{
		"avg_upload_time": avgUploadDuration.String(),
		"uptime":         uptime,
	})
}
