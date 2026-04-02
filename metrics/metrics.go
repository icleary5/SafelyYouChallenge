package metrics

import (
	"time"

	"github.com/icleary5/SafelyYouChallenge/model"
)

// Uptime computes heartbeat count per elapsed minute scaled by 100.
func Uptime(heartbeats []model.Heartbeat) float64 {
	if len(heartbeats) == 0 {
		return 0.0
	}

	first := heartbeats[0].SentAt
	last := heartbeats[len(heartbeats)-1].SentAt
	elapsed := last.Sub(first).Minutes()
	if elapsed <= 0 {
		return 0.0
	}

	return float64(len(heartbeats)) / elapsed * 100
}

// AverageUploadDuration computes average upload time and returns it as a duration.
func AverageUploadDuration(stats []model.Stats) time.Duration {
	if len(stats) == 0 {
		return 0
	}

	var totalUploadTime int
	for _, stat := range stats {
		totalUploadTime += stat.UploadTime
	}

	avgUploadTime := float64(totalUploadTime) / float64(len(stats))
	if avgUploadTime == 0 {
		return 0
	}

	return time.Duration(avgUploadTime) * time.Nanosecond
}
