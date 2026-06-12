// Package middleware holds the Gin middleware chain: auth, role enforcement
// and a lightweight in-memory rate limiter for sensitive endpoints.
package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/egorbr4z/panel/internal/api/dto"
	"github.com/egorbr4z/panel/internal/models"
	"github.com/egorbr4z/panel/internal/service/auth"
	"github.com/gin-gonic/gin"
)

const (
	ctxAdminID = "admin_id"
	ctxRole    = "admin_role"
)

// JWT validates the Bearer access token and stashes admin id/role in context.
func JWT(jm *auth.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			dto.Fail(c, http.StatusUnauthorized, "missing bearer token")
			return
		}
		claims, err := jm.Parse(strings.TrimPrefix(header, "Bearer "))
		if err != nil || claims.Type != auth.AccessToken {
			dto.Fail(c, http.StatusUnauthorized, "invalid or expired token")
			return
		}
		c.Set(ctxAdminID, claims.AdminID)
		c.Set(ctxRole, claims.Role)
		c.Next()
	}
}

// RequireSudo aborts unless the authenticated admin has the sudo role.
func RequireSudo() gin.HandlerFunc {
	return func(c *gin.Context) {
		if Role(c) != models.RoleSudo {
			dto.Fail(c, http.StatusForbidden, "sudo role required")
			return
		}
		c.Next()
	}
}

// AdminID returns the authenticated admin id from context (0 if absent).
func AdminID(c *gin.Context) int64 {
	if v, ok := c.Get(ctxAdminID); ok {
		if id, ok := v.(int64); ok {
			return id
		}
	}
	return 0
}

// Role returns the authenticated admin role from context.
func Role(c *gin.Context) models.AdminRole {
	if v, ok := c.Get(ctxRole); ok {
		if r, ok := v.(models.AdminRole); ok {
			return r
		}
	}
	return ""
}

// RateLimit applies a simple fixed-window per-IP limit. Cheap and good enough
// to slow brute-force on login/subscription endpoints without pulling in Redis.
func RateLimit(maxPerMinute int) gin.HandlerFunc {
	type bucket struct {
		count int
		reset time.Time
	}
	var mu sync.Mutex
	buckets := map[string]*bucket{}

	return func(c *gin.Context) {
		ip := c.ClientIP()
		now := time.Now()

		mu.Lock()
		b, ok := buckets[ip]
		if !ok || now.After(b.reset) {
			b = &bucket{reset: now.Add(time.Minute)}
			buckets[ip] = b
		}
		b.count++
		over := b.count > maxPerMinute
		mu.Unlock()

		if over {
			dto.Fail(c, http.StatusTooManyRequests, "rate limit exceeded")
			return
		}
		c.Next()
	}
}
