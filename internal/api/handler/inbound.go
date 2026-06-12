package handler

import (
	"net/http"

	"github.com/egorbr4z/panel/internal/api/dto"
	"github.com/egorbr4z/panel/internal/core"
	"github.com/egorbr4z/panel/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// InboundHandler manages inbound CRUD and core helpers.
type InboundHandler struct {
	db  *gorm.DB
	mgr *core.Manager
}

// NewInboundHandler builds an InboundHandler.
func NewInboundHandler(db *gorm.DB, mgr *core.Manager) *InboundHandler {
	return &InboundHandler{db: db, mgr: mgr}
}

// List returns all inbounds with their hosts.
func (h *InboundHandler) List(c *gin.Context) {
	var inbounds []models.Inbound
	if err := h.db.Preload("Hosts").Order("sort_order asc").Find(&inbounds).Error; err != nil {
		dto.Fail(c, http.StatusInternalServerError, "failed to list inbounds")
		return
	}
	dto.OK(c, http.StatusOK, inbounds)
}

// Get returns a single inbound.
func (h *InboundHandler) Get(c *gin.Context) {
	var ib models.Inbound
	if err := h.db.Preload("Hosts").First(&ib, c.Param("id")).Error; err != nil {
		dto.Fail(c, http.StatusNotFound, "inbound not found")
		return
	}
	dto.OK(c, http.StatusOK, ib)
}

// Create adds a new inbound and reconciles the affected core.
func (h *InboundHandler) Create(c *gin.Context) {
	var ib models.Inbound
	if err := c.ShouldBindJSON(&ib); err != nil {
		dto.Fail(c, http.StatusBadRequest, "invalid request body")
		return
	}
	if !validInbound(c, &ib) {
		return
	}
	if err := h.db.Create(&ib).Error; err != nil {
		dto.Fail(c, http.StatusConflict, "could not create inbound (duplicate tag/port?)")
		return
	}
	h.mgr.ReconcileSoon()
	dto.OK(c, http.StatusCreated, ib)
}

// Update edits an existing inbound and reconciles.
func (h *InboundHandler) Update(c *gin.Context) {
	var existing models.Inbound
	if err := h.db.First(&existing, c.Param("id")).Error; err != nil {
		dto.Fail(c, http.StatusNotFound, "inbound not found")
		return
	}
	var in models.Inbound
	if err := c.ShouldBindJSON(&in); err != nil {
		dto.Fail(c, http.StatusBadRequest, "invalid request body")
		return
	}
	in.ID = existing.ID
	if !validInbound(c, &in) {
		return
	}
	if err := h.db.Save(&in).Error; err != nil {
		dto.Fail(c, http.StatusInternalServerError, "could not update inbound")
		return
	}
	h.mgr.ReconcileSoon()
	dto.OK(c, http.StatusOK, in)
}

// Delete removes an inbound and reconciles.
func (h *InboundHandler) Delete(c *gin.Context) {
	if err := h.db.Delete(&models.Inbound{}, c.Param("id")).Error; err != nil {
		dto.Fail(c, http.StatusInternalServerError, "could not delete inbound")
		return
	}
	h.mgr.ReconcileSoon()
	dto.OK(c, http.StatusOK, gin.H{"ok": true})
}

// Toggle enables/disables an inbound and reconciles.
func (h *InboundHandler) Toggle(c *gin.Context) {
	var ib models.Inbound
	if err := h.db.First(&ib, c.Param("id")).Error; err != nil {
		dto.Fail(c, http.StatusNotFound, "inbound not found")
		return
	}
	ib.Enabled = !ib.Enabled
	h.db.Model(&ib).Update("enabled", ib.Enabled)
	h.mgr.ReconcileSoon()
	dto.OK(c, http.StatusOK, ib)
}

// RealityKeys generates a fresh X25519 keypair + short id for VLESS+Reality.
func (h *InboundHandler) RealityKeys(c *gin.Context) {
	kp, err := core.GenerateRealityKeypair()
	if err != nil {
		dto.Fail(c, http.StatusInternalServerError, "key generation failed")
		return
	}
	dto.OK(c, http.StatusOK, kp)
}

// validInbound performs minimal validation and writes an error response if invalid.
func validInbound(c *gin.Context, ib *models.Inbound) bool {
	if ib.Tag == "" || ib.Protocol == "" || ib.Port == 0 {
		dto.Fail(c, http.StatusBadRequest, "tag, protocol and port are required")
		return false
	}
	if ib.Core != models.CoreXray && ib.Core != models.CoreSingbox {
		dto.Fail(c, http.StatusBadRequest, "core must be 'xray' or 'singbox'")
		return false
	}
	if ib.Core == models.CoreSingbox && !core.SingboxSupports(ib.Protocol) {
		dto.Fail(c, http.StatusBadRequest, "unsupported sing-box protocol: "+ib.Protocol)
		return false
	}
	return true
}
