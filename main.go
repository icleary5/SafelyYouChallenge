package main

import (
	"bytes"
	"encoding/json"
	"log"
	"mime"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/icleary5/SafelyYouChallenge/model"
)

type HeartbeatRequest struct {
	SentAt *time.Time `json:"sent_at"`
}

type StatsRequest struct {
	SentAt     *time.Time `json:"sent_at"`
	UploadTime *int       `json:"upload_time"`
}

func main() {

	if err := model.InitializeDevices("devices.csv"); err != nil {
		log.Fatal("Error initializing devices.csv: ", err)
	}

	r := setupRouter()

	// Start the server on port 6733
	// Note: In a production environment, you would typically want to make the host and port configurable via environment variables and/or command-line flags.
	// But because the API specification explicitly states that the server should listen on 127.0.0.1:6733, we will hardcode it here for simplicity.
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

func isJSONContentType(c *gin.Context) bool {
	contentType := c.GetHeader("Content-Type")
	if contentType == "" {
		return false
	}

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}

	return mediaType == "application/json"
}

// Handler for the /heartbeat endpoint
func postHeartbeat(c *gin.Context) {

	deviceID := c.Param("device_id")
	device := model.GetDevice(deviceID)
	if device == nil {
		c.JSON(http.StatusNotFound, gin.H{"msg": "Device not found"})
		return
	}

	raw, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Server Error"})
		return
	}

	if len(bytes.TrimSpace(raw)) == 0 {
		c.Status(http.StatusNoContent)
		return
	}

	if !isJSONContentType(c) {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Server Error"})
		return
	}

	var req HeartbeatRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Server Error"})
		return
	}

	if req.SentAt == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Server Error"})
		return
	}

	device.AddHeartbeat(*req.SentAt)
	c.Status(http.StatusNoContent)
}

func postStats(c *gin.Context) {
	deviceID := c.Param("device_id")
	device := model.GetDevice(deviceID)
	if device == nil {
		c.JSON(http.StatusNotFound, gin.H{"msg": "Device not found"})
		return
	}

	raw, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Server Error"})
		return
	}

	if len(bytes.TrimSpace(raw)) == 0 {
		c.Status(http.StatusNoContent)
		return
	}

	if !isJSONContentType(c) {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Server Error"})
		return
	}

	var req StatsRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Server Error"})
		return
	}

	if req.SentAt == nil || req.UploadTime == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Server Error"})
		return
	}

	device.AddStats(*req.SentAt, *req.UploadTime)
	c.Status(http.StatusNoContent)
}

func getStats(c *gin.Context) {
	deviceID := c.Param("device_id")
	device := model.GetDevice(deviceID)
	if device == nil {
		c.JSON(http.StatusNotFound, gin.H{"msg": "Device not found"})
		return
	}
	firstHeartbeatAt, lastHeartbeatAt, heartbeatCount := device.HeartbeatSummary()
	statsCount := device.StatsCount()

	if heartbeatCount == 0 && statsCount == 0 {
		c.Status(http.StatusNoContent)
		return
	}

	uptime := 0.0
	if heartbeatCount > 0 {
		elapsedMinutes := lastHeartbeatAt.Sub(firstHeartbeatAt).Minutes()
		if elapsedMinutes > 0 {
			uptime = float64(heartbeatCount) / elapsedMinutes * 100
		}
	}
	avgUploadDuration := time.Duration(device.UploadTimeMean()) * time.Nanosecond

	c.JSON(http.StatusOK, gin.H{
		"avg_upload_time": avgUploadDuration.String(),
		"uptime":          uptime,
	})
}
