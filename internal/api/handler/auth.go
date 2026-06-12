// Package handler implements the HTTP handlers grouped by domain.
package handler

import (
	"net/http"
	"time"

	"github.com/egorbr4z/panel/internal/api/dto"
	"github.com/egorbr4z/panel/internal/api/middleware"
	"github.com/egorbr4z/panel/internal/models"
	"github.com/egorbr4z/panel/internal/service/auth"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AuthHandler handles login, refresh, logout and the current-admin endpoint.
type AuthHandler struct {
	db *gorm.DB
	jm *auth.Manager
}

// NewAuthHandler builds an AuthHandler.
func NewAuthHandler(db *gorm.DB, jm *auth.Manager) *AuthHandler {
	return &AuthHandler{db: db, jm: jm}
}

// Login validates credentials and issues access + refresh tokens.
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.Fail(c, http.StatusBadRequest, "invalid request body")
		return
	}

	var admin models.Admin
	if err := h.db.Where("username = ?", req.Username).First(&admin).Error; err != nil {
		dto.Fail(c, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if !admin.IsActive {
		dto.Fail(c, http.StatusForbidden, "account disabled")
		return
	}
	ok, err := auth.VerifyPassword(req.Password, admin.PasswordHash)
	if err != nil || !ok {
		dto.Fail(c, http.StatusUnauthorized, "invalid credentials")
		return
	}

	now := time.Now()
	admin.LastLoginAt = &now
	h.db.Model(&admin).Update("last_login_at", now)

	h.issueTokens(c, &admin)
}

// Refresh exchanges a valid refresh token for a new token pair.
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req dto.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.Fail(c, http.StatusBadRequest, "invalid request body")
		return
	}
	claims, err := h.jm.Parse(req.RefreshToken)
	if err != nil || claims.Type != auth.RefreshToken {
		dto.Fail(c, http.StatusUnauthorized, "invalid refresh token")
		return
	}
	var admin models.Admin
	if err := h.db.First(&admin, claims.AdminID).Error; err != nil || !admin.IsActive {
		dto.Fail(c, http.StatusUnauthorized, "account unavailable")
		return
	}
	h.issueTokens(c, &admin)
}

// Logout is a no-op server-side (stateless JWT); clients drop their tokens.
func (h *AuthHandler) Logout(c *gin.Context) {
	dto.OK(c, http.StatusOK, gin.H{"ok": true})
}

// Me returns the authenticated admin.
func (h *AuthHandler) Me(c *gin.Context) {
	var admin models.Admin
	if err := h.db.First(&admin, middleware.AdminID(c)).Error; err != nil {
		dto.Fail(c, http.StatusNotFound, "admin not found")
		return
	}
	dto.OK(c, http.StatusOK, admin)
}

func (h *AuthHandler) issueTokens(c *gin.Context, admin *models.Admin) {
	access, err := h.jm.Issue(admin, auth.AccessToken)
	if err != nil {
		dto.Fail(c, http.StatusInternalServerError, "failed to issue token")
		return
	}
	refresh, err := h.jm.Issue(admin, auth.RefreshToken)
	if err != nil {
		dto.Fail(c, http.StatusInternalServerError, "failed to issue token")
		return
	}
	dto.OK(c, http.StatusOK, dto.TokenResponse{
		AccessToken:  access,
		RefreshToken: refresh,
		TokenType:    "Bearer",
		ExpiresIn:    900,
	})
}
