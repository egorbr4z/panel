package handler

import (
	"net/http"
	"runtime"
	"time"

	"github.com/egorbr4z/panel/internal/api/dto"
	"github.com/gin-gonic/gin"
)

var startedAt = time.Now()

// Version is set at build time via -ldflags.
var Version = "dev"

// SystemHandler exposes health and host statistics.
type SystemHandler struct{}

// NewSystemHandler builds a SystemHandler.
func NewSystemHandler() *SystemHandler { return &SystemHandler{} }

// Health is an unauthenticated liveness probe.
func (h *SystemHandler) Health(c *gin.Context) {
	dto.OK(c, http.StatusOK, gin.H{
		"status":  "ok",
		"version": Version,
		"uptime":  int(time.Since(startedAt).Seconds()),
	})
}

// Stats returns basic process/runtime stats for the dashboard. Host-level
// CPU/RAM gauges are added in a later phase via gopsutil.
func (h *SystemHandler) Stats(c *gin.Context) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	dto.OK(c, http.StatusOK, gin.H{
		"version":    Version,
		"uptime":     int(time.Since(startedAt).Seconds()),
		"goroutines": runtime.NumGoroutine(),
		"heap_alloc": m.HeapAlloc,
		"sys_mem":    m.Sys,
		"num_cpu":    runtime.NumCPU(),
	})
}
