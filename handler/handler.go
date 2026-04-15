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

// deviceKey is the Gin context key used to pass a looked-up Device from
// requireDevice middleware to route handlers.
const deviceKey = "device"

// RegisterRoutes attaches all API routes to the supplied Gin engine.
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	devices := r.Group("/api/v1/devices/:device_id", h.requireDevice)
	devices.POST("/heartbeat", requireJSON, h.postHeartbeat)
	devices.POST("/stats", requireJSON, h.postStats)
	devices.GET("/stats", h.getStats)
}

// requireDevice is middleware that looks up the device by ID and stores it in
// the Gin context under deviceKey. Aborts with 404 if the device is not found.
func (h *Handler) requireDevice(c *gin.Context) {
	device, ok := h.store.GetDevice(c.Param("device_id"))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"msg": "device not found"})
		c.Abort()
		return
	}
	c.Set(deviceKey, device)
	c.Next()
}

// requireJSON is middleware that rejects requests with a non-JSON Content-Type
// when a body is present. Empty-body requests pass through so that handlers
// can return 204 No Content.
// NOTE: 415 Unsupported Media Type would be semantically correct.
func requireJSON(c *gin.Context) {
	if c.Request.Body == nil || c.Request.ContentLength == 0 {
		c.Next()
		return
	}
	if c.ContentType() != "application/json" {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "server error"})
		c.Abort()
		return
	}
	c.Next()
}

// postHeartbeat handles POST /api/v1/devices/:device_id/heartbeat.
//
// Empty body → 204 No Content (no state change).
// JSON body must include sent_at; missing or malformed fields → 500.
// NOTE: The spec requires 500 for client input errors; semantically 400 Bad
// Request would be correct.
func (h *Handler) postHeartbeat(c *gin.Context) {
	if c.Request.Body == nil || c.Request.ContentLength == 0 {
		c.Status(http.StatusNoContent)
		return
	}

	var req heartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// NOTE: 400 Bad Request would be semantically correct.
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "server error"})
		return
	}

	c.MustGet(deviceKey).(*model.Device).AddHeartbeat(*req.SentAt)
	c.Status(http.StatusNoContent)
}

// postStats handles POST /api/v1/devices/:device_id/stats.
//
// Empty body → 204 No Content (no state change).
// JSON body must include sent_at and upload_time (nanoseconds); missing or
// malformed fields → 500.
// NOTE: The spec requires 500 for client input errors; semantically 400 Bad
// Request would be correct.
func (h *Handler) postStats(c *gin.Context) {
	if c.Request.Body == nil || c.Request.ContentLength == 0 {
		c.Status(http.StatusNoContent)
		return
	}

	var req statsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// NOTE: 400 Bad Request would be semantically correct.
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "server error"})
		return
	}

	c.MustGet(deviceKey).(*model.Device).AddStats(*req.SentAt, *req.UploadTime)
	c.Status(http.StatusNoContent)
}

// getStats handles GET /api/v1/devices/:device_id/stats.
//
// Returns 204 when the device has no recorded heartbeats or stats.
// Returns 200 with:
//   - avg_upload_time: mean upload duration formatted as a Go duration string
//   - uptime: (heartbeat count / elapsed minutes) × 100 as a percentage
func (h *Handler) getStats(c *gin.Context) {
	device := c.MustGet(deviceKey).(*model.Device)
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
