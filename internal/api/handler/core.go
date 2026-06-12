package handler

import (
	"net/http"

	"github.com/egorbr4z/panel/internal/api/dto"
	"github.com/egorbr4z/panel/internal/core"
	"github.com/gin-gonic/gin"
)

// CoreHandler exposes proxy-core status, logs, restart and config preview.
type CoreHandler struct {
	mgr *core.Manager
}

// NewCoreHandler builds a CoreHandler.
func NewCoreHandler(mgr *core.Manager) *CoreHandler {
	return &CoreHandler{mgr: mgr}
}

// List returns the status of every core.
func (h *CoreHandler) List(c *gin.Context) {
	dto.OK(c, http.StatusOK, h.mgr.Status())
}

// Restart restarts the named core.
func (h *CoreHandler) Restart(c *gin.Context) {
	if !h.mgr.Restart(c.Param("name")) {
		dto.Fail(c, http.StatusNotFound, "unknown core")
		return
	}
	dto.OK(c, http.StatusOK, gin.H{"ok": true})
}

// Logs returns recent log lines for the named core.
func (h *CoreHandler) Logs(c *gin.Context) {
	logs, ok := h.mgr.Logs(c.Param("name"))
	if !ok {
		dto.Fail(c, http.StatusNotFound, "unknown core")
		return
	}
	dto.OK(c, http.StatusOK, gin.H{"lines": logs})
}

// Config returns the current generated config for the named core.
func (h *CoreHandler) Config(c *gin.Context) {
	data, ok, err := h.mgr.ConfigPreview(c.Param("name"))
	if err != nil {
		dto.Fail(c, http.StatusInternalServerError, "failed to render config")
		return
	}
	if !ok {
		dto.Fail(c, http.StatusNotFound, "unknown core")
		return
	}
	c.Data(http.StatusOK, "application/json", data)
}
