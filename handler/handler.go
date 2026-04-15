package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/icleary5/SafelyYouChallenge/model"
)

// heartbeatRequest is the JSON body accepted by POST /api/v1/devices/:device_id/heartbeat.
type heartbeatRequest struct {
	SentAt *time.Time `json:"sent_at" binding:"required"`
}

// statsRequest is the JSON body accepted by POST /api/v1/devices/:device_id/stats.
// UploadTime is the video-upload duration expressed in nanoseconds.
type statsRequest struct {
	SentAt     *time.Time `json:"sent_at" binding:"required"`
	UploadTime *int64     `json:"upload_time" binding:"required"`
}

// Handler holds the dependencies shared across all HTTP handlers.
type Handler struct {
	store model.Store
}

// New creates a Handler backed by the provided Store.
func New(store model.Store) *Handler {
	return &Handler{store: store}
}

// RegisterRoutes attaches all API routes to the supplied Gin engine.
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.POST("api/v1/devices/:device_id/heartbeat", h.postHeartbeat)
	r.POST("api/v1/devices/:device_id/stats", h.postStats)
	r.GET("api/v1/devices/:device_id/stats", h.getStats)
}

// postHeartbeat handles POST /api/v1/devices/:device_id/heartbeat.
//
// Empty body → 204 No Content (no state change).
// JSON body must include sent_at; missing or malformed fields → 500.
// NOTE: The spec requires 500 for client input errors; semantically 400 Bad
// Request and 415 Unsupported Media Type would be correct.
// Unknown device_id → 404.
func (h *Handler) postHeartbeat(c *gin.Context) {
	device, ok := h.store.GetDevice(c.Param("device_id"))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"msg": "device not found"})
		return
	}

	if c.Request.ContentLength == 0 {
		c.Status(http.StatusNoContent)
		return
	}

	if c.ContentType() != "application/json" {
		// NOTE: 415 Unsupported Media Type would be semantically correct.
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "server error"})
		return
	}

	var req heartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// NOTE: 400 Bad Request would be semantically correct.
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "server error"})
		return
	}

	device.AddHeartbeat(*req.SentAt)
	c.Status(http.StatusNoContent)
}

// postStats handles POST /api/v1/devices/:device_id/stats.
//
// Empty body → 204 No Content (no state change).
// JSON body must include sent_at and upload_time (nanoseconds); missing or
// malformed fields → 500.
// NOTE: The spec requires 500 for client input errors; semantically 400 Bad
// Request and 415 Unsupported Media Type would be correct.
// Unknown device_id → 404.
func (h *Handler) postStats(c *gin.Context) {
	device, ok := h.store.GetDevice(c.Param("device_id"))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"msg": "device not found"})
		return
	}

	if c.Request.ContentLength == 0 {
		c.Status(http.StatusNoContent)
		return
	}

	if c.ContentType() != "application/json" {
		// NOTE: 415 Unsupported Media Type would be semantically correct.
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "server error"})
		return
	}

	var req statsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// NOTE: 400 Bad Request would be semantically correct.
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "server error"})
		return
	}

	device.AddStats(*req.SentAt, *req.UploadTime)
	c.Status(http.StatusNoContent)
}

// getStats handles GET /api/v1/devices/:device_id/stats.
//
// Returns 204 when the device has no recorded heartbeats or stats.
// Returns 200 with:
//   - avg_upload_time: mean upload duration formatted as a Go duration string
//   - uptime: (heartbeat count / elapsed minutes) × 100 as a percentage
//
// Unknown device_id → 404.
func (h *Handler) getStats(c *gin.Context) {
	device, ok := h.store.GetDevice(c.Param("device_id"))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"msg": "device not found"})
		return
	}

	firstAt, lastAt, heartbeatCount := device.HeartbeatSummary()
	statsCount := device.StatsCount()

	if heartbeatCount == 0 && statsCount == 0 {
		c.Status(http.StatusNoContent)
		return
	}

	uptime := 0.0
	if heartbeatCount > 0 {
		if elapsed := lastAt.Sub(firstAt).Minutes(); elapsed > 0 {
			uptime = float64(heartbeatCount) / elapsed * 100
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"avg_upload_time": (time.Duration(device.UploadTimeMean()) * time.Nanosecond).String(),
		"uptime":          uptime,
	})
}
