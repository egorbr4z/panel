package auth

import (
	"errors"
	"time"

	"github.com/egorbr4z/panel/internal/models"
	"github.com/golang-jwt/jwt/v5"
)

// TokenType distinguishes access tokens from refresh tokens.
type TokenType string

const (
	AccessToken  TokenType = "access"
	RefreshToken TokenType = "refresh"
)

// Claims is the JWT payload for admin sessions.
type Claims struct {
	AdminID int64            `json:"sub_id"`
	Role    models.AdminRole `json:"role"`
	Type    TokenType        `json:"typ"`
	jwt.RegisteredClaims
}

// Manager issues and validates JWTs using an HMAC secret.
type Manager struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// NewManager builds a JWT manager.
func NewManager(secret string, accessTTL, refreshTTL time.Duration) *Manager {
	return &Manager{secret: []byte(secret), accessTTL: accessTTL, refreshTTL: refreshTTL}
}

// Issue creates a signed token of the given type for an admin.
func (m *Manager) Issue(admin *models.Admin, typ TokenType) (string, error) {
	ttl := m.accessTTL
	if typ == RefreshToken {
		ttl = m.refreshTTL
	}
	now := time.Now()
	claims := Claims{
		AdminID: admin.ID,
		Role:    admin.Role,
		Type:    typ,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   admin.Username,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
}

// Parse validates a token string and returns its claims.
func (m *Manager) Parse(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
