// Package web embeds the built React SPA and serves it (with SPA fallback) from
// the single Go binary. During development, before the frontend is built, the
// embedded dist contains only a placeholder index.html.
package web

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed all:dist
var distFS embed.FS

// Register mounts the SPA on the router. Unknown non-API paths fall back to
// index.html so client-side routing works.
func Register(r *gin.Engine) {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic(err)
	}
	fileServer := http.FileServer(http.FS(sub))

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		// Never let API or subscription routes fall through to the SPA.
		if strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/sub/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		// Serve the asset if it exists, else index.html (SPA fallback).
		if _, err := fs.Stat(sub, strings.TrimPrefix(path, "/")); err != nil && path != "/" {
			c.Request.URL.Path = "/"
		}
		fileServer.ServeHTTP(c.Writer, c.Request)
	})
}
