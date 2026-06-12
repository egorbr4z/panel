package handler

import (
	"net/http"
	"time"

	"github.com/egorbr4z/panel/internal/api/dto"
	"github.com/egorbr4z/panel/internal/api/middleware"
	"github.com/egorbr4z/panel/internal/core"
	"github.com/egorbr4z/panel/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// UserHandler manages VPN client CRUD and provisioning.
type UserHandler struct {
	db  *gorm.DB
	mgr *core.Manager
}

// NewUserHandler builds a UserHandler.
func NewUserHandler(db *gorm.DB, mgr *core.Manager) *UserHandler {
	return &UserHandler{db: db, mgr: mgr}
}

// UserUpsertRequest is the create/update payload for a VPN user.
type UserUpsertRequest struct {
	Username      string               `json:"username" binding:"required"`
	DataLimit     int64                `json:"data_limit"`
	ExpireAt      *time.Time           `json:"expire_at"`
	ResetStrategy models.ResetStrategy `json:"reset_strategy"`
	Note          string               `json:"note"`
	InboundIDs    []int64              `json:"inbound_ids"`
}

// List returns users (optionally filtered by status/search), scoped to the
// calling admin unless they are sudo.
func (h *UserHandler) List(c *gin.Context) {
	q := h.db.Preload("Inbounds").Model(&models.User{})
	if middleware.Role(c) != models.RoleSudo {
		q = q.Where("admin_id = ?", middleware.AdminID(c))
	}
	if s := c.Query("status"); s != "" {
		q = q.Where("status = ?", s)
	}
	if search := c.Query("search"); search != "" {
		q = q.Where("username LIKE ?", "%"+search+"%")
	}
	var users []models.User
	if err := q.Order("id desc").Find(&users).Error; err != nil {
		dto.Fail(c, http.StatusInternalServerError, "failed to list users")
		return
	}
	dto.OK(c, http.StatusOK, users)
}

// Get returns one user with assigned inbounds.
func (h *UserHandler) Get(c *gin.Context) {
	var u models.User
	if err := h.db.Preload("Inbounds").First(&u, c.Param("id")).Error; err != nil {
		dto.Fail(c, http.StatusNotFound, "user not found")
		return
	}
	dto.OK(c, http.StatusOK, u)
}

// Create provisions a new user, generating credentials and assigning inbounds.
func (h *UserHandler) Create(c *gin.Context) {
	var req UserUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.Fail(c, http.StatusBadRequest, "invalid request body")
		return
	}

	uuid, _ := core.NewUUID()
	password, _ := core.RandomSecret(16)
	token, _ := core.RandomSecret(24)

	u := models.User{
		Username:          req.Username,
		UUID:              uuid,
		Password:          password,
		SubscriptionToken: token,
		Status:            models.StatusActive,
		DataLimit:         req.DataLimit,
		ResetStrategy:     defaultReset(req.ResetStrategy),
		ExpireAt:          req.ExpireAt,
		Note:              req.Note,
		AdminID:           middleware.AdminID(c),
	}
	if err := h.db.Create(&u).Error; err != nil {
		dto.Fail(c, http.StatusConflict, "could not create user (duplicate username?)")
		return
	}
	h.assignInbounds(&u, req.InboundIDs)
	h.mgr.ReconcileSoon()

	h.db.Preload("Inbounds").First(&u, u.ID)
	dto.OK(c, http.StatusCreated, u)
}

// Update edits user limits/expiry/inbound assignment.
func (h *UserHandler) Update(c *gin.Context) {
	var u models.User
	if err := h.db.First(&u, c.Param("id")).Error; err != nil {
		dto.Fail(c, http.StatusNotFound, "user not found")
		return
	}
	var req UserUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.Fail(c, http.StatusBadRequest, "invalid request body")
		return
	}
	u.Username = req.Username
	u.DataLimit = req.DataLimit
	u.ExpireAt = req.ExpireAt
	u.ResetStrategy = defaultReset(req.ResetStrategy)
	u.Note = req.Note
	if err := h.db.Save(&u).Error; err != nil {
		dto.Fail(c, http.StatusInternalServerError, "could not update user")
		return
	}
	if req.InboundIDs != nil {
		h.assignInbounds(&u, req.InboundIDs)
	}
	h.mgr.ReconcileSoon()

	h.db.Preload("Inbounds").First(&u, u.ID)
	dto.OK(c, http.StatusOK, u)
}

// Delete removes a user and reconciles cores.
func (h *UserHandler) Delete(c *gin.Context) {
	if err := h.db.Select("Inbounds").Delete(&models.User{}, c.Param("id")).Error; err != nil {
		dto.Fail(c, http.StatusInternalServerError, "could not delete user")
		return
	}
	h.mgr.ReconcileSoon()
	dto.OK(c, http.StatusOK, gin.H{"ok": true})
}

// SetEnabled toggles a user between active and disabled.
func (h *UserHandler) SetEnabled(enabled bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var u models.User
		if err := h.db.First(&u, c.Param("id")).Error; err != nil {
			dto.Fail(c, http.StatusNotFound, "user not found")
			return
		}
		if enabled {
			u.Status = models.StatusActive
		} else {
			u.Status = models.StatusDisabled
		}
		h.db.Model(&u).Update("status", u.Status)
		h.mgr.ReconcileSoon()
		dto.OK(c, http.StatusOK, u)
	}
}

// ResetTraffic zeroes a user's usage counters.
func (h *UserHandler) ResetTraffic(c *gin.Context) {
	var u models.User
	if err := h.db.First(&u, c.Param("id")).Error; err != nil {
		dto.Fail(c, http.StatusNotFound, "user not found")
		return
	}
	now := time.Now()
	h.db.Model(&u).Updates(map[string]any{"used_up": 0, "used_down": 0, "last_reset_at": now})
	dto.OK(c, http.StatusOK, gin.H{"ok": true})
}

// RevokeSub rotates the subscription token, invalidating old links instantly.
func (h *UserHandler) RevokeSub(c *gin.Context) {
	var u models.User
	if err := h.db.First(&u, c.Param("id")).Error; err != nil {
		dto.Fail(c, http.StatusNotFound, "user not found")
		return
	}
	token, _ := core.RandomSecret(24)
	h.db.Model(&u).Update("subscription_token", token)
	u.SubscriptionToken = token
	dto.OK(c, http.StatusOK, u)
}

func (h *UserHandler) assignInbounds(u *models.User, ids []int64) {
	var inbounds []models.Inbound
	if len(ids) > 0 {
		h.db.Find(&inbounds, ids)
	}
	_ = h.db.Model(u).Association("Inbounds").Replace(inbounds)
}

func defaultReset(s models.ResetStrategy) models.ResetStrategy {
	if s == "" {
		return models.ResetNone
	}
	return s
}
