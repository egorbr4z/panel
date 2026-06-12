// Package dto holds request/response shapes and the consistent JSON envelope
// used across the API.
package dto

import "github.com/gin-gonic/gin"

// Envelope is the uniform response wrapper: {data, error, meta}.
type Envelope struct {
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
	Meta  any    `json:"meta,omitempty"`
}

// OK writes a success envelope.
func OK(c *gin.Context, status int, data any) {
	c.JSON(status, Envelope{Data: data})
}

// OKMeta writes a success envelope with pagination/meta info.
func OKMeta(c *gin.Context, status int, data, meta any) {
	c.JSON(status, Envelope{Data: data, Meta: meta})
}

// Fail writes an error envelope and aborts.
func Fail(c *gin.Context, status int, msg string) {
	c.AbortWithStatusJSON(status, Envelope{Error: msg})
}

// LoginRequest is the admin login payload.
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// TokenResponse carries issued tokens.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"` // seconds
}

// RefreshRequest carries a refresh token to exchange.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}
