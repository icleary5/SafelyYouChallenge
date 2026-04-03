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

// HeartbeatRequest is the JSON body accepted by POST /api/v1/devices/:device_id/heartbeat.
type HeartbeatRequest struct {
	SentAt *time.Time `json:"sent_at"`
}

// StatsRequest is the JSON body accepted by POST /api/v1/devices/:device_id/stats.
// UploadTime is the video-upload duration expressed in nanoseconds.
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

// setupRouter creates and returns the Gin engine with all routes registered.
func setupRouter() *gin.Engine {
	r := gin.Default()
	r.POST("api/v1/devices/:device_id/heartbeat", postHeartbeat)
	r.POST("api/v1/devices/:device_id/stats", postStats)
	r.GET("api/v1/devices/:device_id/stats", getStats)
	return r
}

// isJSONContentType reports whether the request Content-Type header is application/json.
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

// postHeartbeat handles POST /api/v1/devices/:device_id/heartbeat.
// An empty body returns 204 with no state change.
// A JSON body must include sent_at; missing or malformed fields return 500.
// An unknown device_id returns 404.
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

// postStats handles POST /api/v1/devices/:device_id/stats.
// An empty body returns 204 with no state change.
// A JSON body must include sent_at and upload_time (nanoseconds); missing or malformed fields return 500.
// An unknown device_id returns 404.
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

// getStats handles GET /api/v1/devices/:device_id/stats.
// Returns 204 when the device has recorded no heartbeats and no stats.
// Otherwise returns 200 with avg_upload_time (Go duration string) and uptime
// (heartbeats-per-minute × 100, expressed as a percentage).
// An unknown device_id returns 404.
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
