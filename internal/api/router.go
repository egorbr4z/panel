// Package api wires the HTTP routes and middleware chain together.
package api

import (
	"github.com/egorbr4z/panel/internal/api/handler"
	"github.com/egorbr4z/panel/internal/api/middleware"
	"github.com/egorbr4z/panel/internal/config"
	"github.com/egorbr4z/panel/internal/service/auth"
	"github.com/egorbr4z/panel/internal/web"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// NewRouter builds the configured Gin engine with all routes registered.
func NewRouter(cfg *config.Config, db *gorm.DB, jm *auth.Manager) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowAllOrigins: true,
		AllowMethods:    []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:    []string{"Origin", "Content-Type", "Authorization"},
	}))

	authH := handler.NewAuthHandler(db, jm)
	sysH := handler.NewSystemHandler()

	apiGroup := r.Group("/api")
	{
		apiGroup.GET("/health", sysH.Health)

		authGroup := apiGroup.Group("/auth")
		authGroup.POST("/login", middleware.RateLimit(10), authH.Login)
		authGroup.POST("/refresh", authH.Refresh)

		// Protected routes require a valid access token.
		protected := apiGroup.Group("")
		protected.Use(middleware.JWT(jm))
		{
			protected.POST("/auth/logout", authH.Logout)
			protected.GET("/auth/me", authH.Me)
			protected.GET("/system", sysH.Stats)
		}
	}

	// SPA (static, embedded) + SPA fallback for client routing.
	web.Register(r)

	return r
}
